package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bclicn/color"
	"github.com/mscharley/gog-backup/pkg/gog"
	"github.com/vharitonsky/iniflags"
)

var (
	waitGroup = new(sync.WaitGroup)
)

var (
	refreshToken = flag.String("refreshToken", "", "A refresh token for the GoG API.")
	targetDir    = flag.String("targetDir", os.Getenv("HOME")+"/GoG", "The target directory to download to.")
)

// Download is a struct used to store details about a single download that needs to be processed. This is the data format used over the
// internal channels.
type Download struct {
	Name    string
	URL     string
	File    string
	Version string
}

func main() {
	iniflags.Parse()

	if *refreshToken == "" {
		log.Fatalln("You must provide a refresh token for GoG.com via -refreshToken.")
	}

	client := &gog.Client{
		Client:       http.DefaultClient,
		RefreshToken: *refreshToken,
	}

	finished := make(chan bool)
	gameInfo := make(chan int64)
	gameDownload := make(chan *Download)
	extraDownload := make(chan *Download, 10)

	go signalHandler(finished)
	go generateGames(gameInfo, finished, client)
	go fetchDetails(gameInfo, gameDownload, extraDownload, client)

	waitGroup.Add(4)
	go download(gameDownload, client)
	go download(gameDownload, client)
	go download(extraDownload, client)
	go download(extraDownload, client)
	waitGroup.Wait()
}

func generateGames(games chan<- int64, finished <-chan bool, client *gog.Client) {
	page := 0
	totalPages := 1
	defer close(games)
	for page < totalPages {
		page++
		if page == 1 {
			log.Printf("Fetching page %d\n", page)
		} else {
			log.Printf("Fetching page %d/%d\n", page, totalPages)
		}
		result, err := client.GetFilteredProducts(gog.GameMediaType, page)
		if err != nil {
			log.Printf("error: %+v", err)
			return
		}

		totalPages = result.TotalPages
		for _, product := range result.Products {
			select {
			case games <- product.ID:
				// Do nothing, keep looping.
			case _ = <-finished:
				return
			}
		}
	}
}

func safePath(path string) string {
	return strings.Replace(
		strings.Replace(strings.TrimSpace(path), "/", "", -1),
		":", " -", -1)
}

func fetchDetails(games <-chan int64, gameDownload chan<- *Download, extraDownload chan<- *Download, client *gog.Client) {
	for id := range games {
		log.Printf("Fetching details for %d", id)
		result, err := client.GameDetails(id)
		if err != nil {
			log.Printf("Unable for fetch details for %d: %+v", id, err)
		} else {
			var games []struct {
				Path    string
				Details *gog.GameDetails
			}
			games = append(games, struct {
				Path    string
				Details *gog.GameDetails
			}{"/" + safePath(result.Title), result})
			for i := 0; i < len(games); i++ {
				path := games[i].Path
				game := games[i].Details

				for _, extra := range game.Extras {
					extraDownload <- &Download{
						Name:    fmt.Sprintf("%s %s", color.LightPurple("Extra for "+game.Title+": "+extra.Name), color.LightYellow("["+extra.Size+"]")),
						URL:     gog.EmbedEndpoint + extra.ManualDownloadURL,
						File:    path + "/Extras",
						Version: extra.Version,
					}
				}

				if len(game.Downloads) > 0 {
					download := game.Downloads[0]
					for _, d := range download.Platforms.Windows {
						gameDownload <- &Download{
							Name:    fmt.Sprintf("%s %s %s", color.LightPurple(d.Name), color.Red("[Windows]"), color.LightYellow("["+d.Size+"]")),
							URL:     gog.EmbedEndpoint + d.ManualDownloadURL,
							File:    path + "/Windows",
							Version: d.Version,
						}
					}
					for _, d := range download.Platforms.Mac {
						gameDownload <- &Download{
							Name:    fmt.Sprintf("%s %s %s", color.LightPurple(d.Name), color.Red("[Mac]"), color.LightYellow("["+d.Size+"]")),
							URL:     gog.EmbedEndpoint + d.ManualDownloadURL,
							File:    path + "/Mac",
							Version: d.Version,
						}
					}
					for _, d := range download.Platforms.Linux {
						gameDownload <- &Download{
							Name:    fmt.Sprintf("%s %s %s", color.LightPurple(d.Name), color.Red("[Linux]"), color.LightYellow("["+d.Size+"]")),
							URL:     gog.EmbedEndpoint + d.ManualDownloadURL,
							File:    path + "/Linux",
							Version: d.Version,
						}
					}
				}

				for _, dlc := range game.DLCs {
					games = append(games, struct {
						Path    string
						Details *gog.GameDetails
					}{path + "/" + safePath(dlc.Title), dlc})
				}
			}
		}
	}

	close(gameDownload)
	close(extraDownload)
}

func download(downloads <-chan *Download, client *gog.Client) {
	for d := range downloads {
		path := *targetDir + d.File

		filename, reader, err := client.DownloadFile(d.URL)
		if err != nil {
			log.Printf("Unable to connect to GoG: %+v", err)
			continue
		}

		// Check for version information from last time.
		versionFile := path + "/." + filename + ".version"
		if d.Version != "" {
			if lastVersion, _ := ioutil.ReadFile(versionFile); string(lastVersion) == d.Version {
				fmt.Printf("Skipping %s as it is already up to date.\n", d.Name)
				reader.Close()
				continue
			}
		} else if info, _ := os.Stat(path + "/" + filename); info != nil {
			fmt.Printf("Skipping %s as it is already downloaded.\n", d.Name)
			reader.Close()
			continue
		}

		version := ""
		if d.Version != "" {
			version = " (version: " + color.Purple(d.Version) + ")"
		}
		fmt.Printf("%s%s\n  %s -> %s\n", d.Name, version, color.LightBlue(d.URL), color.Green(path))
		err = downloadFile(reader, path, filename)
		if err != nil {
			log.Printf("Unable to download file: %+v", err)
			continue
		}

		if d.Version != "" {
			// Save version information for next time.
			err = ioutil.WriteFile(versionFile, []byte(d.Version), 0666)
			if err != nil {
				log.Printf("Unable to save version file: %+v", err)
				continue
			}
		}
	}

	waitGroup.Done()
}

func downloadFile(reader io.ReadCloser, path string, filename string) error {
	defer reader.Close()
	if filename == "" {
		return fmt.Errorf("No filename available, skipping this file")
	}

	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	tmpfile := path + "/." + filename + ".tmp"
	outfile := path + "/" + filename
	writer, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}

	os.Rename(tmpfile, outfile)
	return err
}

func signalHandler(finished chan<- bool) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	signal := <-c
	finished <- true
	close(finished)
	log.Printf("Received a %s signal, finishing downloads before closing.", signal)
	timeout := time.After(time.Second * 60)
	select {
	case signal = <-c:
		log.Fatalf("Received a second %s signal, closing down without cleanup.", signal)
	case _ = <-timeout:
		log.Fatalln("Closing after waiting 60 seconds.")
	}
}
