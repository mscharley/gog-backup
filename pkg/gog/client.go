package gog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// AuthEndpoint is the base URL for authentication to GoG.com
const AuthEndpoint = "https://auth.gog.com"

// EmbedEndpoint is the base URL for the embed API.
const EmbedEndpoint = "https://embed.gog.com"

// These are 'borrowed' from the Galaxy Client.
// See also: https://gogapidocs.readthedocs.io/en/latest/auth.html
const clientID = "46899977096215655"
const clientSecret = "9d85c43b1482497dbbce61f6e4aa173a433796eeae2ca8c5f6129f2dc4de46d9"

// Client is a a public class for accessing the GoG.com API.
type Client struct {
	*http.Client
	RefreshToken string
	accessToken  *string
	tokenExpiry  int64
	lock         sync.Mutex
}

// MediaType is an enumeration to pick between different supported types of media in GoG.
type MediaType int8

const (
	// GameMediaType is the MediaType that represents games.
	GameMediaType MediaType = iota + 1
	// MovieMediaType is the MediaType that represents movies.
	MovieMediaType
)

func (client *Client) refreshAccess() error {
	client.lock.Lock()
	defer client.lock.Unlock()
	if client.tokenExpiry-time.Now().Unix() > 60 {
		return nil
	}
	fmt.Println("Re-generating the access token for GoG.")
	response, err := client.Get(AuthEndpoint + "/token?client_id=" + clientID + "&client_secret=" + clientSecret + "&grant_type=refresh_token&refresh_token=" + client.RefreshToken)
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("Unexpected status code: %d\n%s", response.StatusCode, buf)
	}
	var login = new(refreshTokenResponse)
	err = json.Unmarshal(buf, &login)
	if err != nil {
		return err
	}

	client.tokenExpiry = time.Now().Unix() + int64(login.ExpiresIn)
	client.accessToken = &login.AccessToken
	return nil
}

// GameList retrieves a list of id's for all games the current user owns.
//
// Many of the ID's returned don't seem to actually be games or are otherwise non-existent entities.
//
// See also GetFilteredProducts()
// See also https://gogapidocs.readthedocs.io/en/latest/account.html#get--user-data-games
func (client *Client) GameList() ([]int64, error) {
	if err := client.refreshAccess(); err != nil {
		return nil, err
	}
	var result = new(gameList)
	err := client.authenticatedGet(EmbedEndpoint+"/user/data/games", result)
	if err != nil {
		return nil, err
	}

	return result.Owned, nil
}

// GetFilteredProducts returns paginated search results for games or movies purchased by the current user.
func (client *Client) GetFilteredProducts(mediaType MediaType, page int) (*FilteredProductPage, error) {
	if err := client.refreshAccess(); err != nil {
		return nil, err
	}
	var result = new(FilteredProductPage)
	err := client.authenticatedGet(fmt.Sprintf("%s/account/getFilteredProducts?mediaType=%d&page=%d", EmbedEndpoint, mediaType, page), result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GameDetails returns detailed information about a single game.
func (client *Client) GameDetails(id int64) (*GameDetails, error) {
	if err := client.refreshAccess(); err != nil {
		return nil, err
	}
	var result = new(GameDetails)
	err := client.authenticatedGet(fmt.Sprintf("%s/account/gameDetails/%d.json", EmbedEndpoint, id), result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
