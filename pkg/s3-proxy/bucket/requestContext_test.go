// +build unit

package bucket

import (
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
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
	handleNotFoundWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
		handleNotFoundCalled = true
	}
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
		handleInternalServerErrorCalled = true
	}
	handleForbiddenWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
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
		{
			name: "Delete succeed with rewrite key",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile("^/file$"),
						Target:      "/fake/file2",
					}},
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
			expectedS3ClientDeleteInput:  "/fake/file2",
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
	handleNotFoundWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
		handleNotFoundCalled = true
	}
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
		handleInternalServerErrorCalled = true
	}
	handleForbiddenWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
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
		{
			name: "should be ok with allow override and key rewrite",
			fields: fields{
				s3Context: &s3clientTest{},
				targetCfg: &config.TargetConfig{
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
				Key:         "/test1/test2/file",
				ContentType: "content-type",
				Metadata: map[string]string{
					"testkey": "testvalue",
				},
				StorageClass: "storage-class",
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
	handleNotFoundWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
		handleNotFoundCalled = true
	}
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
		handleInternalServerErrorCalled = true
	}
	handleForbiddenWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
		handleForbiddenCalled = true
	}
	emptyHeader := http.Header{}
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
		input *GetInput
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
		expectedS3ClientHeadCalled              bool
		expectedS3ClientHeadInput               string
		expectedS3ClientGetCalled               bool
		expectedS3ClientGetInput                *s3client.GetInput
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
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/"},
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
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/"},
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
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/"},
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
					HeadResult: &s3client.HeadOutput{
						Type: "FILE",
						Key:  "/folder/index.html",
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
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						IndexDocument: "index.html",
					}},
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
				input: &GetInput{RequestPath: "/folder/"},
			},
			expectedS3ClientGetCalled:  true,
			expectedS3ClientGetInput:   &s3client.GetInput{Key: "/folder/index.html"},
			expectedS3ClientHeadCalled: true,
			expectedS3ClientHeadInput:  "/folder/index.html",
			expectedHTTPWriter: &respWriterTest{
				Headers: h,
				Status:  http.StatusOK,
				Resp:    []byte("fake-index.html-content"),
			},
		},
		{
			name: "should return a 304 when S3 client return a 304 error",
			fields: fields{
				s3Context: &s3clientTest{
					HeadResult: &s3client.HeadOutput{
						Type: "FILE",
						Key:  "/folder/index.html",
					},
					GetErr: s3client.ErrNotModified,
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						IndexDocument: "index.html",
					}},
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
				input: &GetInput{RequestPath: "/folder/"},
			},
			expectedS3ClientGetCalled:  true,
			expectedS3ClientGetInput:   &s3client.GetInput{Key: "/folder/index.html"},
			expectedS3ClientHeadCalled: true,
			expectedS3ClientHeadInput:  "/folder/index.html",
			expectedHTTPWriter: &respWriterTest{
				Headers: emptyHeader,
				Status:  http.StatusNotModified,
				Resp:    nil,
			},
		},
		{
			name: "should return a 412 when S3 client return a 412 error",
			fields: fields{
				s3Context: &s3clientTest{
					HeadResult: &s3client.HeadOutput{
						Type: "FILE",
						Key:  "/folder/index.html",
					},
					GetErr: s3client.ErrPreconditionFailed,
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						IndexDocument: "index.html",
					}},
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
				input: &GetInput{RequestPath: "/folder/"},
			},
			expectedS3ClientGetCalled:  true,
			expectedS3ClientGetInput:   &s3client.GetInput{Key: "/folder/index.html"},
			expectedS3ClientHeadCalled: true,
			expectedS3ClientHeadInput:  "/folder/index.html",
			expectedHTTPWriter: &respWriterTest{
				Headers: emptyHeader,
				Status:  http.StatusPreconditionFailed,
				Resp:    nil,
			},
		},
		{
			name: "should be ok to not find index document when index document is enabled",
			fields: fields{
				s3Context: &s3clientTest{
					HeadErr: s3client.ErrNotFound,
					GetResult: &s3client.GetOutput{
						Body:        &fakeIndexIoReadCloser,
						ContentType: "text/html; charset=utf-8",
					},
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
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						IndexDocument: "index.html",
					}},
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
				input: &GetInput{RequestPath: "/folder/"},
			},
			expectedS3ClientHeadCalled: true,
			expectedS3ClientHeadInput:  "/folder/index.html",
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
			name: "should fail to find and load index document with unknown error on head file",
			fields: fields{
				s3Context: &s3clientTest{
					HeadErr: errors.New("error"),
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						IndexDocument: "index.html",
					}},
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
				input: &GetInput{RequestPath: "/folder/"},
			},
			expectedS3ClientHeadCalled:              true,
			expectedS3ClientHeadInput:               "/folder/index.html",
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
		{
			name: "should fail to find and load index document with not found error on get file",
			fields: fields{
				s3Context: &s3clientTest{
					HeadResult: &s3client.HeadOutput{
						Type: "FILE",
						Key:  "/folder/index.html",
					},
					GetErr: s3client.ErrNotFound,
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						IndexDocument: "index.html",
					}},
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
				input: &GetInput{RequestPath: "/folder/"},
			},
			expectedS3ClientHeadCalled:   true,
			expectedS3ClientHeadInput:    "/folder/index.html",
			expectedS3ClientGetCalled:    true,
			expectedS3ClientGetInput:     &s3client.GetInput{Key: "/folder/index.html"},
			expectedHTTPWriter:           &respWriterTest{},
			expectedHandleNotFoundCalled: true,
		},
		{
			name: "should fail to find and load index document with error on get file",
			fields: fields{
				s3Context: &s3clientTest{
					HeadResult: &s3client.HeadOutput{
						Type: "FILE",
						Key:  "/folder/index.html",
					},
					GetErr: errors.New("test-error"),
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						IndexDocument: "index.html",
					}},
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
				input: &GetInput{RequestPath: "/folder/"},
			},
			expectedS3ClientHeadCalled:              true,
			expectedS3ClientHeadInput:               "/folder/index.html",
			expectedS3ClientGetCalled:               true,
			expectedS3ClientGetInput:                &s3client.GetInput{Key: "/folder/index.html"},
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
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			expectedS3ClientGetCalled: true,
			expectedS3ClientGetInput:  &s3client.GetInput{Key: "/folder/index.html"},
			expectedHTTPWriter: &respWriterTest{
				Headers: h,
				Status:  http.StatusOK,
				Resp:    []byte("fake-index.html-content"),
			},
		},
		{
			name: "should return a 304 error when S3 return a 304 error",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: s3client.ErrNotModified,
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			expectedS3ClientGetCalled: true,
			expectedS3ClientGetInput:  &s3client.GetInput{Key: "/folder/index.html"},
			expectedHTTPWriter: &respWriterTest{
				Headers: emptyHeader,
				Status:  http.StatusNotModified,
				Resp:    nil,
			},
		},
		{
			name: "should return a 412 error when S3 return a 412 error",
			fields: fields{
				s3Context: &s3clientTest{
					GetErr: s3client.ErrPreconditionFailed,
				},
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			expectedS3ClientGetCalled: true,
			expectedS3ClientGetInput:  &s3client.GetInput{Key: "/folder/index.html"},
			expectedHTTPWriter: &respWriterTest{
				Headers: emptyHeader,
				Status:  http.StatusPreconditionFailed,
				Resp:    nil,
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
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			expectedS3ClientGetCalled:    true,
			expectedS3ClientGetInput:     &s3client.GetInput{Key: "/folder/index.html"},
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
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			expectedS3ClientGetCalled:               true,
			expectedS3ClientGetInput:                &s3client.GetInput{Key: "/folder/index.html"},
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
		{
			name: "should be ok to get file with key rewrite",
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
					KeyRewriteList: []*config.TargetKeyRewriteConfig{{
						SourceRegex: regexp.MustCompile(`^/folder/index\.html$`),
						Target:      "/fake/fake.html",
					}},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{}},
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
				input: &GetInput{RequestPath: "/folder/index.html"},
			},
			expectedS3ClientGetCalled: true,
			expectedS3ClientGetInput:  &s3client.GetInput{Key: "/fake/fake.html"},
			expectedHTTPWriter: &respWriterTest{
				Headers: h,
				Status:  http.StatusOK,
			},
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
			rctx.Get(tt.args.input)
			if handleNotFoundCalled != tt.expectedHandleNotFoundCalled {
				t.Errorf("requestContext.Get() => handleNotFoundCalled = %+v, want %+v", handleNotFoundCalled, tt.expectedHandleNotFoundCalled)
			}
			if handleInternalServerErrorCalled != tt.expectedHandleInternalServerErrorCalled {
				t.Errorf("requestContext.Get() => handleInternalServerErrorCalled = %+v, want %+v", handleInternalServerErrorCalled, tt.expectedHandleInternalServerErrorCalled)
			}
			if handleForbiddenCalled != tt.expectedHandleForbiddenCalled {
				t.Errorf("requestContext.Get() => handleForbiddenCalled = %+v, want %+v", handleForbiddenCalled, tt.expectedHandleForbiddenCalled)
			}
			if tt.expectedS3ClientHeadCalled != tt.fields.s3Context.(*s3clientTest).HeadCalled {
				t.Errorf("requestContext.Get() => s3client.HeadCalled = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).HeadCalled, tt.expectedS3ClientHeadCalled)
			}
			if !reflect.DeepEqual(tt.expectedS3ClientHeadInput, tt.fields.s3Context.(*s3clientTest).HeadInput) {
				t.Errorf("requestContext.Get() => s3client.HeadInput = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).HeadInput, tt.expectedS3ClientHeadInput)
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
