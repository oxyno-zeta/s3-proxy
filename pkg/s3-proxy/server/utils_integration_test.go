package server

import (
	"fmt"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
)

// Generate metrics instance
var metricsCtx = metrics.NewClient()

func setupFakeS3(accessKey, secretAccessKey, region, bucket string) (*httptest.Server, error) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())

	// configure S3 client
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretAccessKey, ""),
		Endpoint:         aws.String(ts.URL),
		Region:           aws.String(region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession := session.New(s3Config)

	s3Client := s3.New(newSession)
	cparams := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// Create a new bucket using the CreateBucket call.
	_, err := s3Client.CreateBucket(cparams)
	if err != nil {
		return nil, err
	}

	files := map[string]string{
		"folder1/test.txt":          "Hello folder1!",
		"folder1/index.html":        "<!DOCTYPE html><html><body><h1>Hello folder1!</h1></body></html>",
		"folder2/index.html":        "<!DOCTYPE html><html><body><h1>Hello folder2!</h1></body></html>",
		"folder3/index.html":        "<!DOCTYPE html><html><body><h1>Hello folder3!</h1></body></html>",
		"folder3/test.txt":          "Hello folder3!",
		"folder4/test.txt":          "Hello folder4!",
		"folder4/index.html":        "<!DOCTYPE html><html><body><h1>Hello folder4!</h1></body></html>",
		"folder4/sub1/test.txt":     "Hello folder4!",
		"folder4/sub2/test.txt":     "Hello folder4!",
		"templates/folder-list.tpl": "fake template !",
	}

	// Inject large number of elements
	for i := 0; i < 2000; i++ {
		// Update map of files
		files[fmt.Sprintf("folder3/%d", i)] = fmt.Sprintf("content %d", i)
	}

	// Upload files
	for k, v := range files {
		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Body:   strings.NewReader(v),
			Bucket: aws.String(bucket),
			Key:    aws.String(k),
		})
		if err != nil {
			return nil, err
		}
	}

	// Add file with content-type
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Body:   strings.NewReader("test"),
		Bucket: aws.String(bucket),
		Key:    aws.String("content-type/file.txt"),
		Metadata: map[string]*string{
			"Content-Type": aws.String("text/plain"),
		},
	})
	if err != nil {
		return nil, err
	}

	return ts, nil
}
