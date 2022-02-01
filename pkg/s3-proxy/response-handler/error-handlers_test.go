//go:build unit

package responsehandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGeneralBadRequestError(t *testing.T) {
	tests := []struct {
		name            string
		inputHeaders    map[string]string
		expectedBody    string
		expectedHeaders map[string][]string
	}{
		{
			name: "default case",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Bad Request</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask html",
			inputHeaders: map[string]string{
				"Accept": "text/html",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Bad Request</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask json",
			inputHeaders: map[string]string{
				"Accept": "application/json",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json; charset=utf-8"},
			},
			expectedBody: `{"error": "fake error"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Loop over input headers
			for k, v := range tt.inputHeaders {
				req.Header.Add(k, v)
			}

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(
				&config.Config{
					Templates: &config.TemplateConfig{
						Helpers: []string{
							"../../../templates/_helpers.tpl",
						},
						BadRequestError: &config.TemplateConfigItem{
							Path: "../../../templates/bad-request-error.tpl",
							Headers: map[string]string{
								"Content-Type": "{{ template \"main.headers.contentType\" . }}",
							},
							Status: "400",
						},
					},
				},
			)

			// Create fake error
			err := errors.New("fake error")

			// Call function
			GeneralBadRequestError(req, res, cfgManagerMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusBadRequest, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func TestGeneralForbiddenError(t *testing.T) {
	tests := []struct {
		name            string
		inputHeaders    map[string]string
		expectedBody    string
		expectedHeaders map[string][]string
	}{
		{
			name: "default case",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask html",
			inputHeaders: map[string]string{
				"Accept": "text/html",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask json",
			inputHeaders: map[string]string{
				"Accept": "application/json",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json; charset=utf-8"},
			},
			expectedBody: `{"error": "fake error"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Loop over input headers
			for k, v := range tt.inputHeaders {
				req.Header.Add(k, v)
			}

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(
				&config.Config{
					Templates: &config.TemplateConfig{
						Helpers: []string{
							"../../../templates/_helpers.tpl",
						},
						ForbiddenError: &config.TemplateConfigItem{
							Path: "../../../templates/forbidden-error.tpl",
							Headers: map[string]string{
								"Content-Type": "{{ template \"main.headers.contentType\" . }}",
							},
							Status: "403",
						},
					},
				},
			)

			// Create fake error
			err := errors.New("fake error")

			// Call function
			GeneralForbiddenError(req, res, cfgManagerMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusForbidden, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func TestGeneralUnauthorizedError(t *testing.T) {
	tests := []struct {
		name            string
		inputHeaders    map[string]string
		expectedBody    string
		expectedHeaders map[string][]string
	}{
		{
			name: "default case",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask html",
			inputHeaders: map[string]string{
				"Accept": "text/html",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask json",
			inputHeaders: map[string]string{
				"Accept": "application/json",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json; charset=utf-8"},
			},
			expectedBody: `{"error": "fake error"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Loop over input headers
			for k, v := range tt.inputHeaders {
				req.Header.Add(k, v)
			}

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(
				&config.Config{
					Templates: &config.TemplateConfig{
						Helpers: []string{
							"../../../templates/_helpers.tpl",
						},
						UnauthorizedError: &config.TemplateConfigItem{
							Path: "../../../templates/unauthorized-error.tpl",
							Headers: map[string]string{
								"Content-Type": "{{ template \"main.headers.contentType\" . }}",
							},
							Status: "401",
						},
					},
				},
			)

			// Create fake error
			err := errors.New("fake error")

			// Call function
			GeneralUnauthorizedError(req, res, cfgManagerMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusUnauthorized, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func TestGeneralNotFoundError(t *testing.T) {
	tests := []struct {
		name            string
		inputHeaders    map[string]string
		expectedBody    string
		expectedHeaders map[string][]string
	}{
		{
			name: "default case",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Not Found /</h1>
  </body>
</html>`,
		},
		{
			name: "input ask html",
			inputHeaders: map[string]string{
				"Accept": "text/html",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Not Found /</h1>
  </body>
</html>`,
		},
		{
			name: "input ask json",
			inputHeaders: map[string]string{
				"Accept": "application/json",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json; charset=utf-8"},
			},
			expectedBody: `{"error": "Not Found"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Loop over input headers
			for k, v := range tt.inputHeaders {
				req.Header.Add(k, v)
			}

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(
				&config.Config{
					Templates: &config.TemplateConfig{
						Helpers: []string{
							"../../../templates/_helpers.tpl",
						},
						NotFoundError: &config.TemplateConfigItem{
							Path: "../../../templates/not-found-error.tpl",
							Headers: map[string]string{
								"Content-Type": "{{ template \"main.headers.contentType\" . }}",
							},
							Status: "404",
						},
					},
				},
			)

			// Call function
			GeneralNotFoundError(req, res, cfgManagerMock)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusNotFound, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func TestGeneralInternalServerError(t *testing.T) {
	tests := []struct {
		name            string
		inputHeaders    map[string]string
		expectedBody    string
		expectedHeaders map[string][]string
	}{
		{
			name: "default case",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask html",
			inputHeaders: map[string]string{
				"Accept": "text/html",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "input ask json",
			inputHeaders: map[string]string{
				"Accept": "application/json",
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json; charset=utf-8"},
			},
			expectedBody: `{"error": "fake error"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Loop over input headers
			for k, v := range tt.inputHeaders {
				req.Header.Add(k, v)
			}

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(
				&config.Config{
					Templates: &config.TemplateConfig{
						Helpers: []string{
							"../../../templates/_helpers.tpl",
						},
						InternalServerError: &config.TemplateConfigItem{
							Path: "../../../templates/internal-server-error.tpl",
							Headers: map[string]string{
								"Content-Type": "{{ template \"main.headers.contentType\" . }}",
							},
							Status: "500",
						},
					},
				},
			)

			// Create fake error
			err := errors.New("fake error")

			// Call function
			GeneralInternalServerError(req, res, cfgManagerMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func Test_handler_handleGenericErrorTemplate(t *testing.T) {
	type args struct {
		tplCfgItem             *config.TargetTemplateConfigItem
		helpersTplCfgItems     []*config.TargetHelperConfigItem
		baseTpl                *config.TemplateConfigItem
		helpersTplFilePathList []string
	}
	tests := []struct {
		name                      string
		args                      args
		loadFileContentMockResult string
		loadFileContentMockError  error
		expectedBody              string
		expectedHeaders           map[string][]string
	}{
		{
			name: "loading helpers not found error (local file path)",
			args: args{
				helpersTplFilePathList: []string{
					"fake.tpl",
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>open fake.tpl: no such file or directory</p>
  </body>
</html>`,
		},
		{
			name: "loading helpers not found error (target override local path)",
			args: args{
				helpersTplCfgItems: []*config.TargetHelperConfigItem{
					{
						Path:     "fake.tpl",
						InBucket: false,
					},
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>open fake.tpl: no such file or directory</p>
  </body>
</html>`,
		},
		{
			name: "loading helpers not found error (target override s3 path)",
			args: args{
				helpersTplCfgItems: []*config.TargetHelperConfigItem{
					{
						Path:     "fake.tpl",
						InBucket: true,
					},
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			loadFileContentMockError: errors.New("not found"),
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>not found</p>
  </body>
</html>`,
		},
		{
			name: "loading template not found error (local file path)",
			args: args{
				helpersTplFilePathList: []string{
					"../../../templates/_helpers.tpl",
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "fake.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>open fake.tpl: no such file or directory</p>
  </body>
</html>`,
		},
		{
			name: "loading template not found error (target override local path)",
			args: args{
				helpersTplFilePathList: []string{
					"../../../templates/_helpers.tpl",
				},
				tplCfgItem: &config.TargetTemplateConfigItem{
					Path: "fake.tpl",
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>open fake.tpl: no such file or directory</p>
  </body>
</html>`,
		},
		{
			name: "loading template not found error (target override s3 path)",
			args: args{
				helpersTplFilePathList: []string{
					"../../../templates/_helpers.tpl",
				},
				tplCfgItem: &config.TargetTemplateConfigItem{
					Path:     "fake.tpl",
					InBucket: true,
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			loadFileContentMockError: errors.New("not found"),
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>not found</p>
  </body>
</html>`,
		},
		{
			name: "loading headers error (local file config)",
			args: args{
				helpersTplFilePathList: []string{
					"../../../templates/_helpers.tpl",
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"h1": "{{ .NotWorking }}",
					},
					Status: "400",
				},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>template: template-string-loaded:25:3: executing "template-string-loaded" at <.NotWorking>: can't evaluate field NotWorking in type *responsehandler.genericHeaderData</p>
  </body>
</html>`,
		},
		{
			name: "loading headers error (target override config)",
			args: args{
				helpersTplFilePathList: []string{
					"../../../templates/_helpers.tpl",
				},
				tplCfgItem: &config.TargetTemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"h1": "{{ .NotWorking }}",
					},
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>template: template-string-loaded:25:3: executing "template-string-loaded" at <.NotWorking>: can't evaluate field NotWorking in type *responsehandler.genericHeaderData</p>
  </body>
</html>`,
		},
		{
			name: "execute main template error (coming from s3)",
			args: args{
				helpersTplFilePathList: []string{
					"../../../templates/_helpers.tpl",
				},
				tplCfgItem: &config.TargetTemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"h1": "fake",
					},
					InBucket: true,
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/bad-request-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "400",
				},
			},
			loadFileContentMockResult: "{{ .NotWorking }}",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>template: template-string-loaded:25:3: executing "template-string-loaded" at <.NotWorking>: can't evaluate field NotWorking in type *responsehandler.errorData</p>
  </body>
</html>`,
		},
		{
			name: "loading from S3 with a working template",
			args: args{
				helpersTplFilePathList: []string{
					"../../../templates/_helpers.tpl",
				},
				tplCfgItem: &config.TargetTemplateConfigItem{
					Path: "../../../templates/internal-server-error.tpl",
					Headers: map[string]string{
						"h1": "fake",
					},
					InBucket: true,
				},
				baseTpl: &config.TemplateConfigItem{
					Path: "../../../templates/internal-server-error.tpl",
					Headers: map[string]string{
						"Content-Type": "{{ template \"main.headers.contentType\" . }}",
					},
					Status: "500",
				},
			},
			loadFileContentMockResult: "{{ .Error }}",
			expectedHeaders: map[string][]string{
				"H1": {"fake"},
			},
			expectedBody: `fake error`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(
				&config.Config{
					Templates: &config.TemplateConfig{
						Helpers: []string{
							"../../../templates/_helpers.tpl",
						},
						InternalServerError: &config.TemplateConfigItem{
							Path: "../../../templates/internal-server-error.tpl",
							Headers: map[string]string{
								"Content-Type": "{{ template \"main.headers.contentType\" . }}",
							},
						},
					},
				},
			)

			// load file content mock
			loadFileContentMock := func(ctx context.Context, path string) (string, error) {
				return tt.loadFileContentMockResult, tt.loadFileContentMockError
			}

			// Fake error
			err := errors.New("fake error")

			h := &handler{
				req:        req,
				res:        res,
				cfgManager: cfgManagerMock,
			}
			h.handleGenericErrorTemplate(
				loadFileContentMock,
				err,
				tt.args.tplCfgItem,
				tt.args.helpersTplCfgItems,
				tt.args.baseTpl,
				tt.args.helpersTplFilePathList,
			)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func Test_handler_BadRequestError(t *testing.T) {
	generalTemplateCfg := &config.TemplateConfig{
		Helpers: []string{
			"../../../templates/_helpers.tpl",
		},
		BadRequestError: &config.TemplateConfigItem{
			Path: "../../../templates/bad-request-error.tpl",
			Headers: map[string]string{
				"Content-Type": "{{ template \"main.headers.contentType\" . }}",
			},
			Status: "400",
		},
	}

	type fields struct {
		targetKey string
	}
	tests := []struct {
		name                      string
		fields                    fields
		loadFileContentMockResult string
		loadFileContentMockError  error
		expectedBody              string
		expectedHeaders           map[string][]string
	}{
		{
			name: "load from local fs",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Bad Request</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "load from s3",
			fields: fields{
				targetKey: "b1",
			},
			loadFileContentMockResult: "{{ .Error }}",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `fake error`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
				Templates: generalTemplateCfg,
				Targets: map[string]*config.TargetConfig{
					"b1": {
						Templates: &config.TargetTemplateConfig{
							BadRequestError: &config.TargetTemplateConfigItem{
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
								InBucket: true,
							},
						},
					},
				},
			})

			// Create error
			err := errors.New("fake error")

			// load file content mock
			loadFileContentMock := func(ctx context.Context, path string) (string, error) {
				return tt.loadFileContentMockResult, tt.loadFileContentMockError
			}

			h := &handler{
				req:        req,
				res:        res,
				cfgManager: cfgManagerMock,
				targetKey:  tt.fields.targetKey,
			}
			h.BadRequestError(loadFileContentMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusBadRequest, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func Test_handler_ForbiddenError(t *testing.T) {
	generalTemplateCfg := &config.TemplateConfig{
		Helpers: []string{
			"../../../templates/_helpers.tpl",
		},
		ForbiddenError: &config.TemplateConfigItem{
			Path: "../../../templates/forbidden-error.tpl",
			Headers: map[string]string{
				"Content-Type": "{{ template \"main.headers.contentType\" . }}",
			},
			Status: "403",
		},
	}

	type fields struct {
		targetKey string
	}
	tests := []struct {
		name                      string
		fields                    fields
		loadFileContentMockResult string
		loadFileContentMockError  error
		expectedBody              string
		expectedHeaders           map[string][]string
	}{
		{
			name: "load from local fs",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "load from s3",
			fields: fields{
				targetKey: "b1",
			},
			loadFileContentMockResult: "{{ .Error }}",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `fake error`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
				Templates: generalTemplateCfg,
				Targets: map[string]*config.TargetConfig{
					"b1": {
						Templates: &config.TargetTemplateConfig{
							ForbiddenError: &config.TargetTemplateConfigItem{
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
								InBucket: true,
							},
						},
					},
				},
			})

			// Create error
			err := errors.New("fake error")

			// load file content mock
			loadFileContentMock := func(ctx context.Context, path string) (string, error) {
				return tt.loadFileContentMockResult, tt.loadFileContentMockError
			}

			h := &handler{
				req:        req,
				res:        res,
				cfgManager: cfgManagerMock,
				targetKey:  tt.fields.targetKey,
			}
			h.ForbiddenError(loadFileContentMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusForbidden, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func Test_handler_NotFoundError(t *testing.T) {
	generalTemplateCfg := &config.TemplateConfig{
		Helpers: []string{
			"../../../templates/_helpers.tpl",
		},
		NotFoundError: &config.TemplateConfigItem{
			Path: "../../../templates/not-found-error.tpl",
			Headers: map[string]string{
				"Content-Type": "{{ template \"main.headers.contentType\" . }}",
			},
			Status: "404",
		},
	}

	type fields struct {
		targetKey string
	}
	tests := []struct {
		name                      string
		fields                    fields
		loadFileContentMockResult string
		loadFileContentMockError  error
		expectedBody              string
		expectedHeaders           map[string][]string
	}{
		{
			name: "load from local fs",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Not Found /</h1>
  </body>
</html>`,
		},
		{
			name: "load from s3",
			fields: fields{
				targetKey: "b1",
			},
			loadFileContentMockResult: "{{ .Error }}",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `Not Found`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
				Templates: generalTemplateCfg,
				Targets: map[string]*config.TargetConfig{
					"b1": {
						Templates: &config.TargetTemplateConfig{
							NotFoundError: &config.TargetTemplateConfigItem{
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
								InBucket: true,
							},
						},
					},
				},
			})

			// load file content mock
			loadFileContentMock := func(ctx context.Context, path string) (string, error) {
				return tt.loadFileContentMockResult, tt.loadFileContentMockError
			}

			h := &handler{
				req:        req,
				res:        res,
				cfgManager: cfgManagerMock,
				targetKey:  tt.fields.targetKey,
			}
			h.NotFoundError(loadFileContentMock)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusNotFound, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func Test_handler_UnauthorizedError(t *testing.T) {
	generalTemplateCfg := &config.TemplateConfig{
		Helpers: []string{
			"../../../templates/_helpers.tpl",
		},
		UnauthorizedError: &config.TemplateConfigItem{
			Path: "../../../templates/unauthorized-error.tpl",
			Headers: map[string]string{
				"Content-Type": "{{ template \"main.headers.contentType\" . }}",
			},
			Status: "401",
		},
	}

	type fields struct {
		targetKey string
	}
	tests := []struct {
		name                      string
		fields                    fields
		loadFileContentMockResult string
		loadFileContentMockError  error
		expectedBody              string
		expectedHeaders           map[string][]string
	}{
		{
			name: "load from local fs",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
    <p>fake error</p>
  </body>
</html>`,
		},
		{
			name: "load from s3",
			fields: fields{
				targetKey: "b1",
			},
			loadFileContentMockResult: "{{ .Error }}",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `fake error`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
				Templates: generalTemplateCfg,
				Targets: map[string]*config.TargetConfig{
					"b1": {
						Templates: &config.TargetTemplateConfig{
							UnauthorizedError: &config.TargetTemplateConfigItem{
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
								InBucket: true,
							},
						},
					},
				},
			})

			// Create error
			err := errors.New("fake error")

			// load file content mock
			loadFileContentMock := func(ctx context.Context, path string) (string, error) {
				return tt.loadFileContentMockResult, tt.loadFileContentMockError
			}

			h := &handler{
				req:        req,
				res:        res,
				cfgManager: cfgManagerMock,
				targetKey:  tt.fields.targetKey,
			}
			h.UnauthorizedError(loadFileContentMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusUnauthorized, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}

func Test_handler_InternalServerError(t *testing.T) {
	type fields struct {
		targetKey string
	}
	tests := []struct {
		name                      string
		fields                    fields
		inputHeaders              map[string]string
		loadFileContentMockResult string
		loadFileContentMockError  error
		expectedBody              string
		expectedHeaders           map[string][]string
	}{
		{
			name:                      "loading template from s3",
			fields:                    fields{targetKey: "b1"},
			loadFileContentMockResult: "{{ .Error }}",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `fake error`,
		},
		{
			name:                      "last case error management",
			fields:                    fields{targetKey: "b1"},
			loadFileContentMockResult: "{{ .NotWorking }}",
			expectedHeaders: map[string][]string{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>template: template-string-loaded:25:3: executing "template-string-loaded" at <.NotWorking>: can't evaluate field NotWorking in type responsehandler.errorData</p>
  </body>
</html>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest(
				"GET",
				"http://fake.com/",
				nil,
			)

			// Loop over input headers
			for k, v := range tt.inputHeaders {
				req.Header.Add(k, v)
			}

			// Add logger to request
			req = req.WithContext(log.SetLoggerInContext(req.Context(), log.NewLogger()))

			// Create fake response
			res := httptest.NewRecorder()

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().Return(
				&config.Config{
					Templates: &config.TemplateConfig{
						Helpers: []string{
							"../../../templates/_helpers.tpl",
						},
						InternalServerError: &config.TemplateConfigItem{
							Path: "../../../templates/internal-server-error.tpl",
							Headers: map[string]string{
								"Content-Type": "{{ template \"main.headers.contentType\" . }}",
							},
							Status: "500",
						},
					},
					Targets: map[string]*config.TargetConfig{
						"b1": {
							Templates: &config.TargetTemplateConfig{
								InternalServerError: &config.TargetTemplateConfigItem{
									Headers: map[string]string{
										"Content-Type": "{{ template \"main.headers.contentType\" . }}",
									},
									InBucket: true,
								},
							},
						},
					},
				},
			)

			// load file content mock
			loadFileContentMock := func(ctx context.Context, path string) (string, error) {
				return tt.loadFileContentMockResult, tt.loadFileContentMockError
			}

			// Create fake error
			err := errors.New("fake error")

			// Call function
			h := &handler{
				req:        req,
				res:        res,
				cfgManager: cfgManagerMock,
				targetKey:  tt.fields.targetKey,
			}
			h.InternalServerError(loadFileContentMock, err)

			// Get all res headers
			headers := map[string][]string{}
			for k, v := range res.HeaderMap {
				headers[k] = v
			}

			// Tests
			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, tt.expectedHeaders, headers)
			assert.Equal(t, tt.expectedBody, res.Body.String())
		})
	}
}
