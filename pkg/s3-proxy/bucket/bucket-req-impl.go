package bucket

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"emperror.dev/errors"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	responsehandlermodels "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	utils "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/templateutils"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook"
)

// bucketReqImpl Bucket request context.
type bucketReqImpl struct {
	s3ClientManager s3client.Manager
	webhookManager  webhook.Manager
	targetCfg       *config.TargetConfig
	mountPath       string
	generalHelpers  []string
}

// generateStartKey will generate start key used in all functions.
func (bri *bucketReqImpl) generateStartKey(requestPath string) string {
	bucketRootPrefixKey := bri.targetCfg.Bucket.GetRootPrefix()
	// Key must begin by bucket prefix
	key := bucketRootPrefixKey
	// Trim first / if exists
	key += strings.TrimPrefix(requestPath, "/")

	return key
}

func (bri *bucketReqImpl) manageKeyRewrite(ctx context.Context, key string) (string, error) {
	// Check if key rewrite list exists
	if bri.targetCfg.KeyRewriteList != nil {
		// Get logger
		logger := log.GetLoggerFromContext(ctx)

		// Loop over key rewrite list
		for _, kr := range bri.targetCfg.KeyRewriteList {
			// Check if key is matching
			if kr.SourceRegex.MatchString(key) {
				// Debug log
				logger.Debugf("Rewrite key : Key %s is matching the source rewrite %s", key, kr.SourceRegex.String())

				// Find submatches
				submatches := kr.SourceRegex.FindStringSubmatchIndex(key)

				// Check if there isn't any submatch
				if len(submatches) == 0 {
					// Debug log
					logger.Debugf("Rewrite key : Key %s have been changed to %s", key, kr.Target)

					return kr.Target, nil
				}

				// Check if target key rewrite is type REGEX
				if kr.TargetType == config.RegexTargetKeyRewriteTargetType {
					// Create result
					result := []byte{}
					// Replace matches in target
					result = kr.SourceRegex.ExpandString(result, kr.Target, key, submatches)

					// Debug log
					logger.Debugf("Rewrite key : Key %s have been changed to %s", key, string(result))

					// Return result
					return string(result), nil
				}

				// Template case

				// Initialize variable
				targetTplHelpers := make([]*config.TargetHelperConfigItem, 0)
				// Check if helpers have been declared in target configuration
				if bri.targetCfg.Templates != nil && bri.targetCfg.Templates.Helpers != nil {
					// Save target template helpers
					targetTplHelpers = bri.targetCfg.Templates.Helpers
				}

				// Load all helpers
				helpersString, err := templateutils.LoadAllHelpersContent(
					ctx,
					bri.LoadFileContent,
					targetTplHelpers,
					bri.generalHelpers,
				)
				// Check error
				if err != nil {
					return "", err
				}

				// Create template
				tpl := helpersString + "\n" + kr.Target

				// Get user from context
				user := models.GetAuthenticatedUserFromContext(ctx)
				// Get response handler
				resHan := responsehandler.GetResponseHandlerFromContext(ctx)

				// Execute template
				buf, err := templateutils.ExecuteTemplate(tpl, &targetKeyRewriteData{
					Request: resHan.GetRequest(),
					User:    user,
					Target:  bri.targetCfg,
					Key:     key,
				})
				// Check error
				if err != nil {
					return "", err
				}

				// Get string from buffer
				str := buf.String()
				// Remove all new lines
				str = utils.NewLineMatcherRegex.ReplaceAllString(str, "")
				// Trim spaces
				str = strings.TrimSpace(str)

				// Debug log
				logger.Debugf("Rewrite key : Key %s have been changed to %s", key, str)

				return str, nil
			}
		}
	}

	// Default case is returning the input key
	return key, nil
}

// Proxy GET requests.
func (bri *bucketReqImpl) Get(ctx context.Context, input *GetInput) {
	bri.internalGetOrHead(ctx, input, false)
}

// Proxy HEAD requests.
func (bri *bucketReqImpl) Head(ctx context.Context, input *GetInput) {
	bri.internalGetOrHead(ctx, input, true)
}

func (bri *bucketReqImpl) internalGetOrHead(ctx context.Context, input *GetInput, isHeadReq bool) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Generate start key
	key := bri.generateStartKey(input.RequestPath)
	// Manage key rewrite
	key, err := bri.manageKeyRewrite(ctx, key)
	// Check error
	if err != nil {
		resHan.InternalServerError(bri.LoadFileContent, err)
		// Stop
		return
	}

	// Check that the path ends with a / for a directory listing or the main path special case (empty path)
	if strings.HasSuffix(input.RequestPath, "/") || input.RequestPath == "" {
		bri.manageGetFolder(ctx, key, input, isHeadReq)
		// Stop
		return
	}

	// Get or Head object case

	// Check if it is a HEAD request or if it is asked to redirect to signed url
	if isHeadReq || bri.targetCfg.Actions != nil &&
		bri.targetCfg.Actions.GET != nil &&
		bri.targetCfg.Actions.GET.Config != nil &&
		bri.targetCfg.Actions.GET.Config.RedirectToSignedURL {
		// Get S3 client
		s3cl := bri.s3ClientManager.
			GetClientForTarget(bri.targetCfg.Name)
		// Head file in bucket
		headOutput, hInfo, err2 := s3cl.HeadObject(ctx, key)
		// Check if there is an error
		if err2 != nil {
			// Save error
			err = err2
		} else if headOutput != nil {
			// File found
			// Check head request
			if isHeadReq {
				err = bri.answerHead(ctx, input, headOutput, hInfo)
			} else {
				// Redirect to signed url
				err = bri.redirectToSignedURL(ctx, key, input)
			}
		}
	} else {
		// Stream object
		err = bri.streamFileForResponse(ctx, key, input)
	}

	// Check error
	if err != nil {
		// Check if error is a not found error
		//nolint: gocritic // Don't want a switch
		if errors.Is(err, s3client.ErrNotFound) {
			// Test that redirect with trailing slash isn't asked and possible on this request
			if bri.targetCfg.Actions != nil && bri.targetCfg.Actions.GET != nil &&
				bri.targetCfg.Actions.GET.Config != nil &&
				bri.targetCfg.Actions.GET.Config.RedirectWithTrailingSlashForNotFoundFile &&
				!strings.HasSuffix(input.RequestPath, "/") {
				// Redirect with trailing slash
				resHan.RedirectWithTrailingSlash()

				return
			}
			// Not found
			resHan.NotFoundError(bri.LoadFileContent)
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
		resHan.InternalServerError(bri.LoadFileContent, err)
		// Stop
		return
	}
}

func (bri *bucketReqImpl) manageGetFolder(ctx context.Context, key string, input *GetInput, isHeadReq bool) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Check if index document is activated
	if bri.targetCfg.Actions != nil && bri.targetCfg.Actions.GET != nil &&
		bri.targetCfg.Actions.GET.Config != nil &&
		bri.targetCfg.Actions.GET.Config.IndexDocument != "" {
		// Create index key path
		indexKey := path.Join(key, bri.targetCfg.Actions.GET.Config.IndexDocument)
		// Head index file in bucket
		headOutput, hInfo, err := bri.s3ClientManager.
			GetClientForTarget(bri.targetCfg.Name).
			HeadObject(ctx, indexKey)
		// Check if error exists and not a not found error
		if err != nil && !errors.Is(err, s3client.ErrNotFound) {
			// Manage error response
			resHan.InternalServerError(bri.LoadFileContent, err)
			// Stop
			return
		}
		// Check that we found the file
		if headOutput != nil {
			// Check if it is head request
			if isHeadReq { //nolint:gocritic // Ignore this
				// Answer with head
				err = bri.answerHead(ctx, input, headOutput, hInfo)
			} else if bri.targetCfg.Actions.GET.Config.RedirectToSignedURL { // Check if it is asked to redirect to signed url
				// Redirect to signed url
				err = bri.redirectToSignedURL(ctx, indexKey, input)
			} else {
				// Get data
				err = bri.streamFileForResponse(ctx, headOutput.Key, input)
			}
			// Check if error exists
			if err != nil {
				// Check if error is a not found error
				//nolint: gocritic // Don't want a switch
				if errors.Is(err, s3client.ErrNotFound) {
					// Not found
					resHan.NotFoundError(bri.LoadFileContent)

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
				resHan.InternalServerError(bri.LoadFileContent, err)
				// Stop
				return
			}
			// Stop here because no error are present
			return
		}
	}

	// Check if list folders is disabled
	if bri.targetCfg.Actions != nil && bri.targetCfg.Actions.GET != nil &&
		bri.targetCfg.Actions.GET.Config != nil &&
		bri.targetCfg.Actions.GET.Config.DisableListing {
		// Answer directly
		resHan.FoldersFilesList(
			bri.LoadFileContent,
			make([]*responsehandlermodels.Entry, 0),
		)

		// Stop
		return
	}

	// Directory listing case
	s3Entries, info, err := bri.s3ClientManager.
		GetClientForTarget(bri.targetCfg.Name).
		ListFilesAndDirectories(ctx, key)
	// Check error
	if err != nil {
		resHan.InternalServerError(bri.LoadFileContent, err)
		// Stop
		return
	}

	if isHeadReq {
		// Send hook
		bri.webhookManager.ManageHEADHooks(
			ctx,
			bri.targetCfg.Name,
			input.RequestPath,
			&webhook.HeadInputMetadata{
				IfModifiedSince:   input.IfModifiedSince,
				IfMatch:           input.IfMatch,
				IfNoneMatch:       input.IfNoneMatch,
				IfUnmodifiedSince: input.IfUnmodifiedSince,
			},
			&webhook.S3Metadata{
				Bucket:     info.Bucket,
				Region:     info.Region,
				S3Endpoint: info.S3Endpoint,
				Key:        info.Key,
			},
		)
	} else {
		// Send hook
		bri.webhookManager.ManageGETHooks(
			ctx,
			bri.targetCfg.Name,
			input.RequestPath,
			&webhook.GetInputMetadata{
				IfModifiedSince:   input.IfModifiedSince,
				IfMatch:           input.IfMatch,
				IfNoneMatch:       input.IfNoneMatch,
				IfUnmodifiedSince: input.IfUnmodifiedSince,
				Range:             input.Range,
			},
			&webhook.S3Metadata{
				Bucket:     info.Bucket,
				Region:     info.Region,
				S3Endpoint: info.S3Endpoint,
				Key:        info.Key,
			},
		)
	}

	// Transform entries in entry with path objects
	bucketRootPrefixKey := bri.targetCfg.Bucket.GetRootPrefix()
	entries := transformS3Entries(s3Entries, bri, bucketRootPrefixKey)

	// Answer
	resHan.FoldersFilesList(
		bri.LoadFileContent,
		entries,
	)
}

// Put proxy PUT requests.
func (bri *bucketReqImpl) Put(ctx context.Context, inp *PutInput) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Generate start key
	key := bri.generateStartKey(inp.RequestPath)
	// Add / at the end if not present
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	// Add filename at the end of key
	key += inp.Filename
	// Manage key rewrite
	key, err := bri.manageKeyRewrite(ctx, key)
	// Check error
	if err != nil {
		resHan.InternalServerError(bri.LoadFileContent, err)
		// Stop
		return
	}

	// Create input
	input := &s3client.PutInput{
		Key:         key,
		Body:        inp.Body,
		ContentType: inp.ContentType,
		ContentSize: inp.ContentSize,
	}

	// Check if post actions configuration exists
	if bri.targetCfg.Actions.PUT != nil &&
		bri.targetCfg.Actions.PUT.Config != nil {
		// Check if system metadata is defined
		if bri.targetCfg.Actions.PUT.Config.SystemMetadata != nil {
			// Manage cache control
			if bri.targetCfg.Actions.PUT.Config.SystemMetadata.CacheControl != "" {
				// Execute template
				val, err2 := bri.tplPutData(ctx, inp, key, bri.targetCfg.Actions.PUT.Config.SystemMetadata.CacheControl)
				// Check error
				if err2 != nil {
					resHan.InternalServerError(bri.LoadFileContent, err2)

					return
				}
				// Check if value is empty or not
				if val != "" {
					// Store
					input.CacheControl = val
				}
			}
			// Manage content disposition
			if bri.targetCfg.Actions.PUT.Config.SystemMetadata.ContentDisposition != "" {
				// Execute template
				val, err2 := bri.tplPutData(ctx, inp, key, bri.targetCfg.Actions.PUT.Config.SystemMetadata.ContentDisposition)
				// Check error
				if err2 != nil {
					resHan.InternalServerError(bri.LoadFileContent, err2)

					return
				}
				// Check if value is empty or not
				if val != "" {
					// Store
					input.ContentDisposition = val
				}
			}
			// Manage content encoding
			if bri.targetCfg.Actions.PUT.Config.SystemMetadata.ContentEncoding != "" {
				// Execute template
				val, err2 := bri.tplPutData(ctx, inp, key, bri.targetCfg.Actions.PUT.Config.SystemMetadata.ContentEncoding)
				// Check error
				if err2 != nil {
					resHan.InternalServerError(bri.LoadFileContent, err2)

					return
				}
				// Check if value is empty or not
				if val != "" {
					// Store
					input.ContentEncoding = val
				}
			}
			// Manage content language
			if bri.targetCfg.Actions.PUT.Config.SystemMetadata.ContentLanguage != "" {
				// Execute template
				val, err2 := bri.tplPutData(ctx, inp, key, bri.targetCfg.Actions.PUT.Config.SystemMetadata.ContentLanguage)
				// Check error
				if err2 != nil {
					resHan.InternalServerError(bri.LoadFileContent, err2)

					return
				}
				// Check if value is empty or not
				if val != "" {
					// Store
					input.ContentLanguage = val
				}
			}
			// Manage content language
			if bri.targetCfg.Actions.PUT.Config.SystemMetadata.Expires != "" {
				// Execute template
				val, err2 := bri.tplPutData(ctx, inp, key, bri.targetCfg.Actions.PUT.Config.SystemMetadata.Expires)
				// Check error
				if err2 != nil {
					resHan.InternalServerError(bri.LoadFileContent, err2)

					return
				}
				// Check if value is empty or not
				if val != "" {
					// Parse
					d, err3 := time.Parse(time.RFC3339, val)
					// Check error
					if err3 != nil {
						resHan.InternalServerError(bri.LoadFileContent, errors.WithStack(err3))

						return
					}
					// Store
					input.Expires = &d
				}
			}
		}

		// Check if metadata is configured in target configuration
		if bri.targetCfg.Actions.PUT.Config.Metadata != nil {
			// Store templated data
			metadata := map[string]string{}

			// Render templates
			for k, v := range bri.targetCfg.Actions.PUT.Config.Metadata {
				// Execute template
				val, err2 := bri.tplPutData(ctx, inp, key, v)
				// Check error
				if err2 != nil {
					resHan.InternalServerError(bri.LoadFileContent, err2)

					return
				}
				// Check if value is empty or not
				if val != "" {
					// Store
					metadata[k] = val
				}
			}

			// Store all metadata
			input.Metadata = metadata
		}

		// Check if storage class is present in target configuration
		if bri.targetCfg.Actions.PUT.Config.StorageClass != "" {
			// Execute template
			val, err2 := bri.tplPutData(ctx, inp, key, bri.targetCfg.Actions.PUT.Config.StorageClass)
			// Check error
			if err2 != nil {
				resHan.InternalServerError(bri.LoadFileContent, err2)

				return
			}
			// Check if value is empty or not
			if val != "" {
				// Store
				input.StorageClass = val
			}
		}

		// Check if allow override is enabled
		if !bri.targetCfg.Actions.PUT.Config.AllowOverride {
			// Need to check if file already exists
			headOutput, _, err2 := bri.s3ClientManager.
				GetClientForTarget(bri.targetCfg.Name).
				HeadObject(ctx, key)
			// Check if error is not found if exists
			if err2 != nil && !errors.Is(err2, s3client.ErrNotFound) {
				resHan.InternalServerError(bri.LoadFileContent, err2)
				// Stop
				return
			}
			// Check if file exists
			if headOutput != nil {
				// Create error
				err2 := fmt.Errorf("file detected on path %s for PUT request and override isn't allowed", key)
				// Response
				resHan.ForbiddenError(bri.LoadFileContent, err2)
				// Stop
				return
			}
		}
	}

	// Put file
	info, err := bri.s3ClientManager.
		GetClientForTarget(bri.targetCfg.Name).
		PutObject(ctx, input)
	// Check error
	if err != nil {
		resHan.InternalServerError(bri.LoadFileContent, err)
		// Stop
		return
	}

	// Send hook
	bri.webhookManager.ManagePUTHooks(
		ctx,
		bri.targetCfg.Name,
		inp.RequestPath,
		&webhook.PutInputMetadata{
			Filename:    inp.Filename,
			ContentType: inp.ContentType,
			ContentSize: inp.ContentSize,
		},
		&webhook.S3Metadata{
			Bucket:     info.Bucket,
			Region:     info.Region,
			S3Endpoint: info.S3Endpoint,
			Key:        info.Key,
		},
	)

	// Answer
	resHan.Put(
		bri.LoadFileContent,
		&responsehandlermodels.PutInput{
			Key:          key,
			ContentType:  inp.ContentType,
			ContentSize:  inp.ContentSize,
			Metadata:     input.Metadata,
			StorageClass: input.StorageClass,
			Filename:     inp.Filename,
		},
	)
}

func (*bucketReqImpl) tplPutData(ctx context.Context, inp *PutInput, key, tplStr string) (string, error) {
	// Execute template
	buf, err := templateutils.ExecuteTemplate(tplStr, &PutData{
		User:  models.GetAuthenticatedUserFromContext(ctx),
		Input: inp,
		Key:   key,
	})

	// Check error
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Store value
	val := buf.String()
	// Remove all new lines
	val = utils.NewLineMatcherRegex.ReplaceAllString(val, "")
	// Check if value is empty or not
	if val != "" {
		// Store
		return val, nil
	}

	// Default
	return "", nil
}

// Delete will delete object in S3.
func (bri *bucketReqImpl) Delete(ctx context.Context, requestPath string) {
	// Get response handler
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Generate start key
	key := bri.generateStartKey(requestPath)
	// Manage key rewrite
	key, err := bri.manageKeyRewrite(ctx, key)
	// Check error
	if err != nil {
		resHan.InternalServerError(bri.LoadFileContent, err)
		// Stop
		return
	}

	// Check that the path ends with a / for a directory or the main path special case (empty path)
	if strings.HasSuffix(requestPath, "/") || requestPath == "" {
		resHan.InternalServerError(bri.LoadFileContent, ErrRemovalFolder)
		// Stop
		return
	}

	// Delete object in S3
	info, err := bri.s3ClientManager.
		GetClientForTarget(bri.targetCfg.Name).
		DeleteObject(ctx, key)
	// Check if error exists
	if err != nil {
		resHan.InternalServerError(bri.LoadFileContent, err)
		// Stop
		return
	}

	// Send hook
	bri.webhookManager.ManageDELETEHooks(
		ctx,
		bri.targetCfg.Name,
		requestPath,
		&webhook.S3Metadata{
			Bucket:     info.Bucket,
			Region:     info.Region,
			S3Endpoint: info.S3Endpoint,
			Key:        info.Key,
		},
	)

	// Answer
	resHan.Delete(
		bri.LoadFileContent,
		&responsehandlermodels.DeleteInput{
			Key: key,
		},
	)
}

func transformS3Entries(
	s3Entries []*s3client.ListElementOutput,
	rctx *bucketReqImpl,
	bucketRootPrefixKey string,
) []*responsehandlermodels.Entry {
	// Prepare result
	entries := make([]*responsehandlermodels.Entry, 0)
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
		entries = append(entries, &responsehandlermodels.Entry{
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

func (bri *bucketReqImpl) LoadFileContent(ctx context.Context, fpath string) (string, error) {
	// Get object from s3
	objOutput, _, err := bri.s3ClientManager.GetClientForTarget(bri.targetCfg.Name).GetObject(ctx, &s3client.GetInput{
		Key: fpath,
	})
	// Check error
	if err != nil {
		return "", err
	}

	// Read all body
	bb, err := io.ReadAll(objOutput.Body)
	// Check error
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Transform it to string and return
	return string(bb), nil
}

func (bri *bucketReqImpl) redirectToSignedURL(ctx context.Context, key string, input *GetInput) error {
	// Get response handler from context
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)
	// Get signed url
	url, err := bri.s3ClientManager.
		GetClientForTarget(bri.targetCfg.Name).
		GetObjectSignedURL(
			ctx,
			&s3client.GetInput{
				Key:               key,
				IfModifiedSince:   input.IfModifiedSince,
				IfMatch:           input.IfMatch,
				IfNoneMatch:       input.IfNoneMatch,
				IfUnmodifiedSince: input.IfUnmodifiedSince,
				Range:             input.Range,
			},
			bri.targetCfg.Actions.GET.Config.SignedURLExpiration,
		)
	// Check error
	if err != nil {
		return err
	}
	// Redirect
	resHan.RedirectTo(url)

	return nil
}

func (bri *bucketReqImpl) answerHead(
	ctx context.Context,
	input *GetInput,
	hOutput *s3client.HeadOutput,
	info *s3client.ResultInfo,
) error {
	// Get response handler from context
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Send hook
	bri.webhookManager.ManageHEADHooks(
		ctx,
		bri.targetCfg.Name,
		input.RequestPath,
		&webhook.HeadInputMetadata{
			IfModifiedSince:   input.IfModifiedSince,
			IfMatch:           input.IfMatch,
			IfNoneMatch:       input.IfNoneMatch,
			IfUnmodifiedSince: input.IfUnmodifiedSince,
		},
		&webhook.S3Metadata{
			Bucket:     info.Bucket,
			Region:     info.Region,
			S3Endpoint: info.S3Endpoint,
			Key:        info.Key,
		},
	)

	// Transform input
	inp := &responsehandlermodels.StreamInput{
		CacheControl:       hOutput.CacheControl,
		Expires:            hOutput.Expires,
		ContentDisposition: hOutput.ContentDisposition,
		ContentEncoding:    hOutput.ContentEncoding,
		ContentLanguage:    hOutput.ContentLanguage,
		ContentLength:      hOutput.ContentLength,
		ContentType:        hOutput.ContentType,
		ETag:               hOutput.ETag,
		LastModified:       hOutput.LastModified,
		Metadata:           hOutput.Metadata,
	}

	// Stream
	return resHan.StreamFile(bri.LoadFileContent, inp)
}

func (bri *bucketReqImpl) streamFileForResponse(ctx context.Context, key string, input *GetInput) error {
	// Get response handler from context
	resHan := responsehandler.GetResponseHandlerFromContext(ctx)

	// Get object from s3
	objOutput, info, err := bri.s3ClientManager.
		GetClientForTarget(bri.targetCfg.Name).
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

	// Defer body closing
	defer objOutput.Body.Close()

	// Transform input
	inp := &responsehandlermodels.StreamInput{
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
		Metadata:           objOutput.Metadata,
	}

	// Stream
	err = resHan.StreamFile(bri.LoadFileContent, inp)
	// Check error
	if err != nil {
		return err
	}

	// Send hook
	bri.webhookManager.ManageGETHooks(
		ctx,
		bri.targetCfg.Name,
		input.RequestPath,
		&webhook.GetInputMetadata{
			IfModifiedSince:   input.IfModifiedSince,
			IfMatch:           input.IfMatch,
			IfNoneMatch:       input.IfNoneMatch,
			IfUnmodifiedSince: input.IfUnmodifiedSince,
		},
		&webhook.S3Metadata{
			Bucket:     info.Bucket,
			Region:     info.Region,
			S3Endpoint: info.S3Endpoint,
			Key:        info.Key,
		},
	)

	// Default return
	return nil
}
