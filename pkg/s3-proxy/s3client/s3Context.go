package s3client

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
)

type s3Context struct {
	svcClient  s3iface.S3API
	uploader   s3manageriface.UploaderAPI
	target     *config.TargetConfig
	logger     log.Logger
	metricsCtx metrics.Client
}

// ListObjectsOperation List objects operation
const ListObjectsOperation = "list-objects"

// GetObjectOperation Get object operation
const GetObjectOperation = "get-object"

// HeadObjectOperation Head object operation
const HeadObjectOperation = "head-object"

// PutObjectOperation Put object operation
const PutObjectOperation = "put-object"

// DeleteObjectOperation Delete object operation
const DeleteObjectOperation = "delete-object"

// ListFilesAndDirectories List files and directories
func (s3ctx *s3Context) ListFilesAndDirectories(key string) ([]*ListElementOutput, error) {
	// List files on path
	folders := make([]*ListElementOutput, 0)
	files := make([]*ListElementOutput, 0)
	err := s3ctx.svcClient.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket:    aws.String(s3ctx.target.Bucket.Name),
			Prefix:    aws.String(key),
			Delimiter: aws.String("/"),
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			// Manage folders
			for _, item := range page.CommonPrefixes {
				name := strings.TrimPrefix(*item.Prefix, key)
				folders = append(folders, &ListElementOutput{
					Type: FolderType,
					Key:  *item.Prefix,
					Name: name,
				})
			}
			// Manage files
			for _, item := range page.Contents {
				name := strings.TrimPrefix(*item.Key, key)
				if name != "" {
					files = append(files, &ListElementOutput{
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
	// Metrics
	s3ctx.metricsCtx.IncS3Operations(ListObjectsOperation)
	// Check if errors exists
	if err != nil {
		return nil, err
	}
	// Concat folders and files
	all := append(folders, files...)

	return all, nil
}

// GetObject Get object from S3 bucket
func (s3ctx *s3Context) GetObject(key string) (*GetOutput, error) {
	obj, err := s3ctx.svcClient.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3ctx.target.Bucket.Name),
		Key:    aws.String(key),
	})
	// Metrics
	s3ctx.metricsCtx.IncS3Operations(GetObjectOperation)
	// Check if error exists
	if err != nil {
		// Try to cast error into an AWS Error if possible
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, ErrNotFound
		}

		return nil, err
	}
	// Build output
	output := &GetOutput{
		Body: &obj.Body,
	}

	if obj.CacheControl != nil {
		output.CacheControl = *obj.CacheControl
	}

	if obj.Expires != nil {
		output.Expires = *obj.Expires
	}

	if obj.ContentDisposition != nil {
		output.ContentDisposition = *obj.ContentDisposition
	}

	if obj.ContentEncoding != nil {
		output.ContentEncoding = *obj.ContentEncoding
	}

	if obj.ContentLanguage != nil {
		output.ContentLanguage = *obj.ContentLanguage
	}

	if obj.ContentLength != nil {
		output.ContentLength = *obj.ContentLength
	}

	if obj.ContentRange != nil {
		output.ContentRange = *obj.ContentRange
	}

	if obj.ContentType != nil {
		output.ContentType = *obj.ContentType
	}

	if obj.ETag != nil {
		output.ETag = *obj.ETag
	}

	if obj.LastModified != nil {
		output.LastModified = *obj.LastModified
	}

	return output, nil
}

func (s3ctx *s3Context) PutObject(input *PutInput) error {
	inp := &s3manager.UploadInput{
		Bucket: aws.String(s3ctx.target.Bucket.Name),
		Key:    aws.String(input.Key),
		Body:   input.Body,
	}
	// Manage content type case
	if input.ContentType != "" {
		inp.ContentType = aws.String(input.ContentType)
	}
	// Manage metadata case
	if input.Metadata != nil {
		inp.Metadata = aws.StringMap(input.Metadata)
	}
	// Manage storage class
	if input.StorageClass != "" {
		inp.StorageClass = aws.String(input.StorageClass)
	}
	// Upload to S3 bucket
	_, err := s3ctx.uploader.Upload(inp)
	// Metrics
	s3ctx.metricsCtx.IncS3Operations(PutObjectOperation)
	// Return error
	return err
}

func (s3ctx *s3Context) HeadObject(key string) (*HeadOutput, error) {
	// Head object in bucket
	_, err := s3ctx.svcClient.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s3ctx.target.Bucket.Name),
		Key:    aws.String(key),
	})
	// Metrics
	s3ctx.metricsCtx.IncS3Operations(HeadObjectOperation)
	// Test error
	if err != nil {
		// Try to cast error into an AWS Error if possible
		aerr, ok := err.(awserr.Error)
		if ok {
			// Issue not fixed: https://github.com/aws/aws-sdk-go/issues/1208
			if aerr.Code() == "NotFound" {
				return nil, ErrNotFound
			}
		}

		return nil, err
	}
	// Generate output
	output := &HeadOutput{
		Type: FileType,
		Key:  key,
	}
	// Return output
	return output, nil
}

func (s3ctx *s3Context) DeleteObject(key string) error {
	// Delete object
	_, err := s3ctx.svcClient.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s3ctx.target.Bucket.Name),
		Key:    aws.String(key),
	})
	// Metrics
	s3ctx.metricsCtx.IncS3Operations(DeleteObjectOperation)
	// Return error
	return err
}
