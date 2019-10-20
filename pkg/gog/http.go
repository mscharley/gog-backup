package gog

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func (client *Client) authenticatedGet(URL string, result interface{}) error {
	_, body, _, err := client.DownloadFile(URL)
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
func (client *Client) DownloadFile(URL string) (string, io.ReadCloser, *int64, error) {
	if err := client.refreshAccess(); err != nil {
		return "", nil, nil, err
	}
	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return "", nil, nil, err
	}
	request.Header.Add("Authorization", "Bearer "+*client.accessToken)
	response, err := client.Do(request)
	if err != nil {
		return "", nil, nil, err
	}
	if response.StatusCode/100 != 2 {
		return "", nil, nil, fmt.Errorf("Unexpected status code: %d", response.StatusCode)
	}

	segments := strings.Split(response.Request.URL.Path, "/")
	var length *int64
	if len(response.Header["Content-Length"]) > 0 {
		len, err := strconv.ParseInt(response.Header["Content-Length"][0], 10, 64)
		if err != nil {
			return "", nil, nil, err
		}
		length = &len
	}

	return segments[len(segments)-1], response.Body, length, nil
}
