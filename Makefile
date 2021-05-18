libs := $(shell find internal pkg -type f -name '*.go')

default: gog-backup
all: gog-backup
.PHONY: all

gog-backup: cmd/gog-backup/*.go $(libs)
	go get -v ./...
	go build -o gog-backup cmd/gog-backup/main.go
