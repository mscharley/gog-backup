package gog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
}

// MediaType is an enumeration to pick between different supported types of media in GoG.
type MediaType int8

const (
	// GameMediaType is the MediaType that represents games.
	GameMediaType MediaType = iota + 1
	// MovieMediaType is the MediaType that represents movies.
	MovieMediaType
)

func (c *Client) refreshAccess() error {
	if c.tokenExpiry-time.Now().Unix() > 60 {
		return nil
	}
	response, err := c.Get(AuthEndpoint + "/token?client_id=" + clientID + "&client_secret=" + clientSecret + "&grant_type=refresh_token&refresh_token=" + c.RefreshToken)
	if err != nil {
		return err
	}
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("Unexpected status code: %d", response.StatusCode)
	}

	var login = new(refreshTokenResponse)
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &login)
	if err != nil {
		return err
	}

	c.tokenExpiry = time.Now().Unix() + int64(login.ExpiresIn)
	c.accessToken = &login.AccessToken
	return nil
}

func (c *Client) GameList() ([]int64, error) {
	c.refreshAccess()
	var result = new(gameList)
	err := c.authenticatedGet(EmbedEndpoint+"/user/data/games", result)
	if err != nil {
		return nil, err
	}

	return result.Owned, nil
}

func (c *Client) GetFilteredProducts(mediaType MediaType, page int) (*FilteredProductPage, error) {
	c.refreshAccess()
	var result = new(FilteredProductPage)
	err := c.authenticatedGet(fmt.Sprintf("%s/account/getFilteredProducts?mediaType=%d&page=%d", EmbedEndpoint, mediaType, page), result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) GameDetails(id int64) (*GameDetails, error) {
	c.refreshAccess()
	var result = new(GameDetails)
	err := c.authenticatedGet(fmt.Sprintf("%s/account/gameDetails/%d.json", EmbedEndpoint, id), result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
