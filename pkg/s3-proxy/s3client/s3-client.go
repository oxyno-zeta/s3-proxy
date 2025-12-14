package s3client

import (
	"context"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
)

type s3client struct {
	svcClient         s3iface.S3API
	s3managerUploader s3manageriface.UploaderAPI
	target            *config.TargetConfig
	metricsCtx        metrics.Client
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

const s3MaxKeys int64 = 1000

func formatContentDigestFromS3Checksums(checksumSHA256, checksumSHA1, checksumCRC32C, checksumCRC32 *string) string {
	// Only return checksums for FULL_OBJECT type.
	// COMPOSITE checksums (from multipart uploads) are the checksum of the concatenation of the parts' checksums and
	// have the format "{base64 hash}-{partcount}" '-' is not a valid base64 character so we can detect this.
	if checksumSHA256 != nil && !strings.Contains(*checksumSHA256, "-") {
		return "sha-256=:" + *checksumSHA256 + ":"
	}

	if checksumSHA1 != nil && !strings.Contains(*checksumSHA1, "-") {
		return "sha-1=:" + *checksumSHA1 + ":"
	}

	if checksumCRC32C != nil && !strings.Contains(*checksumCRC32C, "-") {
		return "crc32c=:" + *checksumCRC32C + ":"
	}

	if checksumCRC32 != nil && !strings.Contains(*checksumCRC32, "-") {
		return "crc32=:" + *checksumCRC32 + ":"
	}

	return ""
}

func (s3cl *s3client) buildGetObjectInputFromInput(input *GetInput) *s3.GetObjectInput {
	s3Input := &s3.GetObjectInput{
		Bucket:            aws.String(s3cl.target.Bucket.Name),
		Key:               aws.String(input.Key),
		IfModifiedSince:   input.IfModifiedSince,
		IfUnmodifiedSince: input.IfUnmodifiedSince,
		ChecksumMode:      aws.String("ENABLED"),
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

func (s3cl *s3client) GetObjectSignedURL(ctx context.Context, input *GetInput, expiration time.Duration) (string, error) {
	// Build input
	s3Input := s3cl.buildGetObjectInputFromInput(input)

	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.get-object-signed-url-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3cl.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3cl.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3cl.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3cl.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-bucket.bucket-key", *s3Input.Key)
	childTrace.SetTag("s3-proxy.target-name", s3cl.target.Name)

	defer childTrace.Finish()

	// Get logger
	logger := log.GetLoggerFromContext(ctx)
	// Build logger
	logger = logger.WithFields(map[string]any{
		"bucket": s3cl.target.Bucket.Name,
		"key":    *s3Input.Key,
		"region": s3cl.target.Bucket.Region,
	})
	// Log
	logger.Debugf("Trying to get object presigned url")

	// Build object request
	req, _ := s3cl.svcClient.GetObjectRequest(s3Input)
	// Build url
	urlStr, err := req.Presign(expiration)
	// Check error
	if err != nil {
		// Try to cast error into an AWS Error if possible
		//nolint: errorlint // Cast
		aerr, ok := err.(awserr.Error)
		if ok {
			// Check if it is a not found case
			if aerr.Code() == s3.ErrCodeNoSuchKey {
				return "", ErrNotFound
			} else if aerr.Code() == "PreconditionFailed" {
				return "", ErrPreconditionFailed
			}
		}

		return "", errors.WithStack(err)
	}

	// Log
	logger.Debug("Get object presigned url with success")

	return urlStr, nil
}

// ListFilesAndDirectories List files and directories.
func (s3cl *s3client) ListFilesAndDirectories(ctx context.Context, key string) ([]*ListElementOutput, *ResultInfo, error) {
	// Get logger
	logger := log.GetLoggerFromContext(ctx)

	// List files on path
	folders := make([]*ListElementOutput, 0)
	files := make([]*ListElementOutput, 0)
	// Prepare next token structure
	var nextToken *string
	// Temporary max elements for limits
	tmpMaxElements := s3cl.target.Bucket.S3ListMaxKeys
	// Loop control
	loopControl := true
	// Initialize max keys
	maxKeys := min(
		// Check size of max keys
		s3cl.target.Bucket.S3ListMaxKeys, s3MaxKeys)

	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)

	// Init & get request headers
	var requestHeaders map[string]string
	if s3cl.target.Bucket.RequestConfig != nil {
		requestHeaders = s3cl.target.Bucket.RequestConfig.ListHeaders
	}

	// Loop
	for loopControl {
		// Create child trace
		childTrace := parentTrace.GetChildTrace("s3-bucket.list-objects-request")
		childTrace.SetTag("s3-bucket.bucket-name", s3cl.target.Bucket.Name)
		childTrace.SetTag("s3-bucket.bucket-region", s3cl.target.Bucket.Region)
		childTrace.SetTag("s3-bucket.bucket-prefix", s3cl.target.Bucket.Prefix)
		childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3cl.target.Bucket.S3Endpoint)
		childTrace.SetTag("s3-bucket.bucket-key", key)
		childTrace.SetTag("s3-proxy.target-name", s3cl.target.Name)

		// Build logger
		logger = logger.WithFields(map[string]any{
			"bucket":  s3cl.target.Bucket.Name,
			"key":     key,
			"region":  s3cl.target.Bucket.Region,
			"maxKeys": maxKeys,
		})
		// Log
		logger.Debugf("Trying to list objects")

		// Request S3
		err := s3cl.svcClient.ListObjectsV2PagesWithContext(
			ctx,
			&s3.ListObjectsV2Input{
				Bucket:            aws.String(s3cl.target.Bucket.Name),
				Prefix:            aws.String(key),
				Delimiter:         aws.String("/"),
				MaxKeys:           aws.Int64(maxKeys),
				ContinuationToken: nextToken,
			},
			func(page *s3.ListObjectsV2Output, lastPage bool) bool {
				// Store next token
				nextToken = page.NextContinuationToken

				// Check if keycount exists
				if page.KeyCount != nil {
					// Remove current keys to tmp max elements
					tmpMaxElements -= *page.KeyCount
					// Update max keys if needed
					if tmpMaxElements < maxKeys {
						maxKeys = tmpMaxElements
					}
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
							Size:         *item.Size,
							Key:          *item.Key,
						})
					}
				}

				return lastPage
			},
			addHeadersToRequest(requestHeaders),
		)

		// Metrics
		s3cl.metricsCtx.IncS3Operations(s3cl.target.Name, s3cl.target.Bucket.Name, ListObjectsOperation)

		// End trace
		childTrace.Finish()

		// Check if errors exists
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		// Log
		logger.Debugf("List objects done with success")
	}

	// Concat folders and files
	//nolint:gocritic // Ignoring this: appendAssign: append result not assigned to the same slice
	all := append(folders, files...)

	// Create info
	info := &ResultInfo{
		Bucket:     s3cl.target.Bucket.Name,
		Region:     s3cl.target.Bucket.Region,
		S3Endpoint: s3cl.target.Bucket.S3Endpoint,
		Key:        key,
	}

	return all, info, nil
}

// GetObject Get object from S3 bucket.
func (s3cl *s3client) GetObject(ctx context.Context, input *GetInput) (*GetOutput, *ResultInfo, error) {
	// Build input
	s3Input := s3cl.buildGetObjectInputFromInput(input)

	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.get-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3cl.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3cl.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3cl.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3cl.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-bucket.bucket-key", *s3Input.Key)
	childTrace.SetTag("s3-proxy.target-name", s3cl.target.Name)

	defer childTrace.Finish()

	// Init & get request headers
	var requestHeaders map[string]string
	if s3cl.target.Bucket.RequestConfig != nil {
		requestHeaders = s3cl.target.Bucket.RequestConfig.GetHeaders
	}

	// Get logger
	logger := log.GetLoggerFromContext(ctx)
	// Build logger
	logger = logger.WithFields(map[string]any{
		"bucket": s3cl.target.Bucket.Name,
		"key":    *s3Input.Key,
		"region": s3cl.target.Bucket.Region,
	})
	// Log
	logger.Debugf("Trying to get object")

	obj, err := s3cl.svcClient.GetObjectWithContext(
		ctx,
		s3Input,
		addHeadersToRequest(requestHeaders),
	)
	// Metrics
	s3cl.metricsCtx.IncS3Operations(s3cl.target.Name, s3cl.target.Bucket.Name, GetObjectOperation)
	// Check if error exists
	if err != nil {
		// Try to cast error into an AWS Error if possible
		//nolint: errorlint // Cast
		aerr, ok := err.(awserr.Error)
		if ok {
			// Check if it is a not found case
			//nolint: gocritic // Because don't want to write a switch for the moment
			if aerr.Code() == s3.ErrCodeNoSuchKey {
				return nil, nil, ErrNotFound
			} else if aerr.Code() == "NotModified" {
				return nil, nil, ErrNotModified
			} else if aerr.Code() == "PreconditionFailed" {
				return nil, nil, ErrPreconditionFailed
			}
		}

		return nil, nil, errors.WithStack(err)
	}
	// Build output
	output := &GetOutput{
		BaseFileOutput: &BaseFileOutput{},
		Body:           obj.Body,
	}

	// Metadata transformation
	if obj.Metadata != nil {
		output.Metadata = aws.StringValueMap(obj.Metadata)
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

	output.ContentDigest = formatContentDigestFromS3Checksums(obj.ChecksumSHA256, obj.ChecksumSHA1, obj.ChecksumCRC32C, obj.ChecksumCRC32)

	// Create info
	info := &ResultInfo{
		Bucket:     s3cl.target.Bucket.Name,
		S3Endpoint: s3cl.target.Bucket.S3Endpoint,
		Region:     s3cl.target.Bucket.Region,
		Key:        input.Key,
	}

	// Log
	logger.Debugf("Get object done with success")

	return output, info, nil
}

func (s3cl *s3client) PutObject(ctx context.Context, input *PutInput) (*ResultInfo, error) {
	// Build input
	inp := &s3manager.UploadInput{
		Body:    input.Body,
		Bucket:  aws.String(s3cl.target.Bucket.Name),
		Key:     aws.String(input.Key),
		Expires: input.Expires,
	}

	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.put-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3cl.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3cl.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3cl.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3cl.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-bucket.bucket-key", *inp.Key)
	childTrace.SetTag("s3-proxy.target-name", s3cl.target.Name)

	defer childTrace.Finish()

	// Get logger
	logger := log.GetLoggerFromContext(ctx)
	// Build logger
	logger = logger.WithFields(map[string]any{
		"bucket": s3cl.target.Bucket.Name,
		"key":    *inp.Key,
		"region": s3cl.target.Bucket.Region,
	})
	// Log
	logger.Debugf("Trying to put object")

	// Manage ACL
	if s3cl.target.Actions != nil &&
		s3cl.target.Actions.PUT != nil &&
		s3cl.target.Actions.PUT.Config != nil &&
		s3cl.target.Actions.PUT.Config.CannedACL != nil &&
		*s3cl.target.Actions.PUT.Config.CannedACL != "" {
		// Inject ACL
		inp.ACL = s3cl.target.Actions.PUT.Config.CannedACL
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
	// Manage metadata case
	if input.Metadata != nil {
		inp.Metadata = aws.StringMap(input.Metadata)
	}
	// Manage storage class
	if input.StorageClass != "" {
		inp.StorageClass = aws.String(input.StorageClass)
	}

	// Init & get request headers
	var requestHeaders map[string]string
	if s3cl.target.Bucket.RequestConfig != nil {
		requestHeaders = s3cl.target.Bucket.RequestConfig.PutHeaders
	}

	// Upload to S3 bucket
	_, err := s3cl.s3managerUploader.UploadWithContext(ctx, inp, func(u *s3manager.Uploader) {
		// Check if it exists
		if u.RequestOptions == nil {
			u.RequestOptions = make([]request.Option, 0)
		}
		// Save new option
		u.RequestOptions = append(u.RequestOptions, addHeadersToRequest(requestHeaders))
	})
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Metrics
	s3cl.metricsCtx.IncS3Operations(s3cl.target.Name, s3cl.target.Bucket.Name, PutObjectOperation)

	// Create info
	info := &ResultInfo{
		Bucket:     s3cl.target.Bucket.Name,
		S3Endpoint: s3cl.target.Bucket.S3Endpoint,
		Region:     s3cl.target.Bucket.Region,
		Key:        input.Key,
	}

	// Log
	logger.Debugf("Put object done with success")

	// Return
	return info, nil
}

func (s3cl *s3client) HeadObject(ctx context.Context, key string) (*HeadOutput, *ResultInfo, error) {
	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.head-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3cl.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3cl.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3cl.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3cl.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-bucket.bucket-key", key)
	childTrace.SetTag("s3-proxy.target-name", s3cl.target.Name)

	defer childTrace.Finish()

	// Get logger
	logger := log.GetLoggerFromContext(ctx)
	// Build logger
	logger = logger.WithFields(map[string]any{
		"bucket": s3cl.target.Bucket.Name,
		"key":    key,
		"region": s3cl.target.Bucket.Region,
	})
	// Log
	logger.Debugf("Trying to head object")

	// Init & get request headers
	var requestHeaders map[string]string
	if s3cl.target.Bucket.RequestConfig != nil {
		requestHeaders = s3cl.target.Bucket.RequestConfig.GetHeaders
	}

	// Head object in bucket
	obj, err := s3cl.svcClient.HeadObjectWithContext(
		ctx,
		&s3.HeadObjectInput{
			Bucket:       aws.String(s3cl.target.Bucket.Name),
			Key:          aws.String(key),
			ChecksumMode: aws.String("ENABLED"),
		},
		addHeadersToRequest(requestHeaders),
	)
	// Metrics
	s3cl.metricsCtx.IncS3Operations(s3cl.target.Name, s3cl.target.Bucket.Name, HeadObjectOperation)
	// Test error
	if err != nil {
		// Try to cast error into an AWS Error if possible
		//nolint: errorlint // Cast
		aerr, ok := err.(awserr.Error)
		if ok {
			// Issue not fixed: https://github.com/aws/aws-sdk-go/issues/1208
			if aerr.Code() == "NotFound" {
				return nil, nil, ErrNotFound
			}
		}

		return nil, nil, errors.WithStack(err)
	}
	// Generate output
	output := &HeadOutput{
		BaseFileOutput: &BaseFileOutput{},
		Type:           FileType,
		Key:            key,
	}

	// Metadata transformation
	if obj.Metadata != nil {
		output.Metadata = aws.StringValueMap(obj.Metadata)
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

	if obj.ContentType != nil {
		output.ContentType = *obj.ContentType
	}

	if obj.ETag != nil {
		output.ETag = *obj.ETag
	}

	if obj.LastModified != nil {
		output.LastModified = *obj.LastModified
	}

	output.ContentDigest = formatContentDigestFromS3Checksums(obj.ChecksumSHA256, obj.ChecksumSHA1, obj.ChecksumCRC32C, obj.ChecksumCRC32)

	// Create info
	info := &ResultInfo{
		Bucket:     s3cl.target.Bucket.Name,
		S3Endpoint: s3cl.target.Bucket.S3Endpoint,
		Region:     s3cl.target.Bucket.Region,
		Key:        key,
	}

	// Log
	logger.Debugf("Head object done with success")

	// Return output
	return output, info, nil
}

func (s3cl *s3client) DeleteObject(ctx context.Context, key string) (*ResultInfo, error) {
	// Get trace
	parentTrace := tracing.GetTraceFromContext(ctx)
	// Create child trace
	childTrace := parentTrace.GetChildTrace("s3-bucket.delete-object-request")
	childTrace.SetTag("s3-bucket.bucket-name", s3cl.target.Bucket.Name)
	childTrace.SetTag("s3-bucket.bucket-region", s3cl.target.Bucket.Region)
	childTrace.SetTag("s3-bucket.bucket-prefix", s3cl.target.Bucket.Prefix)
	childTrace.SetTag("s3-bucket.bucket-s3-endpoint", s3cl.target.Bucket.S3Endpoint)
	childTrace.SetTag("s3-bucket.bucket-key", key)
	childTrace.SetTag("s3-proxy.target-name", s3cl.target.Name)

	defer childTrace.Finish()

	// Get logger
	logger := log.GetLoggerFromContext(ctx)
	// Build logger
	logger = logger.WithFields(map[string]any{
		"bucket": s3cl.target.Bucket.Name,
		"key":    key,
		"region": s3cl.target.Bucket.Region,
	})
	// Log
	logger.Debugf("Trying to delete object")

	// Init & get request headers
	var requestHeaders map[string]string
	if s3cl.target.Bucket.RequestConfig != nil {
		requestHeaders = s3cl.target.Bucket.RequestConfig.DeleteHeaders
	}

	// Delete object
	_, err := s3cl.svcClient.DeleteObjectWithContext(
		ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(s3cl.target.Bucket.Name),
			Key:    aws.String(key),
		},
		addHeadersToRequest(requestHeaders),
	)
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Metrics
	s3cl.metricsCtx.IncS3Operations(s3cl.target.Name, s3cl.target.Bucket.Name, DeleteObjectOperation)

	// Create info
	info := &ResultInfo{
		Bucket:     s3cl.target.Bucket.Name,
		S3Endpoint: s3cl.target.Bucket.S3Endpoint,
		Region:     s3cl.target.Bucket.Region,
		Key:        key,
	}

	// Log
	logger.Debugf("Delete object done with success")

	// Return
	return info, nil
}

func addHeadersToRequest(headers map[string]string) func(r *request.Request) {
	return func(r *request.Request) {
		// Loop over them
		for k, v := range headers {
			r.HTTPRequest.Header.Set(k, v)
		}
	}
}
