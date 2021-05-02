package bucket

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
)

// requestContext Bucket request context.
type requestContext struct {
	s3ClientManager s3client.Manager
	targetCfg       *config.TargetConfig
	mountPath       string
}

// generateStartKey will generate start key used in all functions.
func (rctx *requestContext) generateStartKey(requestPath string) string {
	bucketRootPrefixKey := rctx.targetCfg.Bucket.GetRootPrefix()
	// Key must begin by bucket prefix
	key := bucketRootPrefixKey
	// Trim first / if exists
	key += strings.TrimPrefix(requestPath, "/")

	return key
}

func (rctx *requestContext) manageKeyRewrite(key string) string {
	// Check if key rewrite list exists
	if rctx.targetCfg.KeyRewriteList != nil {
		// Loop over key rewrite list
		for _, kr := range rctx.targetCfg.KeyRewriteList {
			// Check if key is matching
			if kr.SourceRegex.MatchString(key) {
				// Find submatches
				submatches := kr.SourceRegex.FindStringSubmatchIndex(key)

				// Check if there is a submatch
				if len(submatches) == 0 {
					return kr.Target
				}

				// Create result
				result := []byte{}
				// Replace matches in target
				result = kr.SourceRegex.ExpandString(result, kr.Target, key, submatches)
				// Return result
				return string(result)
			}
		}
	}

	// Default case is returning the input key
	return key
}

// Get proxy GET requests.
func (rctx *requestContext) Get(ctx context.Context, input *GetInput) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Generate start key
	key := rctx.generateStartKey(input.RequestPath)
	// Manage key rewrite
	key = rctx.manageKeyRewrite(key)
	// Check that the path ends with a / for a directory listing or the main path special case (empty path)
	if strings.HasSuffix(input.RequestPath, "/") || input.RequestPath == "" {
		rctx.manageGetFolder(ctx, key, input)
		// Stop
		return
	}

	// Get object case
	err := rctx.streamFileForResponse(ctx, key, input)
	if err != nil {
		// Check if error is a not found error
		// nolint: gocritic // Don't want a switch
		if errors.Is(err, s3client.ErrNotFound) {
			// Test that redirect with trailing slash isn't asked and possible on this request
			if rctx.targetCfg.Actions != nil && rctx.targetCfg.Actions.GET != nil &&
				rctx.targetCfg.Actions.GET.Config != nil &&
				rctx.targetCfg.Actions.GET.Config.RedirectWithTrailingSlashForNotFoundFile &&
				!strings.HasSuffix(input.RequestPath, "/") {
				// Redirect with trailing slash
				resHan.RedirectWithTrailingSlash()

				return
			}
			// Not found
			resHan.NotFoundError(rctx.LoadFileContent)
			// Stop
			return
		} else if errors.Is(err, s3client.ErrNotModified) {
			// Not modified
			resHan.NotModified()

			return
		} else if errors.Is(err, s3client.ErrPreconditionFailed) {
			// Precondition failed
			resHan.PreconditionFailed()

			return
		}
		// Manage error response
		resHan.InternalServerError(rctx.LoadFileContent, err)
		// Stop
		return
	}
}

func (rctx *requestContext) manageGetFolder(ctx context.Context, key string, input *GetInput) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Check if index document is activated
	if rctx.targetCfg.Actions != nil && rctx.targetCfg.Actions.GET != nil &&
		rctx.targetCfg.Actions.GET.Config != nil &&
		rctx.targetCfg.Actions.GET.Config.IndexDocument != "" {
		// Create index key path
		indexKey := path.Join(key, rctx.targetCfg.Actions.GET.Config.IndexDocument)
		// Head index file in bucket
		headOutput, err := rctx.s3ClientManager.
			GetClientForTarget(rctx.targetCfg.Name).
			HeadObject(ctx, indexKey)
		// Check if error exists and not a not found error
		if err != nil && !errors.Is(err, s3client.ErrNotFound) {
			// Manage error response
			resHan.InternalServerError(rctx.LoadFileContent, err)
			// Stop
			return
		}
		// Check that we found the file
		if headOutput != nil {
			// Get data
			err = rctx.streamFileForResponse(ctx, headOutput.Key, input)
			// Check if error exists
			if err != nil {
				// Check if error is a not found error
				// nolint: gocritic // Don't want a switch
				if errors.Is(err, s3client.ErrNotFound) {
					// Not found
					resHan.NotFoundError(rctx.LoadFileContent)

					return
				} else if errors.Is(err, s3client.ErrNotModified) {
					// Not modified
					resHan.NotModified()

					return
				} else if errors.Is(err, s3client.ErrPreconditionFailed) {
					// Precondition failed
					resHan.PreconditionFailed()

					return
				}
				// Response with error
				resHan.InternalServerError(rctx.LoadFileContent, err)
				// Stop
				return
			}
			// Stop here because no error are present
			return
		}
	}

	// Directory listing case
	s3Entries, err := rctx.s3ClientManager.
		GetClientForTarget(rctx.targetCfg.Name).
		ListFilesAndDirectories(ctx, key)
	if err != nil {
		resHan.InternalServerError(rctx.LoadFileContent, err)
		// Stop
		return
	}

	// Transform entries in entry with path objects
	bucketRootPrefixKey := rctx.targetCfg.Bucket.GetRootPrefix()
	entries := transformS3Entries(s3Entries, rctx, bucketRootPrefixKey)

	// Answer
	resHan.FoldersFilesList(
		rctx.LoadFileContent,
		entries,
	)
}

// Put proxy PUT requests.
func (rctx *requestContext) Put(ctx context.Context, inp *PutInput) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Generate start key
	key := rctx.generateStartKey(inp.RequestPath)
	// Add / at the end if not present
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	// Add filename at the end of key
	key += inp.Filename
	// Manage key rewrite
	key = rctx.manageKeyRewrite(key)
	// Create input
	input := &s3client.PutInput{
		Key:         key,
		Body:        inp.Body,
		ContentType: inp.ContentType,
		ContentSize: inp.ContentSize,
	}

	// Check if post actions configuration exists
	if rctx.targetCfg.Actions.PUT != nil &&
		rctx.targetCfg.Actions.PUT.Config != nil {
		// Check if metadata is configured in target configuration
		if rctx.targetCfg.Actions.PUT.Config.Metadata != nil {
			input.Metadata = rctx.targetCfg.Actions.PUT.Config.Metadata
		}

		// Check if storage class is present in target configuration
		if rctx.targetCfg.Actions.PUT.Config.StorageClass != "" {
			input.StorageClass = rctx.targetCfg.Actions.PUT.Config.StorageClass
		}

		// Check if allow override is enabled
		if !rctx.targetCfg.Actions.PUT.Config.AllowOverride {
			// Need to check if file already exists
			headOutput, err := rctx.s3ClientManager.
				GetClientForTarget(rctx.targetCfg.Name).
				HeadObject(ctx, key)
			// Check if error is not found if exists
			if err != nil && !errors.Is(err, s3client.ErrNotFound) {
				resHan.InternalServerError(rctx.LoadFileContent, err)
				// Stop
				return
			}
			// Check if file exists
			if headOutput != nil {
				// Create error
				err := fmt.Errorf("file detected on path %s for PUT request and override isn't allowed", key)
				// Response
				resHan.ForbiddenError(rctx.LoadFileContent, err)
				// Stop
				return
			}
		}
	}
	// Put file
	err := rctx.s3ClientManager.
		GetClientForTarget(rctx.targetCfg.Name).
		PutObject(ctx, input)
	if err != nil {
		resHan.InternalServerError(rctx.LoadFileContent, err)
		// Stop
		return
	}

	// Answer with no content
	resHan.NoContent()
}

// Delete will delete object in S3.
func (rctx *requestContext) Delete(ctx context.Context, requestPath string) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Generate start key
	key := rctx.generateStartKey(requestPath)
	// Manage key rewrite
	key = rctx.manageKeyRewrite(key)
	// Check that the path ends with a / for a directory or the main path special case (empty path)
	if strings.HasSuffix(requestPath, "/") || requestPath == "" {
		resHan.InternalServerError(rctx.LoadFileContent, ErrRemovalFolder)
		// Stop
		return
	}
	// Delete object in S3
	err := rctx.s3ClientManager.GetClientForTarget(rctx.targetCfg.Name).DeleteObject(ctx, key)
	// Check if error exists
	if err != nil {
		resHan.InternalServerError(rctx.LoadFileContent, err)
		// Stop
		return
	}

	// Answer with no content
	resHan.NoContent()
}

func transformS3Entries(
	s3Entries []*s3client.ListElementOutput,
	rctx *requestContext,
	bucketRootPrefixKey string,
) []*responsehandler.Entry {
	// Prepare result
	entries := make([]*responsehandler.Entry, 0)
	// Loop over s3 entries
	for _, item := range s3Entries {
		// Store path
		ePath := path.Join(rctx.mountPath, strings.TrimPrefix(item.Key, bucketRootPrefixKey))
		// Check if type is a folder in order to add a trailing /
		// Note: path.Join removed trailing /
		if item.Type == s3client.FolderType {
			ePath += "/"
		}
		// Save new entry
		entries = append(entries, &responsehandler.Entry{
			Type:         item.Type,
			ETag:         item.ETag,
			Name:         item.Name,
			LastModified: item.LastModified,
			Size:         item.Size,
			Key:          item.Key,
			Path:         ePath,
		})
	}
	// Return result
	return entries
}

func (rctx *requestContext) LoadFileContent(ctx context.Context, path string) (string, error) {
	// Get object from s3
	objOutput, err := rctx.s3ClientManager.GetClientForTarget(rctx.targetCfg.Name).GetObject(ctx, &s3client.GetInput{
		Key: path,
	})
	// Check error
	if err != nil {
		return "", err
	}

	// Read all body
	bb, err := ioutil.ReadAll(objOutput.Body)
	// Check error
	if err != nil {
		return "", err
	}

	// Transform it to string and return
	return string(bb), nil
}

func (rctx *requestContext) streamFileForResponse(ctx context.Context, key string, input *GetInput) error {
	// Get response handler from context
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Get object from s3
	objOutput, err := rctx.s3ClientManager.
		GetClientForTarget(rctx.targetCfg.Name).
		GetObject(ctx, &s3client.GetInput{
			Key:               key,
			IfModifiedSince:   input.IfModifiedSince,
			IfMatch:           input.IfMatch,
			IfNoneMatch:       input.IfNoneMatch,
			IfUnmodifiedSince: input.IfUnmodifiedSince,
			Range:             input.Range,
		})
	// Check error
	if err != nil {
		return err
	}

	// Transform input
	inp := &responsehandler.StreamInput{
		Body:               objOutput.Body,
		CacheControl:       objOutput.CacheControl,
		Expires:            objOutput.Expires,
		ContentDisposition: objOutput.ContentDisposition,
		ContentEncoding:    objOutput.ContentEncoding,
		ContentLanguage:    objOutput.ContentLanguage,
		ContentLength:      objOutput.ContentLength,
		ContentRange:       objOutput.ContentRange,
		ContentType:        objOutput.ContentType,
		ETag:               objOutput.ETag,
		LastModified:       objOutput.LastModified,
	}

	// Stream
	err = resHan.StreamFile(inp)

	// Return potential error
	return err
}
