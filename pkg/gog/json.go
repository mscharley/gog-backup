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

// GameDetails is a detailed set of content about a single game.
type GameDetails struct {
	Title     string           `json:"title"`
	CDKey     string           `json:"cd_key"`
	Downloads []*GameLanguages `json:"downloads"`
	Extras    []*GameDownload  `json:"extras"`
	DLCs      []*GameDetails   `json:"dlcs"`
	Tags      []*GameTag       `json:"tags"`
}

// GameLanguages descibes the downloadable files per language and platform.
type GameLanguages struct {
	Language  string
	Platforms *GamePlatforms
}

// UnmarshalJSON is used by the JSON marshaller to generate GameLanguages structs from JSON tuples.
//
// The structure that GoG uses for these fields is a tuple where the first element is a string that describes
// the language. The second object in the tuple is an object describing the platforms supported by that language.
func (gl *GameLanguages) UnmarshalJSON(b []byte) error {
	var languages []json.RawMessage
	err := json.Unmarshal(b, &languages)
	if err != nil {
		return err
	}
	if len(languages) != 2 {
		return fmt.Errorf("Expected an array of length 2 but got %d", len(languages))
	}
	err = json.Unmarshal(languages[0], &gl.Language)
	if err != nil {
		return err
	}
	err = json.Unmarshal(languages[1], &gl.Platforms)
	if err != nil {
		return err
	}
	return nil
}

// GamePlatforms is a struct that describes the various platforms that a game may support.
//
// Each platform could have zero or more downloads associated with it.
type GamePlatforms struct {
	Windows []*GameDownload `json:"windows"`
	Mac     []*GameDownload `json:"mac"`
	Linux   []*GameDownload `json:"linux"`
}

// GameDownload represents a single downloadable file.
//
// This may be an installer, part of an installer (Windows installers are broken up into 4GB chunks) or in the case
// of extras it may be something else entirely.
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

// GameTag is a structure that repesents a tag that can be assigned to games.
type GameTag struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProductCount string `json:"productCount"`
}

// FilteredProductPage is a single page of results returned by the GetFilteredProducts() endpoint.
type FilteredProductPage struct {
	Page            int `json:"page"`
	TotalProducts   int `json:"totalProducts"`
	TotalPages      int `json:"totalPages"`
	ProductsPerPage int `json:"productsPerPage"`
	Products        []FilteredProduct
}

// FilteredProduct is a single result returned by the GetFilteredProducts() endpoint.
type FilteredProduct struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}
