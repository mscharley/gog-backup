package gog

import (
	"encoding/json"
	"fmt"
)

type refreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	UserID      string `json:"user_id"`
}

type gameList struct {
	Owned []int64 `json:"owned"`
}

type GameDetails struct {
	Title     string           `json:"title"`
	CDKey     string           `json:"cd_key"`
	Downloads []*GameLanguages `json:"downloads"`
	Extras    []*GameDownload  `json:"extras"`
	DLCs      []*GameDetails   `json:"dlcs"`
	Tags      []*GameTag       `json:"tags"`
}

type GameLanguages struct {
	Language  string
	Platforms *GamePlatforms
}

func (this *GameLanguages) UnmarshalJSON(b []byte) error {
	var languages []json.RawMessage
	err := json.Unmarshal(b, &languages)
	if err != nil {
		return err
	}
	if len(languages) != 2 {
		return fmt.Errorf("Expected an array of length 2 but got %d", len(languages))
	}
	err = json.Unmarshal(languages[0], &this.Language)
	if err != nil {
		return err
	}
	err = json.Unmarshal(languages[1], &this.Platforms)
	if err != nil {
		return err
	}
	return nil
}

type GamePlatforms struct {
	Windows []*GameDownload `json:"windows"`
	Mac     []*GameDownload `json:"mac"`
	Linux   []*GameDownload `json:"linux"`
}

type GameDownload struct {
	ManualDownloadURL string `json:"manualUrl"`
	DownloaderURL     string `json:"downloaderUrl"`
	Name              string `json:"name"`
	Version           string `json:"version"`
	Type              string `json:"type"`
	Info              int    `json:"info"`
	// This is a textual representation of the size of the download, eg. "6MB"
	Size string `json:"Size"`
}

type GameTag struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProductCount string `json:"productCount"`
}

type FilteredProductPage struct {
	Page            int `json:"page"`
	TotalProducts   int `json:"totalProducts"`
	TotalPages      int `json:"totalPages"`
	ProductsPerPage int `json:"productsPerPage"`
	Products        []FilteredProduct
}

type FilteredProduct struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}
