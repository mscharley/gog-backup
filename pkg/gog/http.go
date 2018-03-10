package gog

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

func (c *Client) authenticatedGet(url string, result interface{}) error {
	body, err := c.DownloadFile(url)
	if err != nil {
		return err
	}
	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &result)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DownloadFile(url string) (io.ReadCloser, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "Bearer "+*c.accessToken)
	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Unexpected status code: %d", response.StatusCode)
	}

	return response.Body, nil
}
