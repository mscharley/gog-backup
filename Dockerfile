FROM golang:1.16-alpine

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

ENTRYPOINT [ "gog-backup" ]
CMD [ "-config", "/etc/gog-backup.ini" ]
