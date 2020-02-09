package bucket

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
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
					&s3client.ListElementOutput{
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
				&Entry{
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
	handleNotFound := func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig) {
		handleNotFoundCalled = true
	}
	handleInternalServerError := func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorCalled = true
	}
	handleForbidden := func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig) {
		handleForbiddenCalled = true
	}
	type fields struct {
		s3Context                 s3client.Client
		logger                    logrus.FieldLogger
		bucketInstance            *config.TargetConfig
		tplConfig                 *config.TemplateConfig
		mountPath                 string
		httpRW                    http.ResponseWriter
		handleNotFound            func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
		handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
		handleForbidden           func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
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
				logger:    &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
			},
			args:                                    args{requestPath: ""},
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
		{
			name: "Can't delete a directory",
			fields: fields{
				s3Context: &s3clientTest{},
				logger:    &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
			},
			args:                                    args{requestPath: "/directory/"},
			expectedHTTPWriter:                      &respWriterTest{},
			expectedHandleInternalServerErrorCalled: true,
		},
		{
			name: "Can't delete file because of error",
			fields: fields{
				s3Context: &s3clientTest{
					Err: errors.New("test"),
				},
				logger: &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
				logger:    &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
				s3Context:                 tt.fields.s3Context,
				logger:                    tt.fields.logger,
				bucketInstance:            tt.fields.bucketInstance,
				tplConfig:                 tt.fields.tplConfig,
				mountPath:                 tt.fields.mountPath,
				httpRW:                    tt.fields.httpRW,
				handleNotFound:            tt.fields.handleNotFound,
				handleInternalServerError: tt.fields.handleInternalServerError,
				handleForbidden:           tt.fields.handleForbidden,
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
	handleNotFound := func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig) {
		handleNotFoundCalled = true
	}
	handleInternalServerError := func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig) {
		handleInternalServerErrorCalled = true
	}
	handleForbidden := func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig) {
		handleForbiddenCalled = true
	}
	type fields struct {
		s3Context                 s3client.Client
		logger                    logrus.FieldLogger
		bucketInstance            *config.TargetConfig
		tplConfig                 *config.TemplateConfig
		mountPath                 string
		httpRW                    http.ResponseWriter
		handleNotFound            func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
		handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
		handleForbidden           func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
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
					Err: errors.New("test"),
				},
				logger: &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket:  &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
					Err: errors.New("test"),
				},
				logger: &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
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
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
				logger:    &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
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
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
					Err: errors.New("test"),
				},
				logger: &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								AllowOverride: false,
							},
						},
					},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
				logger: &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								AllowOverride: false,
							},
						},
					},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
				logger: &logrus.Logger{},
				bucketInstance: &config.TargetConfig{
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						PUT: &config.PutActionConfig{
							Config: &config.PutActionConfigConfig{
								AllowOverride: false,
							},
						},
					},
				},
				tplConfig:                 &config.TemplateConfig{},
				mountPath:                 "/mount",
				httpRW:                    &respWriterTest{},
				handleNotFound:            handleNotFound,
				handleInternalServerError: handleInternalServerError,
				handleForbidden:           handleForbidden,
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
				s3Context:                 tt.fields.s3Context,
				logger:                    tt.fields.logger,
				bucketInstance:            tt.fields.bucketInstance,
				tplConfig:                 tt.fields.tplConfig,
				mountPath:                 tt.fields.mountPath,
				httpRW:                    tt.fields.httpRW,
				handleNotFound:            tt.fields.handleNotFound,
				handleInternalServerError: tt.fields.handleInternalServerError,
				handleForbidden:           tt.fields.handleForbidden,
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
				t.Errorf("requestContext.Head() => s3client.HeadCalled = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).HeadCalled, tt.expectedS3ClientHeadCalled)
			}
			if !reflect.DeepEqual(tt.expectedS3ClientHeadInput, tt.fields.s3Context.(*s3clientTest).HeadInput) {
				t.Errorf("requestContext.Head() => s3client.HeadInput = %+v, want %+v", tt.fields.s3Context.(*s3clientTest).HeadInput, tt.expectedS3ClientHeadInput)
			}
			if !reflect.DeepEqual(tt.expectedHTTPWriter, tt.fields.httpRW) {
				t.Errorf("requestContext.Put() => httpWriter = %+v, want %+v", tt.fields.httpRW, tt.expectedHTTPWriter)
			}
		})
	}
}
