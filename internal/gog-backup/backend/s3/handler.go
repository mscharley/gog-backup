package s3

import (
	"flag"
	"io"
	"log"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/juju/ratelimit"
	"github.com/mscharley/gog-backup/internal/gog-backup/backend"
)

var (
	bucket = flag.String("s3-bucket", "", "The bucket to upload to. (backend=s3)")
	prefix = flag.String("s3-prefix", "", "A prefix path to upload into a directory. (backend=s3)")
)

type handler struct {
	downloader   *s3manager.Downloader
	uploader     *s3manager.Uploader
	uploadBucket *ratelimit.Bucket
	svc          *s3.S3
}

// NewHandler creates a backend linked to an S3 bucket.
func NewHandler(uploadBucket *ratelimit.Bucket) (backend.Handler, error) {
	sess := session.Must(session.NewSession())
	region, err := s3manager.GetBucketRegion(aws.BackgroundContext(), sess, *bucket, "us-east-1")
	if err != nil {
		return nil, err
	}
	sess.Config.Region = &region

	log.Printf("Detected s3://%s in region %s\n", *bucket, region)

	return &handler{
		downloader:   s3manager.NewDownloader(sess),
		uploader:     s3manager.NewUploader(sess),
		uploadBucket: uploadBucket,
		svc:          s3.New(sess),
	}, nil
}

func (h *handler) GetPrefix() string {
	return *prefix
}

func (h *handler) GetDisplayPrefix() string {
	return "s3://" + *bucket
}

func (h *handler) ReadFile(filename string) (string, error) {
	buff := aws.NewWriteAtBuffer(make([]byte, 64))
	_, err := (*h.downloader).Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(filename),
	})

	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(buff.Bytes()), "\x00"), nil
}

func (h *handler) WriteFile(filename string, content string) error {
	_, err := (*h.uploader).Upload(&s3manager.UploadInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(filename),
		Body:   strings.NewReader(content),
	})

	return err
}

func (h *handler) FileExists(filename string) (bool, error) {
	_, err := (*h.svc).HeadObject(&s3.HeadObjectInput{
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

func (h *handler) TransferFile(reader io.Reader, basepath string, filename string) error {
	key := path.Join(basepath, filename)
	var Body io.Reader
	if (*h).uploadBucket == nil {
		Body = reader
	} else {
		Body = ratelimit.Reader(reader, (*h).uploadBucket)
	}

	_, err := (*h.uploader).Upload(&s3manager.UploadInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(key),
		Body:   Body,
	})

	return err
}
