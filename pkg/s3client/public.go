package s3client

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

func NewS3Context(bcfg *config.BucketConfig, logger *logrus.FieldLogger) (*S3Context, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(bcfg.Region)},
	)
	if err != nil {
		return nil, err
	}
	svcClient := s3.New(sess)
	return &S3Context{svcClient: svcClient, BucketConfig: bcfg, logger: logger}, nil
}

func (s3ctx *S3Context) ListFilesAndDirectories(path string) ([]*Entry, error) {
	// Trim first / if exists
	key := strings.TrimPrefix(path, "/")
	// List files on path
	folders := make([]*Entry, 0)
	files := make([]*Entry, 0)
	err := s3ctx.svcClient.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket:    aws.String(s3ctx.BucketConfig.Bucket),
			Prefix:    aws.String(key),
			Delimiter: aws.String("/"),
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			// Manage folders
			for _, item := range page.CommonPrefixes {
				name := strings.TrimPrefix(*item.Prefix, path)
				folders = append(folders, &Entry{
					Type: FolderType,
					Path: *item.Prefix,
					Name: name,
				})
			}
			// Manage files
			for _, item := range page.Contents {
				name := strings.TrimPrefix(*item.Key, path)
				if name != "" {
					files = append(files, &Entry{
						Type:         FileType,
						ETag:         *item.ETag,
						Name:         name,
						LastModified: *item.LastModified,
						Size:         *item.Size,
						Path:         *item.Key,
					})
				}
			}
			return lastPage
		})
	// Check if errors exists
	if err != nil {
		return nil, err
	}
	// Concat folders and files
	all := append(folders, files...)
	return all, nil
}
