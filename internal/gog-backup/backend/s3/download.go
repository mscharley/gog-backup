package s3

import (
	"flag"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/bclicn/color"
	"github.com/mscharley/gog-backup/internal/gog-backup/backend"
	"github.com/mscharley/gog-backup/pkg/gog"
)

var (
	bucket = flag.String("s3-bucket", "", "The bucket to upload to. (backend=s3)")
	prefix = flag.String("s3-prefix", "", "A prefix path to upload into a directory. You probably want a trailing forwardslash. (backend=s3)")
)

func DownloadFile(retries *int) (backend.Handler, error) {
	// The session for S3.
	sess := session.Must(session.NewSession())
	region, err := s3manager.GetBucketRegion(aws.BackgroundContext(), sess, *bucket, "us-east-1")
	if err != nil {
		return nil, err
	}
	log.Printf("Detected s3://%s in region %s\n", *bucket, region)
	sess.Config.Region = &region
	svc := s3.New(sess)

	// Create an interface with S3
	uploader := s3manager.NewUploader(sess)
	_ = uploader
	downloader := s3manager.NewDownloader(sess)

	readFile := func(filename string) (string, error) {
		buff := aws.NewWriteAtBuffer(make([]byte, 64))
		_, err := downloader.Download(buff, &s3.GetObjectInput{
			Bucket: aws.String(*bucket),
			Key:    aws.String(filename),
		})

		if err != nil {
			return "", err
		}

		return string(buff.Bytes()), nil
	}

	fileExists := func(filename string) (bool, error) {
		_, err = svc.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(*bucket),
			Key:    aws.String(filename),
		})

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return false, nil
			default:
				return false, err
			}
		}

		if err != nil {
			return false, err
		}
		return true, nil
	}

	writeFile := func(filename string, content string) error {
		_, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(*bucket),
			Key:    aws.String(filename),
			Body:   strings.NewReader(content),
		})

		return err
	}

	downloadFile := func(reader io.ReadCloser, basepath string, filename string) error {
		defer reader.Close()
		tmpKey := path.Join(basepath, "."+filename+".tmp")
		key := path.Join(basepath, filename)

		_, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(*bucket),
			Key:    aws.String(tmpKey),
			Body:   reader,
		})

		if err != nil {
			return err
		}

		defer svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(*bucket),
			Key:    aws.String(tmpKey),
		})

		_, err = svc.CopyObject(&s3.CopyObjectInput{
			Bucket:     aws.String(*bucket),
			CopySource: aws.String("/" + *bucket + "/" + tmpKey),
			Key:        aws.String(key),
		})
		return err
	}

	handler := func(downloads <-chan *backend.GogFile, waitGroup *sync.WaitGroup, client *gog.Client) {
		for d := range downloads {
			basepath := d.File
			if len(*prefix) > 0 {
				basepath = path.Join(*prefix, basepath)
			}

			for i := 1; i <= *retries; i++ {
				filename, reader, err := client.DownloadFile(d.URL)

				if err != nil {
					log.Printf("Unable to connect to GoG: %+v", err)
					continue
				}

				// Check for version information from last time.
				versionFile := path.Join(basepath, "."+filename+".version")
				if d.Version != "" {
					if lastVersion, _ := readFile(versionFile); string(lastVersion) == d.Version {
						fmt.Printf("Skipping %s as it is already up to date.\n", d.Name)
						reader.Close()
						break
					}
				} else if info, _ := fileExists(path.Join(basepath, filename)); info {
					fmt.Printf("Skipping %s as it is already downloaded.\n", d.Name)
					reader.Close()
					break
				}

				version := ""
				if d.Version != "" {
					version = " (version: " + color.Purple(d.Version) + ")"
				}
				fmt.Printf("%s%s\n  %s -> %s\n", d.Name, version, color.LightBlue(d.URL), color.Green("s3://"+*bucket+"/"+path.Join(basepath, filename)))
				err = downloadFile(reader, basepath, filename)
				if err != nil {
					log.Printf("Unable to download file: %+v", err)
					continue
				}

				if d.Version != "" {
					// Save version information for next time.
					err = writeFile(versionFile, d.Version)
					if err != nil {
						log.Printf("Unable to save version file: %+v", err)
						// Good enough for this run through - we'll redownload next time and retry saving the version file then.
						break
					}
				}

				// We successfully managed to download this file, skip the rest of our retries.
				break
			}
		}

		waitGroup.Done()
	}

	return handler, nil
}
