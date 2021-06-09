package local

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/mscharley/gog-backup/internal/gog-backup/backend"
)

var (
	targetDir = flag.String("local-dir", os.Getenv("HOME")+"/GoG", "The target directory to download to. (backend=local)")
)

type handler struct{}

// NewHandler creates a backend linked to a local directory.
func NewHandler() backend.Handler {
	return &handler{}
}

func (h *handler) GetPrefix() string {
	return *targetDir
}

func (h *handler) GetDisplayPrefix() string {
	return ""
}

func (h *handler) ReadFile(filename string) (string, error) {
	contents, err := ioutil.ReadFile(filename)
	return string(contents), err
}

func (h *handler) WriteFile(filename string, content string) error {
	return ioutil.WriteFile(filename, []byte(content), 0666)
}

func (h *handler) FileExists(filename string) (bool, error) {
	info, err := os.Stat(filename)
	return info != nil, err
}

func (h *handler) TransferFile(reader io.Reader, basepath string, filename string) error {
	if filename == "" {
		return fmt.Errorf("No filename available, skipping this file")
	}

	err := os.MkdirAll(basepath, os.ModePerm)
	if err != nil {
		return err
	}

	tmpfile := path.Join(basepath, "."+filename+".tmp")
	outfile := path.Join(basepath, filename)
	writer, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}

	err = os.Rename(tmpfile, outfile)
	return err
}
