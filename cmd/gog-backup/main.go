package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

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
	Name string
	URL  string
	File string
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

	gameInfo := make(chan int64)
	gameDownload := make(chan *Download, 2)
	extraDownload := make(chan *Download, 10)

	go generateGames(gameInfo, client)
	go fetchDetails(gameInfo, gameDownload, extraDownload, client)

	waitGroup.Add(4)
	go download(gameDownload, client)
	go download(gameDownload, client)
	go download(extraDownload, client)
	go download(extraDownload, client)
	waitGroup.Wait()
}

func generateGames(games chan<- int64, client *gog.Client) {
	page := 0
	totalPages := 1
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
			close(games)
			return
		}

		totalPages = result.TotalPages
		for _, product := range result.Products {
			games <- product.ID
		}
	}
	close(games)
}

func safePath(path string) string {
	return strings.Replace(
		strings.Replace(path, "/", "", -1),
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
							Name: fmt.Sprintf("%s [Windows] [%s]", d.Name, d.Size),
							URL:  gog.EmbedEndpoint + d.ManualDownloadURL,
							File: path + "/Windows",
						}
					}
					for _, d := range download.Platforms.Mac {
						gameDownload <- &Download{
							Name: fmt.Sprintf("%s [Mac] [%s]", d.Name, d.Size),
							URL:  gog.EmbedEndpoint + d.ManualDownloadURL,
							File: path + "/Mac",
						}
					}
					for _, d := range download.Platforms.Linux {
						gameDownload <- &Download{
							Name: fmt.Sprintf("%s [Linux] [%s]", d.Name, d.Size),
							URL:  gog.EmbedEndpoint + d.ManualDownloadURL,
							File: path + "/Linux",
						}
					}
				}

				for _, extra := range game.Extras {
					extraDownload <- &Download{
						Name: fmt.Sprintf("Extra for %s: %s [%s]", game.Title, extra.Name, extra.Size),
						URL:  gog.EmbedEndpoint + extra.ManualDownloadURL,
						File: path + "/Extras",
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
		fmt.Printf("%s\n  %s -> %s\n", d.Name, d.URL, path)

		filename, reader, err := client.DownloadFile(d.URL)
		if err != nil {
			log.Printf("Unable to connect to GoG: %+v", err)
			continue
		}

		err = downloadFile(reader, path, filename)
		if err != nil {
			log.Printf("Unable to download file: %+v", err)
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
