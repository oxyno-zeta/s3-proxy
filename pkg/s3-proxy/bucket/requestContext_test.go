// +build unit

package bucket

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
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
		want []*Entry
	}{
		{
			name: "Empty list",
			args: args{
				s3Entries:           []*s3client.ListElementOutput{},
				rctx:                &requestContext{},
				bucketRootPrefixKey: "prefix/",
			},
			want: []*Entry{},
		},
		{
			name: "List",
			args: args{
				s3Entries: []*s3client.ListElementOutput{
					{
						Type:         "type",
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
			want: []*Entry{
				{
					Type:         "type",
					ETag:         "etag",
					Name:         "name",
					LastModified: now,
					Size:         300,
					Key:          "key",
					Path:         "mount/key",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transformS3Entries(tt.args.s3Entries, tt.args.rctx, tt.args.bucketRootPrefixKey); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("transformS3Entries() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_requestContext_Delete(t *testing.T) {
	handleNotFoundCalled := false
	handleInternalServerErrorCalled := false
	handleForbiddenCalled := false
	handleNotFoundWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleNotFoundCalled = true
	}
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorCalled = true
	}
	handleForbiddenWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleForbiddenCalled = true
	}
	type fields struct {
		s3Context     s3client.Client
		targetCfg     *config.TargetConfig
		tplConfig     *config.TemplateConfig
		mountPath     string
		httpRW        http.ResponseWriter
		errorHandlers *ErrorHandlers
	}
	type args struct {
		requestPath string
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleNotFoundCalled            bool
		expectedHandleInternalServerErrorCalled bool
		expectedHandleForbiddenCalled           bool
		expectedHTTPWriter                      *respWriterTest
		expectedS3ClientDeleteCalled            bool
		expectedS3ClientDeleteInput             string
	}{
		{
			name: "Can't delete a directory with empty request path",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args:                                    args{requestPath: ""},
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
		{
			name: "Can't delete a directory",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args:                                    args{requestPath: "/directory/"},
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
		{
			name: "Can't delete file because of error",
			fields: fields{
				s3Context: &s3clientTest{
					DeleteErr: errors.New("test"),
				},
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args:                                    args{requestPath: "/file"},
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
			expectedS3ClientDeleteCalled:            true,
			expectedS3ClientDeleteInput:             "/file",
		},
		{
			name: "Delete file succeed",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args:                         args{requestPath: "/file"},
			expectedHTTPWriter:           &respWriterTest{Status: http.StatusNoContent},
			expectedS3ClientDeleteCalled: true,
			expectedS3ClientDeleteInput:  "/file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleForbiddenCalled = false
			handleInternalServerErrorCalled = false
			handleNotFoundCalled = false
			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorHandlers,
			}
			rctx.Delete(tt.args.requestPath)
			if handleNotFoundCalled != tt.expectedHandleNotFoundCalled {
				t.Errorf("requestContext.Delete() => handleNotFoundCalled = %+v, want %+v", handleNotFoundCalled, tt.expectedHandleNotFoundCalled)
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.Delete() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleForbiddenCalled != tt.expectedHandleForbiddenCalled {
				t.Errorf("requestContext.Delete() => handleForbiddenCalled = %+v, want %+v", handleForbiddenCalled, tt.expectedHandleForbiddenCalled)
			}
			if tt.expectedS3ClientDeleteCalled != tt.fields.s3Context.(*s3clientTest).DeleteCalled {
				t.Errorf("requestContext.Delete() => s3client.DeleteCalled = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).DeleteCalled, tt.expectedS3ClientDeleteCalled)
			}
			if tt.expectedS3ClientDeleteInput != tt.fields.s3Context.(*s3clientTest).DeleteInput {
				t.Errorf("requestContext.Delete() => s3client.DeleteInput = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).DeleteInput, tt.expectedS3ClientDeleteInput)
			}
			if !reflect.DeepEqual(tt.expectedHTTPWriter, tt.fields.httpRW) {
				t.Errorf("requestContext.Delete() => httpWriter = %+v, want %+v", tt.fields.httpRW, tt.expectedHTTPWriter)
			}
		})
	}
}

func Test_requestContext_Put(t *testing.T) {
	handleNotFoundCalled := false
	handleInternalServerErrorCalled := false
	handleForbiddenCalled := false
	handleNotFoundWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleNotFoundCalled = true
	}
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorCalled = true
	}
	handleForbiddenWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleForbiddenCalled = true
	}
	type fields struct {
		s3Context     s3client.Client
		targetCfg     *config.TargetConfig
		tplConfig     *config.TemplateConfig
		mountPath     string
		httpRW        http.ResponseWriter
		errorHandlers *ErrorHandlers
	}
	type args struct {
		inp *PutInput
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleNotFoundCalled            bool
		expectedHandleInternalServerErrorCalled bool
		expectedHandleForbiddenCalled           bool
		expectedHTTPWriter                      *respWriterTest
		expectedS3ClientPutCalled               bool
		expectedS3ClientPutInput                *s3client.PutInput
		expectedS3ClientHeadCalled              bool
		expectedS3ClientHeadInput               string
	}{
		{
			name: "should fail when put object failed and no put configuration exists",
			fields: fields{
				s3Context: &s3clientTest{
					PutErr: errors.New("test"),
				},
				targetCfg: &config.TargetConfig{
					Bucket:  &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			expectedS3ClientPutCalled:               true,
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
			expectedS3ClientPutInput: &s3client.PutInput{
				Key:         "/test/file",
				ContentType: "content-type",
			},
		},
		{
			name: "should fail when put object failed and put configuration exists with allow override",
			fields: fields{
				s3Context: &s3clientTest{
					PutErr: errors.New("test"),
				},
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
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			expectedS3ClientPutCalled:               true,
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
			expectedS3ClientPutInput: &s3client.PutInput{
				Key:         "/test/file",
				ContentType: "content-type",
				Metadata: map[string]string{
					"testkey": "testvalue",
				},
				StorageClass: "storage-class",
			},
		},
		{
			name: "should be ok with allow override",
			fields: fields{
				s3Context: &s3clientTest{},
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
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			expectedS3ClientPutCalled: true,
			expectedHTTPWriter: &respWriterTest{
				Status: http.StatusNoContent,
			},
			expectedS3ClientPutInput: &s3client.PutInput{
				Key:         "/test/file",
				ContentType: "content-type",
				Metadata: map[string]string{
					"testkey": "testvalue",
				},
				StorageClass: "storage-class",
			},
		},
		{
			name: "should be failed when head object failed",
			fields: fields{
				s3Context: &s3clientTest{
					HeadErr: errors.New("test"),
				},
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
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			expectedS3ClientHeadCalled:              true,
			expectedHandleInternalServerErrorCalled: true,
			expectedS3ClientHeadInput:               "/test/file",
			expectedHTTPWriter:                      &respWriterTest{},
		},
		{
			name: "should be failed when head object result that file exists",
			fields: fields{
				s3Context: &s3clientTest{
					HeadResult: &s3client.HeadOutput{Key: "/test/file"},
				},
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
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			expectedS3ClientHeadCalled:    true,
			expectedHandleForbiddenCalled: true,
			expectedS3ClientHeadInput:     "/test/file",
			expectedHTTPWriter:            &respWriterTest{},
		},
		{
			name: "should be failed when head object result that file doesn't exist",
			fields: fields{
				s3Context: &s3clientTest{
					HeadResult: nil,
				},
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
				tplConfig: &config.TemplateConfig{},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/test",
					Filename:    "file",
					Body:        nil,
					ContentType: "content-type",
				},
			},
			expectedS3ClientHeadCalled: true,
			expectedS3ClientPutCalled:  true,
			expectedS3ClientHeadInput:  "/test/file",
			expectedS3ClientPutInput: &s3client.PutInput{
				Key:         "/test/file",
				ContentType: "content-type",
			},
			expectedHTTPWriter: &respWriterTest{
				Status: http.StatusNoContent,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleForbiddenCalled = false
			handleInternalServerErrorCalled = false
			handleNotFoundCalled = false
			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorHandlers,
			}
			rctx.Put(tt.args.inp)
			if handleNotFoundCalled != tt.expectedHandleNotFoundCalled {
				t.Errorf("requestContext.Put() => handleNotFoundCalled = %+v, want %+v", handleNotFoundCalled, tt.expectedHandleNotFoundCalled)
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.Put() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleForbiddenCalled != tt.expectedHandleForbiddenCalled {
				t.Errorf("requestContext.Put() => handleForbiddenCalled = %+v, want %+v", handleForbiddenCalled, tt.expectedHandleForbiddenCalled)
			}
			if tt.expectedS3ClientPutCalled != tt.fields.s3Context.(*s3clientTest).PutCalled {
				t.Errorf("requestContext.Put() => s3client.PutCalled = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).PutCalled, tt.expectedS3ClientPutCalled)
			}
			if !reflect.DeepEqual(tt.expectedS3ClientPutInput, tt.fields.s3Context.(*s3clientTest).PutInput) {
				t.Errorf("requestContext.Put() => s3client.PutInput = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).PutInput, tt.expectedS3ClientPutInput)
			}
			if tt.expectedS3ClientHeadCalled != tt.fields.s3Context.(*s3clientTest).HeadCalled {
				t.Errorf("requestContext.Put() => s3client.HeadCalled = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).HeadCalled, tt.expectedS3ClientHeadCalled)
			}
			if !reflect.DeepEqual(tt.expectedS3ClientHeadInput, tt.fields.s3Context.(*s3clientTest).HeadInput) {
				t.Errorf("requestContext.Put() => s3client.HeadInput = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).HeadInput, tt.expectedS3ClientHeadInput)
			}
			if !reflect.DeepEqual(tt.expectedHTTPWriter, tt.fields.httpRW) {
				t.Errorf("requestContext.Put() => httpWriter = %+v, want %+v", tt.fields.httpRW, tt.expectedHTTPWriter)
			}
		})
	}
}

func Test_requestContext_Get(t *testing.T) {
	fakeDate := time.Date(1990, time.December, 25, 1, 1, 1, 1, time.UTC)
	handleNotFoundCalled := false
	handleInternalServerErrorCalled := false
	handleForbiddenCalled := false
	handleNotFoundWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleNotFoundCalled = true
	}
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorCalled = true
	}
	handleForbiddenWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleForbiddenCalled = true
	}
	h := http.Header{}
	h.Set("Content-Type", "text/html; charset=utf-8")
	fakeIndexIoReadCloser := ioutil.NopCloser(strings.NewReader("fake-index.html-content"))
	fakeIndexIoReadCloser2 := ioutil.NopCloser(strings.NewReader("fake-index.html-content"))
	type fields struct {
		s3Context     s3client.Client
		targetCfg     *config.TargetConfig
		tplConfig     *config.TemplateConfig
		mountPath     string
		httpRW        http.ResponseWriter
		errorHandlers *ErrorHandlers
	}
	type args struct {
		requestPath string
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleNotFoundCalled            bool
		expectedHandleInternalServerErrorCalled bool
		expectedHandleForbiddenCalled           bool
		expectedHTTPWriter                      *respWriterTest
		expectedS3ClientListCalled              bool
		expectedS3ClientListInput               string
		expectedS3ClientGetCalled               bool
		expectedS3ClientGetInput                string
	}{
		{
			name: "should fail if list files and directories failed",
			fields: fields{
				s3Context: &s3clientTest{
					ListErr: errors.New("test"),
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedS3ClientListCalled:              true,
			expectedS3ClientListInput:               "/folder/",
			expectedHTTPWriter:                      &respWriterTest{},
		},
		{
			name: "should fail if list files and directories template failed because template not found",
			fields: fields{
				s3Context: &s3clientTest{
					ListResult: []*s3client.ListElementOutput{
						{
							Name:         "file1",
							Type:         "FILE",
							ETag:         "etag",
							LastModified: fakeDate,
							Size:         300,
							Key:          "/folder/file1",
						},
					},
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "fake/path",
				},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedS3ClientListCalled:              true,
			expectedS3ClientListInput:               "/folder/",
			expectedHTTPWriter:                      &respWriterTest{},
		},
		{
			name: "should be ok to list files and directories",
			fields: fields{
				s3Context: &s3clientTest{
					ListResult: []*s3client.ListElementOutput{
						{
							Name:         "file1",
							Type:         "FILE",
							ETag:         "etag",
							LastModified: fakeDate,
							Size:         300,
							Key:          "/folder/file1",
						},
					},
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW: &respWriterTest{
					Headers: http.Header{},
				},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/",
			},
			expectedS3ClientListCalled: true,
			expectedS3ClientListInput:  "/folder/",
			expectedHTTPWriter: &respWriterTest{
				Headers: h,
				Status:  http.StatusOK,
				Resp: []byte(`<!DOCTYPE html>
<html>
  <body>
    <h1>Index of /mount/folder/</h1>
    <table style="width:100%">
        <thead>
            <tr>
                <th style="border-right:1px solid black;text-align:start">Entry</th>
                <th style="border-right:1px solid black;text-align:start">Size</th>
                <th style="border-right:1px solid black;text-align:start">Last modified</th>
            </tr>
        </thead>
        <tbody style="border-top:1px solid black">
          <tr>
            <td style="border-right:1px solid black;padding: 0 5px"><a href="..">..</a></td>
            <td style="border-right:1px solid black;padding: 0 5px"> - </td>
            <td style="padding: 0 5px"> - </td>
          </tr>
          <tr>
              <td style="border-right:1px solid black;padding: 0 5px"><a href="/mountfolder/file1">file1</a></td>
              <td style="border-right:1px solid black;padding: 0 5px">300 B</td>
              <td style="padding: 0 5px">1990-12-25 01:01:01.000000001 &#43;0000 UTC</td>
          </tr>
        </tbody>
    </table>
  </body>
</html>
`),
			},
		},
		{
			name: "should be ok to find and load index document",
			fields: fields{
				s3Context: &s3clientTest{
					ListResult: []*s3client.ListElementOutput{
						{
							Name:         "index.html",
							Type:         "FILE",
							ETag:         "etag",
							LastModified: fakeDate,
							Size:         300,
							Key:          "/folder/index.html",
						},
					},
					GetResult: &s3client.GetOutput{
						Body:        &fakeIndexIoReadCloser,
						ContentType: "text/html; charset=utf-8",
					},
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					IndexDocument: "index.html",
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW: &respWriterTest{
					Headers: http.Header{},
				},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/",
			},
			expectedS3ClientListCalled: true,
			expectedS3ClientListInput:  "/folder/",
			expectedS3ClientGetCalled:  true,
			expectedS3ClientGetInput:   "/folder/index.html",
			expectedHTTPWriter: &respWriterTest{
				Headers: h,
				Status:  http.StatusOK,
				Resp:    []byte("fake-index.html-content"),
			},
		},
		{
			name: "should fail to find and load index document with not found error",
			fields: fields{
				s3Context: &s3clientTest{
					ListResult: []*s3client.ListElementOutput{
						{
							Name:         "index.html",
							Type:         "FILE",
							ETag:         "etag",
							LastModified: fakeDate,
							Size:         300,
							Key:          "/folder/index.html",
						},
					},
					GetErr: s3client.ErrNotFound,
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					IndexDocument: "index.html",
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/",
			},
			expectedS3ClientListCalled:   true,
			expectedS3ClientListInput:    "/folder/",
			expectedS3ClientGetCalled:    true,
			expectedS3ClientGetInput:     "/folder/index.html",
			expectedHTTPWriter:           &respWriterTest{},
			expectedHandleNotFoundCalled: true,
		},
		{
			name: "should fail to find and load index document with error",
			fields: fields{
				s3Context: &s3clientTest{
					ListResult: []*s3client.ListElementOutput{
						{
							Name:         "index.html",
							Type:         "FILE",
							ETag:         "etag",
							LastModified: fakeDate,
							Size:         300,
							Key:          "/folder/index.html",
						},
					},
					GetErr: errors.New("test-error"),
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					IndexDocument: "index.html",
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/",
			},
			expectedS3ClientListCalled:              true,
			expectedS3ClientListInput:               "/folder/",
			expectedS3ClientGetCalled:               true,
			expectedS3ClientGetInput:                "/folder/index.html",
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
		{
			name: "should be ok to get file",
			fields: fields{
				s3Context: &s3clientTest{
					GetResult: &s3client.GetOutput{
						Body:        &fakeIndexIoReadCloser2,
						ContentType: "text/html; charset=utf-8",
					},
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW: &respWriterTest{
					Headers: http.Header{},
				},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/index.html",
			},
			expectedS3ClientGetCalled: true,
			expectedS3ClientGetInput:  "/folder/index.html",
			expectedHTTPWriter: &respWriterTest{
				Headers: h,
				Status:  http.StatusOK,
				Resp:    []byte("fake-index.html-content"),
			},
		},
		{
			name: "should fail to get file when not found error is raised",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: s3client.ErrNotFound,
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/index.html",
			},
			expectedS3ClientGetCalled:    true,
			expectedS3ClientGetInput:     "/folder/index.html",
			expectedHTTPWriter:           &respWriterTest{},
			expectedHandleNotFoundCalled: true,
		},
		{
			name: "should fail to get file when error is raised",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: errors.New("test-error"),
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
				},
				tplConfig: &config.TemplateConfig{
					FolderList: "../../../templates/folder-list.tpl",
				},
				mountPath: "/mount",
				httpRW:    &respWriterTest{},
				errorHandlers: &ErrorHandlers{
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				requestPath: "/folder/index.html",
			},
			expectedS3ClientGetCalled:               true,
			expectedS3ClientGetInput:                "/folder/index.html",
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleNotFoundCalled = false
			handleInternalServerErrorCalled = false
			handleForbiddenCalled = false
			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorHandlers,
			}
			rctx.Get(tt.args.requestPath)
			if handleNotFoundCalled != tt.expectedHandleNotFoundCalled {
				t.Errorf("requestContext.Get() => handleNotFoundCalled = %+v, want %+v", handleNotFoundCalled, tt.expectedHandleNotFoundCalled)
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.Get() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleForbiddenCalled != tt.expectedHandleForbiddenCalled {
				t.Errorf("requestContext.Get() => handleForbiddenCalled = %+v, want %+v", handleForbiddenCalled, tt.expectedHandleForbiddenCalled)
			}
			if tt.expectedS3ClientListCalled != tt.fields.s3Context.(*s3clientTest).ListCalled {
				t.Errorf("requestContext.Get() => s3client.ListCalled = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).ListCalled, tt.expectedS3ClientListCalled)
			}
			if !reflect.DeepEqual(tt.expectedS3ClientListInput, tt.fields.s3Context.(*s3clientTest).ListInput) {
				t.Errorf("requestContext.Get() => s3client.ListInput = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).ListInput, tt.expectedS3ClientListInput)
			}
			if tt.expectedS3ClientGetCalled != tt.fields.s3Context.(*s3clientTest).GetCalled {
				t.Errorf("requestContext.Get() => s3client.GetCalled = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).GetCalled, tt.expectedS3ClientGetCalled)
			}
			if !reflect.DeepEqual(tt.expectedS3ClientGetInput, tt.fields.s3Context.(*s3clientTest).GetInput) {
				t.Errorf("requestContext.Get() => s3client.GetInput = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).GetInput, tt.expectedS3ClientGetInput)
			}
			if !reflect.DeepEqual(tt.expectedHTTPWriter, tt.fields.httpRW) {
				t.Errorf("requestContext.Get() => httpWriter = %+v, want %+v", tt.fields.httpRW, tt.expectedHTTPWriter)
			}
		})
	}
}

func Test_requestContext_HandleInternalServerError(t *testing.T) {
	err := errors.New("fake")
	thrownErr := errors.New("fake err")
	handleInternalServerErrorCalled := false
	handleInternalServerErrorTmpl := ""
	var handleInternalServerErrorErr error
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}
	tplFileContent := "Fake template"
	bodyReadCloser := ioutil.NopCloser(strings.NewReader(tplFileContent))

	type fields struct {
		s3Context      s3client.Client
		targetCfg      *config.TargetConfig
		tplConfig      *config.TemplateConfig
		mountPath      string
		httpRW         http.ResponseWriter
		errorsHandlers *ErrorHandlers
	}
	type args struct {
		err         error
		requestPath string
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleInternalServerErrorCalled bool
		expectedHandleInternalServerErrorTmpl   string
		expectedhandleInternalServerErrorErr    error
		shouldCreateFile                        bool
	}{
		{
			name: "should work without templates in target configuration",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    err,
		},
		{
			name: "should handle error from S3 client",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: thrownErr,
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						InternalServerError: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    thrownErr,
		},
		{
			name: "should work with templates in bucket",
			fields: fields{
				s3Context: &s3clientTest{
					GetResult: &s3client.GetOutput{
						Body: &bodyReadCloser,
					},
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						InternalServerError: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   tplFileContent,
			expectedhandleInternalServerErrorErr:    err,
		},
		{
			name: "should handle error from FS read",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						InternalServerError: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    errors.New("open /fake/path/file: no such file or directory"),
		},
		{
			name: "should read FS for file template",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						InternalServerError: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    err,
			shouldCreateFile:                        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleInternalServerErrorCalled = false
			handleInternalServerErrorErr = nil

			if tt.shouldCreateFile {
				dir, err := ioutil.TempDir("", "s3-proxy")
				if err != nil {
					t.Error(err)
					return
				}

				defer os.RemoveAll(dir) // clean up
				tmpfn := filepath.Join(dir, tt.fields.targetCfg.Templates.InternalServerError.Path)
				// Get base directory
				fulldir := filepath.Dir(tmpfn)
				// Create all directories
				err = os.MkdirAll(fulldir, os.ModePerm)
				if err != nil {
					t.Error(err)
					return
				}
				// Write file
				err = ioutil.WriteFile(tmpfn, []byte(tt.expectedHandleInternalServerErrorTmpl), 0666)
				if err != nil {
					t.Error(err)
					return
				}

				// Edit file path in config
				tt.fields.targetCfg.Templates.InternalServerError.Path = tmpfn
			}

			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorsHandlers,
			}

			rctx.HandleInternalServerError(tt.args.err, tt.args.requestPath)

			// Tests
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleInternalServerErrorTmpl != tt.expectedHandleInternalServerErrorTmpl {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorTmpl = %+v, want %+v", handleInternalServerErrorTmpl, tt.expectedHandleInternalServerErrorTmpl)
			}
			if handleInternalServerErrorErr.Error() != tt.expectedhandleInternalServerErrorErr.Error() {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
			}
		})
	}
}

func Test_requestContext_HandleNotFound(t *testing.T) {
	thrownErr := errors.New("fake err")
	handleInternalServerErrorCalled := false
	handleInternalServerErrorTmpl := ""
	var handleInternalServerErrorErr error
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleNotFoundCalled := false
	handleNotFoundTmpl := ""
	handleNotFoundWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleNotFoundCalled = true
		handleNotFoundTmpl = tplString
	}
	tplFileContent := "Fake template"
	bodyReadCloser := ioutil.NopCloser(strings.NewReader(tplFileContent))

	type fields struct {
		s3Context      s3client.Client
		targetCfg      *config.TargetConfig
		tplConfig      *config.TemplateConfig
		mountPath      string
		httpRW         http.ResponseWriter
		errorsHandlers *ErrorHandlers
	}
	type args struct {
		requestPath string
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleInternalServerErrorCalled bool
		expectedHandleInternalServerErrorTmpl   string
		expectedhandleInternalServerErrorErr    error
		expectedHandleNotFoundCalled            bool
		expectedHandleNotFoundTmpl              string
		shouldCreateFile                        bool
	}{
		{
			name: "should work without templates in target configuration",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleNotFoundCalled: true,
			expectedHandleNotFoundTmpl:   "",
		},
		{
			name: "should handle error from S3 client",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: thrownErr,
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						NotFound: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    thrownErr,
		},
		{
			name: "should work with templates in bucket",
			fields: fields{
				s3Context: &s3clientTest{
					GetResult: &s3client.GetOutput{
						Body: &bodyReadCloser,
					},
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						NotFound: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleNotFoundCalled: true,
			expectedHandleNotFoundTmpl:   tplFileContent,
		},
		{
			name: "should handle error from FS read",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						NotFound: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file-not-found",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    errors.New("open /fake/path/file-not-found: no such file or directory"),
		},
		{
			name: "should read FS for file template",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						NotFound: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleNotFoundWithTemplate:            handleNotFoundWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleNotFoundCalled: true,
			expectedHandleNotFoundTmpl:   tplFileContent,
			shouldCreateFile:             true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleInternalServerErrorCalled = false
			handleInternalServerErrorErr = nil
			handleInternalServerErrorTmpl = ""
			handleNotFoundCalled = false
			handleNotFoundTmpl = ""

			if tt.shouldCreateFile {
				dir, err := ioutil.TempDir("", "s3-proxy")
				if err != nil {
					t.Error(err)
					return
				}

				defer os.RemoveAll(dir) // clean up
				tmpfn := filepath.Join(dir, tt.fields.targetCfg.Templates.NotFound.Path)
				// Get base directory
				fulldir := filepath.Dir(tmpfn)
				// Create all directories
				err = os.MkdirAll(fulldir, os.ModePerm)
				if err != nil {
					t.Error(err)
					return
				}
				// Write file
				err = ioutil.WriteFile(tmpfn, []byte(tt.expectedHandleNotFoundTmpl), 0666)
				if err != nil {
					t.Error(err)
					return
				}

				// Edit file path in config
				tt.fields.targetCfg.Templates.NotFound.Path = tmpfn
			}

			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorsHandlers,
			}
			rctx.HandleNotFound(tt.args.requestPath)

			// Tests
			if handleNotFoundCalled != tt.expectedHandleNotFoundCalled {
				t.Errorf("requestContext.HandleNotFound() => handleNotFoundCalled = %+v, want %+v", handleNotFoundCalled, tt.expectedHandleNotFoundCalled)
			}
			if handleNotFoundTmpl != tt.expectedHandleNotFoundTmpl {
				t.Errorf("requestContext.HandleNotFound() => handleNotFoundTmpl = %+v, want %+v", handleNotFoundTmpl, tt.expectedHandleNotFoundTmpl)
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleInternalServerErrorTmpl != tt.expectedHandleInternalServerErrorTmpl {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorTmpl = %+v, want %+v", handleInternalServerErrorTmpl, tt.expectedHandleInternalServerErrorTmpl)
			}
			if tt.expectedhandleInternalServerErrorErr != nil || handleInternalServerErrorErr != nil {
				if handleInternalServerErrorErr != nil && tt.expectedhandleInternalServerErrorErr != nil &&
					handleInternalServerErrorErr.Error() != tt.expectedhandleInternalServerErrorErr.Error() {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				} else if handleInternalServerErrorErr == nil || tt.expectedhandleInternalServerErrorErr == nil {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				}
			}
		})
	}
}

func Test_requestContext_HandleForbidden(t *testing.T) {
	thrownErr := errors.New("fake err")
	handleInternalServerErrorCalled := false
	handleInternalServerErrorTmpl := ""
	var handleInternalServerErrorErr error
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleForbiddenCalled := false
	handleForbiddenTmpl := ""
	handleForbiddenWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleForbiddenCalled = true
		handleForbiddenTmpl = tplString
	}
	tplFileContent := "Fake template"
	bodyReadCloser := ioutil.NopCloser(strings.NewReader(tplFileContent))

	type fields struct {
		s3Context      s3client.Client
		targetCfg      *config.TargetConfig
		tplConfig      *config.TemplateConfig
		mountPath      string
		httpRW         http.ResponseWriter
		errorsHandlers *ErrorHandlers
	}
	type args struct {
		requestPath string
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleInternalServerErrorCalled bool
		expectedHandleInternalServerErrorTmpl   string
		expectedhandleInternalServerErrorErr    error
		expectedHandleForbiddenCalled           bool
		expectedHandleForbiddenTmpl             string
		shouldCreateFile                        bool
	}{
		{
			name: "should work without templates in target configuration",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleForbiddenCalled: true,
			expectedHandleForbiddenTmpl:   "",
		},
		{
			name: "should handle error from S3 client",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: thrownErr,
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Forbidden: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    thrownErr,
		},
		{
			name: "should work with templates in bucket",
			fields: fields{
				s3Context: &s3clientTest{
					GetResult: &s3client.GetOutput{
						Body: &bodyReadCloser,
					},
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Forbidden: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleForbiddenCalled: true,
			expectedHandleForbiddenTmpl:   tplFileContent,
		},
		{
			name: "should handle error from FS read",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Forbidden: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file-not-found",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    errors.New("open /fake/path/file-not-found: no such file or directory"),
		},
		{
			name: "should read FS for file template",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Forbidden: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleForbiddenWithTemplate:           handleForbiddenWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleForbiddenCalled: true,
			expectedHandleForbiddenTmpl:   tplFileContent,
			shouldCreateFile:              true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleInternalServerErrorCalled = false
			handleInternalServerErrorErr = nil
			handleInternalServerErrorTmpl = ""
			handleForbiddenCalled = false
			handleForbiddenTmpl = ""

			if tt.shouldCreateFile {
				dir, err := ioutil.TempDir("", "s3-proxy")
				if err != nil {
					t.Error(err)
					return
				}

				defer os.RemoveAll(dir) // clean up
				tmpfn := filepath.Join(dir, tt.fields.targetCfg.Templates.Forbidden.Path)
				// Get base directory
				fulldir := filepath.Dir(tmpfn)
				// Create all directories
				err = os.MkdirAll(fulldir, os.ModePerm)
				if err != nil {
					t.Error(err)
					return
				}
				// Write file
				err = ioutil.WriteFile(tmpfn, []byte(tt.expectedHandleForbiddenTmpl), 0666)
				if err != nil {
					t.Error(err)
					return
				}

				// Edit file path in config
				tt.fields.targetCfg.Templates.Forbidden.Path = tmpfn
			}

			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorsHandlers,
			}
			rctx.HandleForbidden(tt.args.requestPath)

			// Tests
			if handleForbiddenCalled != tt.expectedHandleForbiddenCalled {
				t.Errorf("requestContext.HandleForbidden() => handleForbiddenCalled = %+v, want %+v", handleForbiddenCalled, tt.expectedHandleForbiddenCalled)
			}
			if handleForbiddenTmpl != tt.expectedHandleForbiddenTmpl {
				t.Errorf("requestContext.HandleForbidden() => handleForbiddenTmpl = %+v, want %+v", handleForbiddenTmpl, tt.expectedHandleForbiddenTmpl)
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleInternalServerErrorTmpl != tt.expectedHandleInternalServerErrorTmpl {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorTmpl = %+v, want %+v", handleInternalServerErrorTmpl, tt.expectedHandleInternalServerErrorTmpl)
			}
			if tt.expectedhandleInternalServerErrorErr != nil || handleInternalServerErrorErr != nil {
				if handleInternalServerErrorErr != nil && tt.expectedhandleInternalServerErrorErr != nil &&
					handleInternalServerErrorErr.Error() != tt.expectedhandleInternalServerErrorErr.Error() {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				} else if handleInternalServerErrorErr == nil || tt.expectedhandleInternalServerErrorErr == nil {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				}
			}
		})
	}
}

func Test_requestContext_HandleBadRequest(t *testing.T) {
	err := errors.New("bad request error")
	thrownErr := errors.New("fake err")
	handleInternalServerErrorCalled := false
	handleInternalServerErrorTmpl := ""
	var handleInternalServerErrorErr error
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleBadRequestCalled := false
	handleBadRequestTmpl := ""
	var handleBadRequestErr error
	handleBadRequestWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, err error, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleBadRequestCalled = true
		handleBadRequestTmpl = tplString
		handleBadRequestErr = err
	}
	tplFileContent := "Fake template"
	bodyReadCloser := ioutil.NopCloser(strings.NewReader(tplFileContent))

	type fields struct {
		s3Context      s3client.Client
		targetCfg      *config.TargetConfig
		tplConfig      *config.TemplateConfig
		mountPath      string
		httpRW         http.ResponseWriter
		errorsHandlers *ErrorHandlers
	}
	type args struct {
		err         error
		requestPath string
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleInternalServerErrorCalled bool
		expectedHandleInternalServerErrorTmpl   string
		expectedhandleInternalServerErrorErr    error
		expectedHandleBadRequestCalled          bool
		expectedHandleBadRequestTmpl            string
		expectedhandleBadRequestErr             error
		shouldCreateFile                        bool
	}{
		{
			name: "should work without templates in target configuration",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleBadRequestWithTemplate:          handleBadRequestWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleBadRequestCalled: true,
			expectedHandleBadRequestTmpl:   "",
			expectedhandleBadRequestErr:    err,
		},
		{
			name: "should handle error from S3 client",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: thrownErr,
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						BadRequest: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleBadRequestWithTemplate:          handleBadRequestWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    thrownErr,
		},
		{
			name: "should work with templates in bucket",
			fields: fields{
				s3Context: &s3clientTest{
					GetResult: &s3client.GetOutput{
						Body: &bodyReadCloser,
					},
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						BadRequest: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleBadRequestWithTemplate:          handleBadRequestWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleBadRequestCalled: true,
			expectedHandleBadRequestTmpl:   tplFileContent,
			expectedhandleBadRequestErr:    err,
		},
		{
			name: "should handle error from FS read",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						BadRequest: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file-not-found",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleBadRequestWithTemplate:          handleBadRequestWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    errors.New("open /fake/path/file-not-found: no such file or directory"),
		},
		{
			name: "should read FS for file template",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						BadRequest: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleBadRequestWithTemplate:          handleBadRequestWithTemplate,
				},
			},
			args: args{
				err:         err,
				requestPath: "/fake",
			},
			expectedHandleBadRequestCalled: true,
			expectedHandleBadRequestTmpl:   tplFileContent,
			expectedhandleBadRequestErr:    err,
			shouldCreateFile:               true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleInternalServerErrorCalled = false
			handleInternalServerErrorErr = nil
			handleInternalServerErrorTmpl = ""
			handleBadRequestCalled = false
			handleBadRequestTmpl = ""
			handleBadRequestErr = nil

			if tt.shouldCreateFile {
				dir, err := ioutil.TempDir("", "s3-proxy")
				if err != nil {
					t.Error(err)
					return
				}

				defer os.RemoveAll(dir) // clean up
				tmpfn := filepath.Join(dir, tt.fields.targetCfg.Templates.BadRequest.Path)
				// Get base directory
				fulldir := filepath.Dir(tmpfn)
				// Create all directories
				err = os.MkdirAll(fulldir, os.ModePerm)
				if err != nil {
					t.Error(err)
					return
				}
				// Write file
				err = ioutil.WriteFile(tmpfn, []byte(tt.expectedHandleBadRequestTmpl), 0666)
				if err != nil {
					t.Error(err)
					return
				}

				// Edit file path in config
				tt.fields.targetCfg.Templates.BadRequest.Path = tmpfn
			}

			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorsHandlers,
			}
			rctx.HandleBadRequest(tt.args.err, tt.args.requestPath)

			// Tests
			if handleBadRequestCalled != tt.expectedHandleBadRequestCalled {
				t.Errorf("requestContext.HandleBadRequest() => handleBadRequestCalled = %+v, want %+v", handleBadRequestCalled, tt.expectedHandleBadRequestCalled)
			}
			if handleBadRequestTmpl != tt.expectedHandleBadRequestTmpl {
				t.Errorf("requestContext.HandleBadRequest() => handleBadRequestTmpl = %+v, want %+v", handleBadRequestTmpl, tt.expectedHandleBadRequestTmpl)
			}
			if tt.expectedhandleBadRequestErr != nil || handleBadRequestErr != nil {
				if handleBadRequestErr != nil && tt.expectedhandleBadRequestErr != nil &&
					handleBadRequestErr.Error() != tt.expectedhandleBadRequestErr.Error() {
					t.Errorf("requestContext.HandleBadRequest() => handleBadRequestErr = %+v, want %+v", handleBadRequestErr, tt.expectedhandleBadRequestErr)
				} else if handleBadRequestErr == nil || tt.expectedhandleBadRequestErr == nil {
					t.Errorf("requestContext.HandleBadRequest() => handleBadRequestErr = %+v, want %+v", handleBadRequestErr, tt.expectedhandleBadRequestErr)
				}
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleInternalServerErrorTmpl != tt.expectedHandleInternalServerErrorTmpl {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorTmpl = %+v, want %+v", handleInternalServerErrorTmpl, tt.expectedHandleInternalServerErrorTmpl)
			}
			if tt.expectedhandleInternalServerErrorErr != nil || handleInternalServerErrorErr != nil {
				if handleInternalServerErrorErr != nil && tt.expectedhandleInternalServerErrorErr != nil &&
					handleInternalServerErrorErr.Error() != tt.expectedhandleInternalServerErrorErr.Error() {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				} else if handleInternalServerErrorErr == nil || tt.expectedhandleInternalServerErrorErr == nil {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				}
			}
		})
	}
}

func Test_requestContext_HandleUnauthorized(t *testing.T) {
	thrownErr := errors.New("fake err")
	handleInternalServerErrorCalled := false
	handleInternalServerErrorTmpl := ""
	var handleInternalServerErrorErr error
	handleInternalServerErrorWithTemplate := func(tplString string, rw http.ResponseWriter, err error, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleUnauthorizedCalled := false
	handleUnauthorizedTmpl := ""
	handleUnauthorizedWithTemplate := func(tplString string, rw http.ResponseWriter, requestPath string, logger log.Logger, tplCfg *config.TemplateConfig) {
		handleUnauthorizedCalled = true
		handleUnauthorizedTmpl = tplString
	}
	tplFileContent := "Fake template"
	bodyReadCloser := ioutil.NopCloser(strings.NewReader(tplFileContent))

	type fields struct {
		s3Context      s3client.Client
		targetCfg      *config.TargetConfig
		tplConfig      *config.TemplateConfig
		mountPath      string
		httpRW         http.ResponseWriter
		errorsHandlers *ErrorHandlers
	}
	type args struct {
		requestPath string
	}
	tests := []struct {
		name                                    string
		fields                                  fields
		args                                    args
		expectedHandleInternalServerErrorCalled bool
		expectedHandleInternalServerErrorTmpl   string
		expectedhandleInternalServerErrorErr    error
		expectedHandleUnauthorizedCalled        bool
		expectedHandleUnauthorizedTmpl          string
		shouldCreateFile                        bool
	}{
		{
			name: "should work without templates in target configuration",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleUnauthorizedWithTemplate:        handleUnauthorizedWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleUnauthorizedCalled: true,
			expectedHandleUnauthorizedTmpl:   "",
		},
		{
			name: "should handle error from S3 client",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: thrownErr,
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Unauthorized: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleUnauthorizedWithTemplate:        handleUnauthorizedWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    thrownErr,
		},
		{
			name: "should work with templates in bucket",
			fields: fields{
				s3Context: &s3clientTest{
					GetResult: &s3client.GetOutput{
						Body: &bodyReadCloser,
					},
				},
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Unauthorized: &config.TargetTemplateConfigItem{
							InBucket: true,
							Path:     "/fake/path",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleUnauthorizedWithTemplate:        handleUnauthorizedWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleUnauthorizedCalled: true,
			expectedHandleUnauthorizedTmpl:   tplFileContent,
		},
		{
			name: "should handle error from FS read",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Unauthorized: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file-not-found",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleUnauthorizedWithTemplate:        handleUnauthorizedWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleInternalServerErrorCalled: true,
			expectedHandleInternalServerErrorTmpl:   "",
			expectedhandleInternalServerErrorErr:    errors.New("open /fake/path/file-not-found: no such file or directory"),
		},
		{
			name: "should read FS for file template",
			fields: fields{
				s3Context: nil,
				targetCfg: &config.TargetConfig{
					Templates: &config.TargetTemplateConfig{
						Unauthorized: &config.TargetTemplateConfigItem{
							InBucket: false,
							Path:     "/fake/path/file",
						},
					},
				},
				tplConfig: &config.TemplateConfig{},
				mountPath: "/test",
				httpRW:    &respWriterTest{},
				errorsHandlers: &ErrorHandlers{
					HandleInternalServerErrorWithTemplate: handleInternalServerErrorWithTemplate,
					HandleUnauthorizedWithTemplate:        handleUnauthorizedWithTemplate,
				},
			},
			args: args{
				requestPath: "/fake",
			},
			expectedHandleUnauthorizedCalled: true,
			expectedHandleUnauthorizedTmpl:   tplFileContent,
			shouldCreateFile:                 true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleInternalServerErrorCalled = false
			handleInternalServerErrorErr = nil
			handleInternalServerErrorTmpl = ""
			handleUnauthorizedCalled = false
			handleUnauthorizedTmpl = ""

			if tt.shouldCreateFile {
				dir, err := ioutil.TempDir("", "s3-proxy")
				if err != nil {
					t.Error(err)
					return
				}

				defer os.RemoveAll(dir) // clean up
				tmpfn := filepath.Join(dir, tt.fields.targetCfg.Templates.Unauthorized.Path)
				// Get base directory
				fulldir := filepath.Dir(tmpfn)
				// Create all directories
				err = os.MkdirAll(fulldir, os.ModePerm)
				if err != nil {
					t.Error(err)
					return
				}
				// Write file
				err = ioutil.WriteFile(tmpfn, []byte(tt.expectedHandleUnauthorizedTmpl), 0666)
				if err != nil {
					t.Error(err)
					return
				}

				// Edit file path in config
				tt.fields.targetCfg.Templates.Unauthorized.Path = tmpfn
			}

			rctx := &requestContext{
				s3Context:      tt.fields.s3Context,
				logger:         log.NewLogger(),
				targetCfg:      tt.fields.targetCfg,
				tplConfig:      tt.fields.tplConfig,
				mountPath:      tt.fields.mountPath,
				httpRW:         tt.fields.httpRW,
				errorsHandlers: tt.fields.errorsHandlers,
			}
			rctx.HandleUnauthorized(tt.args.requestPath)

			// Tests
			if handleUnauthorizedCalled != tt.expectedHandleUnauthorizedCalled {
				t.Errorf("requestContext.HandleUnauthorized() => handleUnauthorizedCalled = %+v, want %+v", handleUnauthorizedCalled, tt.expectedHandleUnauthorizedCalled)
			}
			if handleUnauthorizedTmpl != tt.expectedHandleUnauthorizedTmpl {
				t.Errorf("requestContext.HandleUnauthorized() => handleUnauthorizedTmpl = %+v, want %+v", handleUnauthorizedTmpl, tt.expectedHandleUnauthorizedTmpl)
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleInternalServerErrorTmpl != tt.expectedHandleInternalServerErrorTmpl {
				t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorTmpl = %+v, want %+v", handleInternalServerErrorTmpl, tt.expectedHandleInternalServerErrorTmpl)
			}
			if tt.expectedhandleInternalServerErrorErr != nil || handleInternalServerErrorErr != nil {
				if handleInternalServerErrorErr != nil && tt.expectedhandleInternalServerErrorErr != nil &&
					handleInternalServerErrorErr.Error() != tt.expectedhandleInternalServerErrorErr.Error() {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				} else if handleInternalServerErrorErr == nil || tt.expectedhandleInternalServerErrorErr == nil {
					t.Errorf("requestContext.HandleInternalServerError() => handleInternalServerErrorErr = %+v, want %+v", handleInternalServerErrorErr, tt.expectedhandleInternalServerErrorErr)
				}
			}
		})
	}
}
