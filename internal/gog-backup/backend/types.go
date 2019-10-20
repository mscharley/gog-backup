package backend

import "io"

// GogFile is a struct used to store details about a single download that needs to be processed. This is the data format used over the
// internal channels.
type GogFile struct {
	Name    string
	URL     string
	File    string
	Version string
}

// Handler is the definition of the interface between the frontend and backend for processing GogFiles.
type Handler interface {
	GetPrefix() string
	GetDisplayPrefix() string
	ReadFile(filename string) (string, error)
	WriteFile(filename string, content string) error
	FileExists(filename string) (bool, error)
	TransferFile(reader io.Reader, basepath string, filename string) error
}
