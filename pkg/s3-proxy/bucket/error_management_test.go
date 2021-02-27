// +build unit

package bucket

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
)

func Test_requestContext_HandleInternalServerError(t *testing.T) {
	err := errors.New("fake")
	thrownErr := errors.New("fake err")
	handleInternalServerErrorCalled := false
	handleInternalServerErrorTmpl := ""
	var handleInternalServerErrorErr error
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
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
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleNotFoundCalled := false
	handleNotFoundTmpl := ""
	handleNotFoundWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
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
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleForbiddenCalled := false
	handleForbiddenTmpl := ""
	handleForbiddenWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
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
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleBadRequestCalled := false
	handleBadRequestTmpl := ""
	var handleBadRequestErr error
	handleBadRequestWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
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
	handleInternalServerErrorWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
		handleInternalServerErrorTmpl = tplString
		handleInternalServerErrorCalled = true
		handleInternalServerErrorErr = err
	}

	handleUnauthorizedCalled := false
	handleUnauthorizedTmpl := ""
	handleUnauthorizedWithTemplate := func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
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
