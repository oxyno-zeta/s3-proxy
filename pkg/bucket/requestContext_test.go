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
			if !reflect.DeepEqual(tt.expectedHTTPWriter, tt.fields.httpRW) {
				t.Errorf("requestContext.Delete() => httpWriter = %+v, want %+v", tt.fields.httpRW, tt.expectedHTTPWriter)
			}
		})
	}
}
