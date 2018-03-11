package backend

import (
	"sync"

	"github.com/mscharley/gog-backup/pkg/gog"
)

// GogFile is a struct used to store details about a single download that needs to be processed. This is the data format used over the
// internal channels.
type GogFile struct {
	Name    string
	URL     string
	File    string
	Version string
}

// Handler is the definition of the interface between the frontend and backend for processing GogFiles.
type Handler func(<-chan *GogFile, *sync.WaitGroup, *gog.Client)
