package s3client

import (
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

func NewS3Context(binst *config.BucketInstance, logger *logrus.FieldLogger) (*S3Context, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(binst.Bucket.Region)},
	)
	if err != nil {
		return nil, err
	}
	svcClient := s3.New(sess)
	return &S3Context{svcClient: svcClient, logger: logger, BucketInstance: binst}, nil
}

func (s3ctx *S3Context) ListFilesAndDirectories(key string) ([]*Entry, error) {
	// List files on path
	folders := make([]*Entry, 0)
	files := make([]*Entry, 0)
	err := s3ctx.svcClient.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket:    aws.String(s3ctx.BucketInstance.Bucket.Name),
			Prefix:    aws.String(key),
			Delimiter: aws.String("/"),
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			// Manage folders
			for _, item := range page.CommonPrefixes {
				name := strings.TrimPrefix(*item.Prefix, key)
				folders = append(folders, &Entry{
					Type: FolderType,
					Key:  *item.Prefix,
					Name: name,
				})
			}
			// Manage files
			for _, item := range page.Contents {
				name := strings.TrimPrefix(*item.Key, key)
				if name != "" {
					files = append(files, &Entry{
						Type:         FileType,
						ETag:         *item.ETag,
						Name:         name,
						LastModified: *item.LastModified,
						Size:         *item.Size,
						Key:          *item.Key,
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

func (s3ctx *S3Context) GetObject(key string) (*io.ReadCloser, error) {
	obj, err := s3ctx.svcClient.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3ctx.BucketInstance.Bucket.Name),
		Key:    aws.String(key),
	})
	if err != nil {
		// Try to cast error into an AWS Error if possible
		aerr, ok := err.(awserr.Error)
		if ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, s3.ErrCodeNoSuchKey:
				return nil, ErrNotFound
			}
		}
		return nil, err
	}
	return &obj.Body, nil
}
