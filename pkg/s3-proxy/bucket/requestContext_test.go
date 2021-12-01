//go:build unit

package bucket

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	responsehandlermocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	s3clientmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook"
	wmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook/mocks"
	"github.com/stretchr/testify/assert"
)

func Test_transformS3Entries(t *testing.T) {
	now := time.Now()
	type args struct {
		s3Entries           []*s3client.ListElementOutput
		rctx                *requestContext
		bucketRootPrefixKey string
	}
	tests := []struct {
		name string
		args args
		want []*responsehandler.Entry
	}{
		{
			name: "Empty list",
			args: args{
				s3Entries:           []*s3client.ListElementOutput{},
				rctx:                &requestContext{},
				bucketRootPrefixKey: "prefix/",
			},
			want: []*responsehandler.Entry{},
		},
		{
			name: "List",
			args: args{
				s3Entries: []*s3client.ListElementOutput{
					{
						Type:         s3client.FileType,
						ETag:         "etag",
						Name:         "name",
						LastModified: now,
						Size:         300,
						Key:          "key",
					},
				},
				rctx: &requestContext{
					mountPath: "mount/",
				},
				bucketRootPrefixKey: "prefix/",
			},
			want: []*responsehandler.Entry{
				{
					Type:         s3client.FileType,
					ETag:         "etag",
					Name:         "name",
					LastModified: now,
					Size:         300,
					Key:          "key",
					Path:         "mount/key",
				},
			},
		},
		{
			name: "/ in bucket prefix key",
			args: args{
				s3Entries: []*s3client.ListElementOutput{
					{
						Type:         s3client.FileType,
						ETag:         "etag",
						Name:         "name",
						LastModified: now,
						Size:         300,
						Key:          "key",
					},
				},
				rctx: &requestContext{
					mountPath: "mount/",
				},
				bucketRootPrefixKey: "/",
			},
			want: []*responsehandler.Entry{
				{
					Type:         s3client.FileType,
					ETag:         "etag",
					Name:         "name",
					LastModified: now,
					Size:         300,
					Key:          "key",
					Path:         "mount/key",
				},
			},
		},
		{
			name: "/ in bucket prefix key and in mount path",
			args: args{
				s3Entries: []*s3client.ListElementOutput{
					{
						Type:         s3client.FileType,
						ETag:         "etag",
						Name:         "name",
						LastModified: now,
						Size:         300,
						Key:          "key",
					},
				},
				rctx: &requestContext{
					mountPath: "/",
				},
				bucketRootPrefixKey: "/",
			},
			want: []*responsehandler.Entry{
				{
					Type:         s3client.FileType,
					ETag:         "etag",
					Name:         "name",
					LastModified: now,
					Size:         300,
					Key:          "key",
					Path:         "/key",
				},
			},
		},
		{
			name: "/ in bucket prefix key and empty mount path",
			args: args{
				s3Entries: []*s3client.ListElementOutput{
					{
						Type:         s3client.FileType,
						ETag:         "etag",
						Name:         "name",
						LastModified: now,
						Size:         300,
						Key:          "key",
					},
				},
				rctx: &requestContext{
					mountPath: "",
				},
				bucketRootPrefixKey: "/",
			},
			want: []*responsehandler.Entry{
				{
					Type:         s3client.FileType,
					ETag:         "etag",
					Name:         "name",
					LastModified: now,
					Size:         300,
					Key:          "key",
					Path:         "key",
				},
			},
		},
		{
			name: "ensure end / is added on folder type",
			args: args{
				s3Entries: []*s3client.ListElementOutput{
					{
						Type:         s3client.FolderType,
						ETag:         "etag",
						Name:         "name",
						LastModified: now,
						Size:         300,
						Key:          "key",
					},
				},
				rctx: &requestContext{
					mountPath: "mount/",
				},
				bucketRootPrefixKey: "/",
			},
			want: []*responsehandler.Entry{
				{
					Type:         s3client.FolderType,
					ETag:         "etag",
					Name:         "name",
					LastModified: now,
					Size:         300,
					Key:          "key",
					Path:         "mount/key/",
				},
			},
		},
		{
			name: "ensure end / isn't added on folder type",
			args: args{
				s3Entries: []*s3client.ListElementOutput{
					{
						Type:         s3client.FolderType,
						ETag:         "etag",
						Name:         "name",
						LastModified: now,
						Size:         300,
						Key:          "key/",
					},
				},
				rctx: &requestContext{
					mountPath: "mount/",
				},
				bucketRootPrefixKey: "/",
			},
			want: []*responsehandler.Entry{
				{
					Type:         s3client.FolderType,
					ETag:         "etag",
					Name:         "name",
					LastModified: now,
					Size:         300,
					Key:          "key/",
					Path:         "mount/key/",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformS3Entries(tt.args.s3Entries, tt.args.rctx, tt.args.bucketRootPrefixKey)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_requestContext_Delete(t *testing.T) {
	type responseHandlerInternalServerErrorMockResult struct {
		input2 error
		times  int
	}
	type s3ClientDeleteObjectMockResult struct {
		input2 string
		res    *s3client.ResultInfo
		err    error
		times  int
	}
	type webhookManagerManageDeleteHooksMockResult struct {
		input2 string
		input3 string
		input4 *webhook.S3Metadata
		times  int
	}
	type fields struct {
		targetCfg *config.TargetConfig
		mountPath string
	}
	type args struct {
		requestPath string
	}
	tests := []struct {
		name                                         string
		fields                                       fields
		args                                         args
		s3clManagerClientForTargetMockInput          string
		responseHandlerNoContentMockResultTimes      int
		responseHandlerInternalServerErrorMockResult responseHandlerInternalServerErrorMockResult
		s3ClientDeleteObjectMockResult               s3ClientDeleteObjectMockResult
		webhookManagerManageDeleteHooksMockResult    webhookManagerManageDeleteHooksMockResult
	}{
		{
			name: "Can't delete a directory with empty request path",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				mountPath: "/mount",
			},
			args: args{requestPath: ""},
			responseHandlerInternalServerErrorMockResult: responseHandlerInternalServerErrorMockResult{
				input2: ErrRemovalFolder,
				times:  1,
			},
		},
		{
			name: "Can't delete a directory",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				mountPath: "/mount",
			},
			args: args{requestPath: "/directory/"},
			responseHandlerInternalServerErrorMockResult: responseHandlerInternalServerErrorMockResult{
				input2: ErrRemovalFolder,
				times:  1,
			},
		},
		{
			name: "Can't delete file because of error",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				mountPath: "/mount",
			},
			args:                                args{requestPath: "/file"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "/file",
				err:    errors.New("fake error"),
				times:  1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerInternalServerErrorMockResult{
				input2: errors.New("fake error"),
				times:  1,
			},
		},
		{
			name: "Delete file succeed",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				mountPath: "/mount",
			},
			args:                                args{requestPath: "/file"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "/file",
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageDeleteHooksMockResult: webhookManagerManageDeleteHooksMockResult{
				input2: "bucket",
				input3: "/file",
				input4: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "Delete succeed with rewrite key",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "/"},
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile("^/file$"),
						Target:      "/fake/file2",
					}},
				},
				mountPath: "/mount",
			},
			args:                                args{requestPath: "/file"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "/fake/file2",
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageDeleteHooksMockResult: webhookManagerManageDeleteHooksMockResult{
				input2: "bucket",
				input3: "/file",
				input4: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerNoContentMockResultTimes: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)

			// Create mocks
			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)

			// Create context
			ctx := context.TODO()

			// Add response handler to context
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)

			// Add logger to context
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())

			resHandlerMock.EXPECT().
				InternalServerError(gomock.Any(), tt.responseHandlerInternalServerErrorMockResult.input2).
				Times(tt.responseHandlerInternalServerErrorMockResult.times)
			resHandlerMock.EXPECT().NoContent().Times(tt.responseHandlerNoContentMockResultTimes)

			s3ClientMock.EXPECT().
				DeleteObject(ctx, tt.s3ClientDeleteObjectMockResult.input2).
				Return(
					tt.s3ClientDeleteObjectMockResult.res,
					tt.s3ClientDeleteObjectMockResult.err,
				).
				Times(tt.s3ClientDeleteObjectMockResult.times)

			s3clManagerMock.EXPECT().
				GetClientForTarget(tt.s3clManagerClientForTargetMockInput).
				AnyTimes().
				Return(s3ClientMock)

			webhookManagerMock.EXPECT().
				ManageDELETEHooks(
					ctx,
					tt.webhookManagerManageDeleteHooksMockResult.input2,
					tt.webhookManagerManageDeleteHooksMockResult.input3,
					tt.webhookManagerManageDeleteHooksMockResult.input4,
				).
				Times(tt.webhookManagerManageDeleteHooksMockResult.times)

			rctx := &requestContext{
				s3ClientManager: s3clManagerMock,
				webhookManager:  webhookManagerMock,
				targetCfg:       tt.fields.targetCfg,
				mountPath:       tt.fields.mountPath,
			}
			rctx.Delete(ctx, tt.args.requestPath)
		})
	}
}

func Test_requestContext_Put(t *testing.T) {
	type responseHandlerErrorsMockResult struct {
		input2 error
		times  int
	}
	type s3ClientPutObjectMockResult struct {
		input2 *s3client.PutInput
		res    *s3client.ResultInfo
		err    error
		times  int
	}
	type s3ClientHeadObjectMockResult struct {
		input2 string
		err    error
		res    *s3client.HeadOutput
		times  int
	}
	type webhookManagerManagePutHooksMockResult struct {
		input2 string
		input3 string
		input4 *webhook.PutInputMetadata
		input5 *webhook.S3Metadata
		times  int
	}
	type fields struct {
		targetCfg *config.TargetConfig
		mountPath string
	}
	type args struct {
		inp *PutInput
	}
	tests := []struct {
		name                                         string
		fields                                       fields
		args                                         args
		responseHandlerInternalServerErrorMockResult responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult      responseHandlerErrorsMockResult
		responseHandlerNoContentMockResultTimes      int
		s3clManagerClientForTargetMockInput          string
		s3ClientHeadObjectMockResult                 s3ClientHeadObjectMockResult
		s3ClientPutObjectMockResult                  s3ClientPutObjectMockResult
		webhookManagerManagePutHooksMockResult       webhookManagerManagePutHooksMockResult
	}{
		{
			name: "should fail when put object failed and no put configuration exists",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Bucket:  &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				err: errors.New("test"),
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
				},
				times: 1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("test"),
				times:  1,
			},
		},
		{
			name: "should fail when put object failed and put configuration exists with allow override",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"testkey": "testvalue",
								},
								StorageClass:  "storage-class",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				err: errors.New("test"),
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"testkey": "testvalue",
					},
					StorageClass: "storage-class",
				},
				times: 1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("test"),
				times:  1,
			},
		},
		{
			name: "should be ok with allow override",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"testkey": "testvalue",
								},
								StorageClass:  "storage-class",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
					ContentSize: 1,
				},
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					ContentSize: 1,
					Metadata: map[string]string{
						"testkey": "testvalue",
					},
					StorageClass: "storage-class",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
					ContentSize: 1,
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be failed when head object failed",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								AllowOverride: false,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/test/file",
				err:    errors.New("test"),
				times:  1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("test"),
				times:  1,
			},
		},
		{
			name: "should be failed when head object result that file exists",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								AllowOverride: false,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("file detected on path /test/file for PUT request and override isn't allowed"),
				times:  1,
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/test/file",
				res:    &s3client.HeadOutput{Key: "/test/file"},
				times:  1,
			},
		},
		{
			name: "should be ok when head object return that file doesn't exist",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								AllowOverride: false,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/test/file",
				res:    nil,
				times:  1,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be ok with allow override and key rewrite",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile("/test/file"),
						Target:      "/test1/test2/file",
					}},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"testkey": "testvalue",
								},
								StorageClass:  "storage-class",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				times: 0,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test1/test2/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"testkey": "testvalue",
					},
					StorageClass: "storage-class",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be ok to do templating on metadata",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"fixed":   "fixed",
									"testkey": "{{ .Key }}",
									"tpl":     "{{ .Key }} - {{ .Input.ContentType }}",
								},
								StorageClass:  "storage-class",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				times: 0,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"fixed":   "fixed",
						"testkey": "/test/file",
						"tpl":     "/test/file - content-type",
					},
					StorageClass: "storage-class",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be ok to flush empty metadata",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"fixed":   "fixed",
									"testkey": "",
								},
								StorageClass:  "storage-class",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				times: 0,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"fixed": "fixed",
					},
					StorageClass: "storage-class",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be ok to do templating on metadata and remove new lines",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"fixed":   "fixed",
									"testkey": "{{ .Key }}",
									"tpl": `
{{ .Key }} - {{ .Input.ContentType }}
`,
								},
								StorageClass:  "storage-class",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				times: 0,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"fixed":   "fixed",
						"testkey": "/test/file",
						"tpl":     "/test/file - content-type",
					},
					StorageClass: "storage-class",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be ok to do templating on storage class",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"fixed": "fixed",
								},
								StorageClass:  "{{ .Key }} - {{ .Input.ContentType }}",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				times: 0,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"fixed": "fixed",
					},
					StorageClass: "/test/file - content-type",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be ok to flush storage class",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"fixed": "fixed",
								},
								StorageClass:  "",
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				times: 0,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"fixed": "fixed",
					},
					StorageClass: "",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
		{
			name: "should be ok to do templating on storage class and remove new lines",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "name",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								Metadata: map[string]string{
									"fixed": "fixed",
								},
								StorageClass: `
{{ .Key }} - {{ .Input.ContentType }}
`,
								AllowOverride: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				times: 0,
			},
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/test/file",
					ContentType: "content-type",
					Metadata: map[string]string{
						"fixed": "fixed",
					},
					StorageClass: "/test/file - content-type",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "name",
				input3: "/test",
				input4: &webhook.PutInputMetadata{
					Filename:    "file",
					ContentType: "content-type",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3clManagerClientForTargetMockInput:     "name",
			responseHandlerNoContentMockResultTimes: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)

			// Create mocks
			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)

			// Create context
			ctx := context.TODO()

			// Add response handler to context
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)

			// Add logger to context
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())

			resHandlerMock.EXPECT().
				InternalServerError(gomock.Any(), tt.responseHandlerInternalServerErrorMockResult.input2).
				Times(tt.responseHandlerInternalServerErrorMockResult.times)
			resHandlerMock.EXPECT().
				ForbiddenError(gomock.Any(), tt.responseHandlerForbiddenErrorMockResult.input2).
				Times(tt.responseHandlerForbiddenErrorMockResult.times)
			resHandlerMock.EXPECT().NoContent().Times(tt.responseHandlerNoContentMockResultTimes)

			s3ClientMock.EXPECT().
				HeadObject(ctx, tt.s3ClientHeadObjectMockResult.input2).
				Return(
					tt.s3ClientHeadObjectMockResult.res,
					tt.s3ClientHeadObjectMockResult.err,
				).
				Times(tt.s3ClientHeadObjectMockResult.times)
			s3ClientMock.EXPECT().
				PutObject(ctx, tt.s3ClientPutObjectMockResult.input2).
				Return(
					tt.s3ClientPutObjectMockResult.res,
					tt.s3ClientPutObjectMockResult.err,
				).
				Times(tt.s3ClientPutObjectMockResult.times)

			s3clManagerMock.EXPECT().
				GetClientForTarget(tt.s3clManagerClientForTargetMockInput).
				AnyTimes().
				Return(s3ClientMock)

			webhookManagerMock.EXPECT().
				ManagePUTHooks(
					ctx,
					tt.webhookManagerManagePutHooksMockResult.input2,
					tt.webhookManagerManagePutHooksMockResult.input3,
					tt.webhookManagerManagePutHooksMockResult.input4,
					tt.webhookManagerManagePutHooksMockResult.input5,
				).
				Times(
					tt.webhookManagerManagePutHooksMockResult.times,
				)

			rctx := &requestContext{
				s3ClientManager: s3clManagerMock,
				webhookManager:  webhookManagerMock,
				targetCfg:       tt.fields.targetCfg,
				mountPath:       tt.fields.mountPath,
			}
			rctx.Put(ctx, tt.args.inp)
		})
	}
}

func Test_requestContext_Get(t *testing.T) {
	fakeDate := time.Date(1990, time.December, 25, 1, 1, 1, 1, time.UTC)

	type responseHandlerErrorsMockResult struct {
		input2 error
		times  int
	}
	type responseHandlerStreamFileMockResult struct {
		input *responsehandler.StreamInput
		err   error
		times int
	}
	type responseHandlerFoldersFilesListMockResult struct {
		input2 []*responsehandler.Entry
		times  int
	}
	type s3ClientListFilesAndDirectoriesMockResult struct {
		input2 string
		res    []*s3client.ListElementOutput
		res2   *s3client.ResultInfo
		err    error
		times  int
	}
	type s3ClientHeadObjectMockResult struct {
		input2 string
		err    error
		res    *s3client.HeadOutput
		times  int
	}
	type s3ClientGetObjectMockResult struct {
		input2 *s3client.GetInput
		res    *s3client.GetOutput
		res2   *s3client.ResultInfo
		err    error
		times  int
	}
	type webhookManagerManageGetHooksMockResult struct {
		input2 string
		input3 string
		input4 *webhook.GetInputMetadata
		input5 *webhook.S3Metadata
		times  int
	}
	type fields struct {
		targetCfg *config.TargetConfig
		mountPath string
	}
	type args struct {
		input *GetInput
	}
	tests := []struct {
		name                                         string
		fields                                       fields
		args                                         args
		responseHandlerInternalServerErrorMockResult responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult      responseHandlerErrorsMockResult
		responseHandlerNotFoundErrorMockResult       responseHandlerErrorsMockResult
		responseHandlerStreamFileMockResult          responseHandlerStreamFileMockResult
		responseHandlerFoldersFilesListMockResult    responseHandlerFoldersFilesListMockResult
		responseHandlerNotModifiedTimes              int
		responseHandlerPreconditionFailedTimes       int
		s3ClientHeadObjectMockResult                 s3ClientHeadObjectMockResult
		s3ClientListFilesAndDirectoriesMockResult    s3ClientListFilesAndDirectoriesMockResult
		s3ClientGetObjectMockResult                  s3ClientGetObjectMockResult
		s3clManagerClientForTargetMockInput          string
		webhookManagerManageGetHooksMockResult       webhookManagerManageGetHooksMockResult
	}{
		{
			name: "should fail if list files and directories failed",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "/folder/",
				err:    errors.New("test"),
				times:  1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("test"),
				times:  1,
			},
		},
		{
			name: "should be ok to list files and directories",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{
					RequestPath:       "/folder/",
					IfModifiedSince:   &fakeDate,
					IfMatch:           "ifmatch",
					IfNoneMatch:       "ifnonematch",
					IfUnmodifiedSince: &fakeDate,
					Range:             "range",
				},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "/folder/",
				res: []*s3client.ListElementOutput{
					{
						Name:         "file1",
						Type:         "FILE",
						ETag:         "etag",
						LastModified: fakeDate,
						Size:         300,
						Key:          "/folder/file1",
					},
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/folder/",
				input4: &webhook.GetInputMetadata{
					IfModifiedSince:   &fakeDate,
					IfMatch:           "ifmatch",
					IfNoneMatch:       "ifnonematch",
					IfUnmodifiedSince: &fakeDate,
					Range:             "range",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerFoldersFilesListMockResult: responseHandlerFoldersFilesListMockResult{
				input2: []*responsehandler.Entry{{
					Type:         "FILE",
					ETag:         "etag",
					LastModified: fakeDate,
					Name:         "file1",
					Size:         300,
					Key:          "/folder/file1",
					Path:         "/mount/folder/file1",
				}},
				times: 1,
			},
		},
		{
			name: "should be ok to find and load index document",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{IndexDocument: "index.html"},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/folder/index.html",
				res: &s3client.HeadOutput{
					Type: "FILE",
					Key:  "/folder/index.html",
				},
				times: 1,
			},
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				res: &s3client.GetOutput{
					ContentType: "text/html; charset=utf-8",
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/folder/",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerStreamFileMockResult: responseHandlerStreamFileMockResult{
				input: &responsehandler.StreamInput{
					ContentType: "text/html; charset=utf-8",
				},
				times: 1,
			},
		},
		{
			name: "should return a 304 when S3 client return a 304 error",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{IndexDocument: "index.html"},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/folder/index.html",
				res: &s3client.HeadOutput{
					Type: "FILE",
					Key:  "/folder/index.html",
				},
				times: 1,
			},
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				err:   s3client.ErrNotModified,
				times: 1,
			},
			responseHandlerNotModifiedTimes: 1,
		},
		{
			name: "should return a 412 when S3 client return a 412 error",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{IndexDocument: "index.html"},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/folder/index.html",
				res: &s3client.HeadOutput{
					Type: "FILE",
					Key:  "/folder/index.html",
				},
				times: 1,
			},
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				err:   s3client.ErrPreconditionFailed,
				times: 1,
			},
			responseHandlerPreconditionFailedTimes: 1,
		},
		{
			name: "should be ok to not find index document when index document is enabled",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{IndexDocument: "index.html"},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "/folder/",
				res: []*s3client.ListElementOutput{
					{
						Name:         "file1",
						Type:         "FILE",
						ETag:         "etag",
						LastModified: fakeDate,
						Size:         300,
						Key:          "/folder/file1",
					},
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/folder/",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/folder/index.html",
				err:    s3client.ErrNotFound,
				times:  1,
			},
			responseHandlerFoldersFilesListMockResult: responseHandlerFoldersFilesListMockResult{
				input2: []*responsehandler.Entry{{
					Type:         "FILE",
					ETag:         "etag",
					LastModified: fakeDate,
					Name:         "file1",
					Size:         300,
					Key:          "/folder/file1",
					Path:         "/mount/folder/file1",
				}},
				times: 1,
			},
		},
		{
			name: "should fail to find and load index document with unknown error on head file",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{IndexDocument: "index.html"},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/folder/index.html",
				err:    errors.New("error"),
				times:  1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("error"),
				times:  1,
			},
		},
		{
			name: "should fail to find and load index document with not found error on get file",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{IndexDocument: "index.html"},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/folder/index.html",
				res: &s3client.HeadOutput{
					Type: "FILE",
					Key:  "/folder/index.html",
				},
				times: 1,
			},
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				err:   s3client.ErrNotFound,
				times: 1,
			},
			responseHandlerNotFoundErrorMockResult: responseHandlerErrorsMockResult{
				input2: nil,
				times:  1,
			},
		},
		{
			name: "should fail to find and load index document with error on get file",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{IndexDocument: "index.html"},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientHeadObjectMockResult: s3ClientHeadObjectMockResult{
				input2: "/folder/index.html",
				res: &s3client.HeadOutput{
					Type: "FILE",
					Key:  "/folder/index.html",
				},
				times: 1,
			},
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				err:   errors.New("test-error"),
				times: 1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("test-error"),
				times:  1,
			},
		},
		{
			name: "should be ok to get file",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				res: &s3client.GetOutput{
					ContentDisposition: "disposition",
					ContentType:        "type",
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/folder/index.html",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerStreamFileMockResult: responseHandlerStreamFileMockResult{
				input: &responsehandler.StreamInput{
					ContentDisposition: "disposition",
					ContentType:        "type",
				},
				times: 1,
			},
		},
		{
			name: "should return a 304 error when S3 return a 304 error",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				err:   s3client.ErrNotModified,
				times: 1,
			},
			responseHandlerNotModifiedTimes: 1,
		},
		{
			name: "should return a 412 error when S3 return a 412 error",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				err:   s3client.ErrPreconditionFailed,
				times: 1,
			},
			responseHandlerPreconditionFailedTimes: 1,
		},
		{
			name: "should fail to get file when not found error is raised",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				err:   s3client.ErrNotFound,
				times: 1,
			},
			responseHandlerNotFoundErrorMockResult: responseHandlerErrorsMockResult{
				times: 1,
			},
		},
		{
			name: "should fail to get file when error is raised",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/folder/index.html",
				},
				err:   errors.New("test-error"),
				times: 1,
			},
			responseHandlerInternalServerErrorMockResult: responseHandlerErrorsMockResult{
				input2: errors.New("test-error"),
				times:  1,
			},
		},
		{
			name: "should be ok to get file with key rewrite",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile(`^/folder/index\.html$`),
						Target:      "/fake/fake.html",
					}},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientGetObjectMockResult: s3ClientGetObjectMockResult{
				input2: &s3client.GetInput{
					Key: "/fake/fake.html",
				},
				res: &s3client.GetOutput{
					ContentType:     "type",
					ContentEncoding: "encoding",
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/folder/index.html",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerStreamFileMockResult: responseHandlerStreamFileMockResult{
				input: &responsehandler.StreamInput{
					ContentEncoding: "encoding",
					ContentType:     "type",
				},
				times: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)

			// Create mocks
			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)

			// Create context
			ctx := context.TODO()

			// Add response handler to context
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)

			// Add logger to context
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())

			resHandlerMock.EXPECT().
				InternalServerError(gomock.Any(), tt.responseHandlerInternalServerErrorMockResult.input2).
				Times(tt.responseHandlerInternalServerErrorMockResult.times)
			resHandlerMock.EXPECT().
				ForbiddenError(gomock.Any(), tt.responseHandlerForbiddenErrorMockResult.input2).
				Times(tt.responseHandlerForbiddenErrorMockResult.times)
			resHandlerMock.EXPECT().
				NotFoundError(gomock.Any()).
				Times(tt.responseHandlerNotFoundErrorMockResult.times)
			resHandlerMock.EXPECT().
				StreamFile(gomock.Any(), tt.responseHandlerStreamFileMockResult.input).
				Return(tt.responseHandlerStreamFileMockResult.err).
				Times(tt.responseHandlerStreamFileMockResult.times)
			resHandlerMock.EXPECT().
				NotModified().
				Times(tt.responseHandlerNotModifiedTimes)
			resHandlerMock.EXPECT().
				PreconditionFailed().
				Times(tt.responseHandlerPreconditionFailedTimes)
			resHandlerMock.EXPECT().
				FoldersFilesList(gomock.Any(), tt.responseHandlerFoldersFilesListMockResult.input2).
				Times(tt.responseHandlerFoldersFilesListMockResult.times)

			s3ClientMock.EXPECT().
				HeadObject(ctx, tt.s3ClientHeadObjectMockResult.input2).
				Return(
					tt.s3ClientHeadObjectMockResult.res,
					tt.s3ClientHeadObjectMockResult.err,
				).
				Times(tt.s3ClientHeadObjectMockResult.times)
			s3ClientMock.EXPECT().
				GetObject(ctx, tt.s3ClientGetObjectMockResult.input2).
				Return(
					tt.s3ClientGetObjectMockResult.res,
					tt.s3ClientGetObjectMockResult.res2,
					tt.s3ClientGetObjectMockResult.err,
				).
				Times(tt.s3ClientGetObjectMockResult.times)
			s3ClientMock.EXPECT().
				ListFilesAndDirectories(ctx, tt.s3ClientListFilesAndDirectoriesMockResult.input2).
				Return(
					tt.s3ClientListFilesAndDirectoriesMockResult.res,
					tt.s3ClientListFilesAndDirectoriesMockResult.res2,
					tt.s3ClientListFilesAndDirectoriesMockResult.err,
				).
				Times(tt.s3ClientListFilesAndDirectoriesMockResult.times)

			s3clManagerMock.EXPECT().
				GetClientForTarget(tt.s3clManagerClientForTargetMockInput).
				AnyTimes().
				Return(s3ClientMock)

			webhookManagerMock.EXPECT().
				ManageGETHooks(
					ctx,
					tt.webhookManagerManageGetHooksMockResult.input2,
					tt.webhookManagerManageGetHooksMockResult.input3,
					tt.webhookManagerManageGetHooksMockResult.input4,
					tt.webhookManagerManageGetHooksMockResult.input5,
				).
				Times(
					tt.webhookManagerManageGetHooksMockResult.times,
				)

			rctx := &requestContext{
				s3ClientManager: s3clManagerMock,
				webhookManager:  webhookManagerMock,
				targetCfg:       tt.fields.targetCfg,
				mountPath:       tt.fields.mountPath,
			}
			rctx.Get(ctx, tt.args.input)
		})
	}
}

func Test_requestContext_manageKeyRewrite(t *testing.T) {
	type fields struct {
		targetCfg *config.TargetConfig
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "no key rewrite list",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: nil,
				},
			},
			args: args{key: "/input"},
			want: "/input",
		},
		{
			name: "empty key rewrite list",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{},
				},
			},
			args: args{key: "/input"},
			want: "/input",
		},
		{
			name: "not matching regexp",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile("^/fake$"),
						Target:      "/fake2",
					}},
				},
			},
			args: args{key: "/input"},
			want: "/input",
		},
		{
			name: "matching fixed regexp and fixed target",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile("^/input$"),
						Target:      "/fake2",
					}},
				},
			},
			args: args{key: "/input"},
			want: "/fake2",
		},
		{
			name: "matching regexp with catch and fixed target",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)$`),
						Target:      "/fake2",
					}},
				},
			},
			args: args{key: "/input"},
			want: "/fake2",
		},
		{
			name: "matching regexp with catch and template target",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)$`),
						Target:      "/$one/",
					}},
				},
			},
			args: args{key: "/input"},
			want: "/input/",
		},
		{
			name: "matching regexp with catch and template target (multiple values)",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)/(?P<two>\w+)/(?P<three>\w+)$`),
						Target:      "/$two/$one/$three/$one/",
					}},
				},
			},
			args: args{key: "/input1/input2/input3"},
			want: "/input2/input1/input3/input1/",
		},
		{
			name: "matching regexp with catch and template target (multiple values 2)",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)/(?P<two>\w+)/(?P<three>\w+)?$`),
						Target:      "/$two/$one/$three/$one/",
					}},
				},
			},
			args: args{key: "/input1/input2/"},
			want: "/input2/input1//input1/",
		},
		{
			name: "matching regexp with catch and template target (multiple values 3)",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)/(?P<two>\w+)/(?P<three>\w+)?$`),
						Target:      "/$two/$one/$three/$one/",
					}},
				},
			},
			args: args{key: "/input1/input2/input3"},
			want: "/input2/input1/input3/input1/",
		},
		{
			name: "matching regexp with catch and template target (multiple key rewrite items)",
			fields: fields{
				targetCfg: &config.TargetConfig{
					KeyRewriteList: []*config.TargetKeyRewriteConfig{
						{
							SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)/$`),
							Target:      "/$one",
						},
						{
							SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)/(?P<two>\w+)/(?P<three>\w+)?$`),
							Target:      "/$two/$one/$three/$one/",
						},
					},
				},
			},
			args: args{key: "/input1/input2/input3"},
			want: "/input2/input1/input3/input1/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rctx := &requestContext{
				targetCfg: tt.fields.targetCfg,
			}
			if got := rctx.manageKeyRewrite(tt.args.key); got != tt.want {
				t.Errorf("requestContext.manageKeyRewrite() = %v, want %v", got, tt.want)
			}
		})
	}
}
