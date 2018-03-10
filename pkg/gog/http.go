package gog

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func (client *Client) authenticatedGet(URL string, result interface{}) error {
	_, body, err := client.DownloadFile(URL)
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

// DownloadFile initiates a download of a file from GoG and returns a filename and ReadCloser
// to control the download.
func (client *Client) DownloadFile(URL string) (string, io.ReadCloser, error) {
	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return "", nil, err
	}
	request.Header.Add("Authorization", "Bearer "+*client.accessToken)
	response, err := client.Do(request)
	if err != nil {
		return "", nil, err
	}
	if response.StatusCode/100 != 2 {
		return "", nil, fmt.Errorf("Unexpected status code: %d", response.StatusCode)
	}

	segments := strings.Split(response.Request.URL.Path, "/")

	return segments[len(segments)-1], response.Body, nil
}
