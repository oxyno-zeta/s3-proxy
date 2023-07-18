package s3client

import (
	"context"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
)

type s3Context struct {
	svcClient     *s3.Client
	presignClient *s3.PresignClient
	target        *config.TargetConfig
	metricsCtx    metrics.Client
}

// ListObjectsOperation List objects operation.
const ListObjectsOperation = "list-objects"

// GetObjectOperation Get object operation.
const GetObjectOperation = "get-object"

// HeadObjectOperation Head object operation.
const HeadObjectOperation = "head-object"

// PutObjectOperation Put object operation.
const PutObjectOperation = "put-object"

// DeleteObjectOperation Delete object operation.
const DeleteObjectOperation = "delete-object"

const s3MaxKeys int32 = 1000

func (s3ctx *s3Context) buildGetObjectInputFromInput(input *GetInput) *s3.GetObjectInput {
	s3Input := &s3.GetObjectInput{
		Bucket:            aws.String(s3ctx.target.Bucket.Name),
		Key:               aws.String(input.Key),
		IfModifiedSince:   input.IfModifiedSince,
		IfUnmodifiedSince: input.IfUnmodifiedSince,
	}

	// Add Range if not empty
	if input.Range != "" {
		s3Input.Range = aws.String(input.Range)
	}

	// Add If Match if not empty
	if input.IfMatch != "" {
		s3Input.IfMatch = aws.String(input.IfMatch)
	}

	// Add If None Match if not empty
	if input.IfNoneMatch != "" {
		s3Input.IfNoneMatch = aws.String(input.IfNoneMatch)
	}

	return s3Input
}

func (s3ctx *s3Context) GetObjectSignedURL(ctx context.Context, input *GetInput, expiration time.Duration) (string, error) {
	// Build input
	s3Input := s3ctx.buildGetObjectInputFromInput(input)

	// Build object request
	req, err := s3ctx.presignClient.PresignGetObject(ctx, s3Input, s3.WithPresignExpires(expiration))
	// Check error
	if err != nil {
		var nsk *types.NoSuchKey
		// Check no such key error
		if errors.As(err, &nsk) {
			return "", ErrNotFound
		}

		// Check precondition failed
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "PreconditionFailed" {
				return "", ErrPreconditionFailed
			}
		}

		return "", errors.WithStack(err)
	}

	return req.URL, nil
}

// ListFilesAndDirectories List files and directories.
func (s3ctx *s3Context) ListFilesAndDirectories(ctx context.Context, key string) ([]*ListElementOutput, *ResultInfo, error) {
	// List files on path
	folders := make([]*ListElementOutput, 0)
	files := make([]*ListElementOutput, 0)
	// Prepare next token structure
	var nextToken *string
	// Temporary max elements for limits
	tmpMaxElements := s3ctx.target.Bucket.S3ListMaxKeys
	// Loop control
	loopControl := true
	// Initialize max keys
	maxKeys := s3MaxKeys
	// Check size of max keys
	if s3ctx.target.Bucket.S3ListMaxKeys < maxKeys {
		maxKeys = s3ctx.target.Bucket.S3ListMaxKeys
	}

	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)

	// Init & get request headers
	var requestHeaders map[string]string
	if s3ctx.target.Bucket.RequestConfig != nil {
		requestHeaders = s3ctx.target.Bucket.RequestConfig.ListHeaders
	}

	// Loop
	for loopControl {
		// Create child trace
		childTrace := parentTrace.GetChildTrace("s3-bucket.list-objects-request")
		childTrace.SetTag("s3-bucket.bucket-name", s3ctx.target.Bucket.Name)
		childTrace.SetTag("s3-bucket.bucket-region", s3ctx.target.Bucket.Region)
		childTrace.SetTag("s3-bucket.bucket-prefix", s3ctx.target.Bucket.Prefix)
		childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3ctx.target.Bucket.S3Endpoint)
		childTrace.SetTag("s3-proxy.target-name", s3ctx.target.Name)

		// Request S3
		page, err := s3ctx.svcClient.ListObjectsV2(
			ctx,
			&s3.ListObjectsV2Input{
				Bucket:            aws.String(s3ctx.target.Bucket.Name),
				Prefix:            aws.String(key),
				Delimiter:         aws.String("/"),
				MaxKeys:           maxKeys,
				ContinuationToken: nextToken,
			},
			func(o *s3.Options) {
				o.HTTPClient = customizeHTTPClient(o.HTTPClient, requestHeaders)
			},
		)

		// Store next token
		nextToken = page.NextContinuationToken

		// Remove current keys to tmp max elements
		tmpMaxElements -= page.KeyCount
		// Update max keys if needed
		if tmpMaxElements < maxKeys {
			maxKeys = tmpMaxElements
		}

		// Manage loop control
		loopControl = nextToken != nil && tmpMaxElements > 0

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
					Size:         item.Size,
					Key:          *item.Key,
				})
			}
		}

		// Metrics
		s3ctx.metricsCtx.IncS3Operations(s3ctx.target.Name, s3ctx.target.Bucket.Name, ListObjectsOperation)

		// End trace
		childTrace.Finish()

		// Check if errors exists
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	// Concat folders and files
	//nolint:gocritic // Ignoring this: appendAssign: append result not assigned to the same slice
	all := append(folders, files...)

	// Create info
	info := &ResultInfo{
		Bucket:     s3ctx.target.Bucket.Name,
		Region:     s3ctx.target.Bucket.Region,
		S3Endpoint: s3ctx.target.Bucket.S3Endpoint,
		Key:        key,
	}

	return all, info, nil
}

// GetObject Get object from S3 bucket.
func (s3ctx *s3Context) GetObject(ctx context.Context, input *GetInput) (*GetOutput, *ResultInfo, error) {
	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.get-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3ctx.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3ctx.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3ctx.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3ctx.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-proxy.target-name", s3ctx.target.Name)

	defer childTrace.Finish()

	// Build input
	s3Input := s3ctx.buildGetObjectInputFromInput(input)

	// Init & get request headers
	var requestHeaders map[string]string
	if s3ctx.target.Bucket.RequestConfig != nil {
		requestHeaders = s3ctx.target.Bucket.RequestConfig.GetHeaders
	}

	obj, err := s3ctx.svcClient.GetObject(
		ctx,
		s3Input,
		func(o *s3.Options) {
			o.HTTPClient = customizeHTTPClient(o.HTTPClient, requestHeaders)
		},
	)
	// Metrics
	s3ctx.metricsCtx.IncS3Operations(s3ctx.target.Name, s3ctx.target.Bucket.Name, GetObjectOperation)
	// Check if error exists
	if err != nil {
		var nsk *types.NoSuchKey
		// Check no such key error
		if errors.As(err, &nsk) {
			return nil, nil, ErrNotFound
		}

		// Check precondition failed
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "PreconditionFailed" {
				return nil, nil, ErrPreconditionFailed
			} else if apiErr.ErrorCode() == "NotModified" {
				return nil, nil, ErrNotModified
			}
		}

		return nil, nil, errors.WithStack(err)
	}
	// Build output
	output := &GetOutput{
		Body:          obj.Body,
		ContentLength: obj.ContentLength,
		Metadata:      obj.Metadata,
	}

	if obj.CacheControl != nil {
		output.CacheControl = *obj.CacheControl
	}

	if obj.Expires != nil {
		output.Expires = obj.Expires.Format(time.RFC1123)
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

	// Create info
	info := &ResultInfo{
		Bucket:     s3ctx.target.Bucket.Name,
		S3Endpoint: s3ctx.target.Bucket.S3Endpoint,
		Region:     s3ctx.target.Bucket.Region,
		Key:        input.Key,
	}

	return output, info, nil
}

func (s3ctx *s3Context) PutObject(ctx context.Context, input *PutInput) (*ResultInfo, error) {
	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.put-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3ctx.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3ctx.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3ctx.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3ctx.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-proxy.target-name", s3ctx.target.Name)

	defer childTrace.Finish()

	inp := &s3.PutObjectInput{
		Body:          input.Body,
		ContentLength: input.ContentSize,
		Bucket:        aws.String(s3ctx.target.Bucket.Name),
		Key:           aws.String(input.Key),
		Expires:       input.Expires,
		Metadata:      input.Metadata,
	}

	// Manage ACL
	if s3ctx.target.Actions != nil &&
		s3ctx.target.Actions.PUT != nil &&
		s3ctx.target.Actions.PUT.Config != nil &&
		s3ctx.target.Actions.PUT.Config.CannedACL != nil &&
		*s3ctx.target.Actions.PUT.Config.CannedACL != "" {
		// Inject ACL
		inp.ACL = types.ObjectCannedACL(*s3ctx.target.Actions.PUT.Config.CannedACL)
	}
	// Manage cache control case
	if input.CacheControl != "" {
		inp.CacheControl = aws.String(input.CacheControl)
	}
	// Manage content disposition case
	if input.ContentDisposition != "" {
		inp.ContentDisposition = aws.String(input.ContentDisposition)
	}
	// Manage content encoding case
	if input.ContentEncoding != "" {
		inp.ContentEncoding = aws.String(input.ContentEncoding)
	}
	// Manage content language case
	if input.ContentLanguage != "" {
		inp.ContentLanguage = aws.String(input.ContentLanguage)
	}
	// Manage content type case
	if input.ContentType != "" {
		inp.ContentType = aws.String(input.ContentType)
	}
	// Manage storage class
	if input.StorageClass != "" {
		inp.StorageClass = types.StorageClass(input.StorageClass)
	}

	// Init & get request headers
	var requestHeaders map[string]string
	if s3ctx.target.Bucket.RequestConfig != nil {
		requestHeaders = s3ctx.target.Bucket.RequestConfig.PutHeaders
	}

	// Upload to S3 bucket
	_, err := s3ctx.svcClient.PutObject(
		ctx,
		inp,
		func(o *s3.Options) {
			o.HTTPClient = customizeHTTPClient(o.HTTPClient, requestHeaders)
		},
	)
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Metrics
	s3ctx.metricsCtx.IncS3Operations(s3ctx.target.Name, s3ctx.target.Bucket.Name, PutObjectOperation)

	// Create info
	info := &ResultInfo{
		Bucket:     s3ctx.target.Bucket.Name,
		S3Endpoint: s3ctx.target.Bucket.S3Endpoint,
		Region:     s3ctx.target.Bucket.Region,
		Key:        input.Key,
	}

	// Return
	return info, nil
}

func (s3ctx *s3Context) HeadObject(ctx context.Context, key string) (*HeadOutput, error) {
	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.head-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3ctx.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3ctx.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3ctx.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3ctx.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-proxy.target-name", s3ctx.target.Name)

	defer childTrace.Finish()

	// Init & get request headers
	var requestHeaders map[string]string
	if s3ctx.target.Bucket.RequestConfig != nil {
		requestHeaders = s3ctx.target.Bucket.RequestConfig.GetHeaders
	}

	// Head object in bucket
	_, err := s3ctx.svcClient.HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(s3ctx.target.Bucket.Name),
			Key:    aws.String(key),
		},
		func(o *s3.Options) {
			o.HTTPClient = customizeHTTPClient(o.HTTPClient, requestHeaders)
		},
	)
	// Metrics
	s3ctx.metricsCtx.IncS3Operations(s3ctx.target.Name, s3ctx.target.Bucket.Name, HeadObjectOperation)
	// Test error
	if err != nil {
		// Check precondition failed
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "NotFound" {
				return nil, ErrNotFound
			}
		}

		return nil, errors.WithStack(err)
	}
	// Generate output
	output := &HeadOutput{
		Type: FileType,
		Key:  key,
	}
	// Return output
	return output, nil
}

func (s3ctx *s3Context) DeleteObject(ctx context.Context, key string) (*ResultInfo, error) {
	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.delete-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3ctx.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3ctx.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3ctx.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3ctx.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-proxy.target-name", s3ctx.target.Name)

	defer childTrace.Finish()

	// Init & get request headers
	var requestHeaders map[string]string
	if s3ctx.target.Bucket.RequestConfig != nil {
		requestHeaders = s3ctx.target.Bucket.RequestConfig.DeleteHeaders
	}

	// Delete object
	_, err := s3ctx.svcClient.DeleteObject(
		ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(s3ctx.target.Bucket.Name),
			Key:    aws.String(key),
		},
		func(o *s3.Options) {
			o.HTTPClient = customizeHTTPClient(o.HTTPClient, requestHeaders)
		},
	)
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Metrics
	s3ctx.metricsCtx.IncS3Operations(s3ctx.target.Name, s3ctx.target.Bucket.Name, DeleteObjectOperation)

	// Create info
	info := &ResultInfo{
		Bucket:     s3ctx.target.Bucket.Name,
		S3Endpoint: s3ctx.target.Bucket.S3Endpoint,
		Region:     s3ctx.target.Bucket.Region,
		Key:        key,
	}

	// Return
	return info, nil
}
