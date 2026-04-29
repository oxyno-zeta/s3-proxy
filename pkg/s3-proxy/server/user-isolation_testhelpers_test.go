//go:build integration

package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/require"

	awssession "github.com/aws/aws-sdk-go/aws/session"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// Test helpers shared across the user-isolation integration tests. Anything
// here is built-tag integration so it does not leak into production builds.

// newIsolationFakeS3 spins up a gofakes3 server and creates the named bucket.
// The caller is responsible for closing the returned httptest.Server.
func newIsolationFakeS3(accessKey, secretAccessKey, region, bucket string) (*s3.S3, *httptest.Server, error) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())

	cfg := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretAccessKey, ""),
		Endpoint:         new(ts.URL),
		Region:           new(region),
		DisableSSL:       new(true),
		S3ForcePathStyle: new(true),
	}
	cli := s3.New(awssession.New(cfg))
	_, err := cli.CreateBucket(&s3.CreateBucketInput{Bucket: new(bucket)})

	return cli, ts, err
}

// seedIsolationFakeS3 PUTs a key→content map of objects into the bucket.
// Tests that need pre-existing data call this after newIsolationFakeS3.
func seedIsolationFakeS3(t *testing.T, cli *s3.S3, bucket string, files map[string]string) {
	t.Helper()

	for k, v := range files {
		_, err := cli.PutObject(&s3.PutObjectInput{
			Body:   strings.NewReader(v),
			Bucket: new(bucket),
			Key:    new(k),
		})
		require.NoError(t, err)
	}
}

// defaultIsolationServerConfig returns the ServerConfig used by every user
// isolation integration test. Centralising avoids drift in the compress
// settings as those defaults evolve.
func defaultIsolationServerConfig() *config.ServerConfig {
	return &config.ServerConfig{
		Compress: &config.ServerCompressConfig{
			Enabled: &config.DefaultServerCompressEnabled,
			Level:   config.DefaultServerCompressLevel,
			Types:   config.DefaultServerCompressTypes,
		},
	}
}
