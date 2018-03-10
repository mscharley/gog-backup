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

type Download struct {
	Name    string
	URL     string
	File    string
	Version string
}

func main() {
	iniflags.Parse()

	client := &gog.Client{
		Client:       http.DefaultClient,
		RefreshToken: *refreshToken,
	}
	err := client.RefreshAccess()
	if err != nil {
		log.Fatalf("login error: %+v", err)
	}

	finished := make(chan bool)
	gameInfo := make(chan int64)
	gameDownload := make(chan *Download)
	extraDownload := make(chan *Download)

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
		fmt.Printf("Fetching details for %d\n", id)
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
				if len(game.Downloads) > 0 {
					download := game.Downloads[0]
					for _, d := range download.Platforms.Windows {
						gameDownload <- &Download{
							Name:    fmt.Sprintf("%s [Windows] [%s]", d.Name, d.Size),
							URL:     gog.EmbedEndpoint + d.ManualDownloadURL,
							File:    path + "/Windows",
							Version: d.Version,
						}
					}
					for _, d := range download.Platforms.Mac {
						gameDownload <- &Download{
							Name:    fmt.Sprintf("%s [Mac] [%s]", d.Name, d.Size),
							URL:     gog.EmbedEndpoint + d.ManualDownloadURL,
							File:    path + "/Mac",
							Version: d.Version,
						}
					}
					for _, d := range download.Platforms.Linux {
						gameDownload <- &Download{
							Name:    fmt.Sprintf("%s [Linux] [%s]", d.Name, d.Size),
							URL:     gog.EmbedEndpoint + d.ManualDownloadURL,
							File:    path + "/Linux",
							Version: d.Version,
						}
					}
				}

				for _, extra := range game.Extras {
					extraDownload <- &Download{
						Name:    fmt.Sprintf("Extra for %s: %s [%s]", game.Title, extra.Name, extra.Size),
						URL:     gog.EmbedEndpoint + extra.ManualDownloadURL,
						File:    path + "/Extras",
						Version: extra.Version,
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
		fmt.Printf("%s (version: %s)\n  %s -> %s\n", d.Name, d.Version, d.URL, path)

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

	writer, err := os.OpenFile(path+"/"+filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)

	return err
}

func signalHandler(finished chan<- bool) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	for {
		signal := <-c
		finished <- true
		log.Printf("Received a %s signal, finishing downloads before closing.\n", signal)
	}
}
