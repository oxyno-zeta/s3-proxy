package responsehandler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/pkg/errors"
)

func GeneralBadRequestError(
	req *http.Request,
	res http.ResponseWriter,
	cfgManager config.Manager,
	err error,
) {
	// Create handler
	resHan := NewHandler(req, res, cfgManager, "")

	// Call bad request
	resHan.BadRequestError(nil, err)
}

func GeneralForbiddenError(
	req *http.Request,
	res http.ResponseWriter,
	cfgManager config.Manager,
	err error,
) {
	// Create handler
	resHan := NewHandler(req, res, cfgManager, "")

	// Call forbidden
	resHan.ForbiddenError(nil, err)
}

func GeneralUnauthorizedError(
	req *http.Request,
	res http.ResponseWriter,
	cfgManager config.Manager,
	err error,
) {
	// Create handler
	resHan := NewHandler(req, res, cfgManager, "")

	// Call unauthorized
	resHan.UnauthorizedError(nil, err)
}

func GeneralNotFoundError(
	req *http.Request,
	res http.ResponseWriter,
	cfgManager config.Manager,
) {
	// Create handler
	resHan := NewHandler(req, res, cfgManager, "")

	// Call not found
	resHan.NotFoundError(nil)
}

func GeneralInternalServerError(
	req *http.Request,
	res http.ResponseWriter,
	cfgManager config.Manager,
	err error,
) {
	// Create handler
	resHan := NewHandler(req, res, cfgManager, "")

	// Call internal server error
	resHan.InternalServerError(nil, err)
}

func (h *handler) BadRequestError(
	loadFileContent func(ctx context.Context, path string) (string, error),
	err error,
) {
	// Get configuration
	cfg := h.cfgManager.GetConfig()

	// Variable to save target template configuration item override
	var tplCfgItem *config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if have a template override
		if targetCfg != nil &&
			targetCfg.Templates != nil &&
			targetCfg.Templates.BadRequest != nil {
			// Save override
			tplCfgItem = targetCfg.Templates.BadRequest
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		cfg.Templates.BadRequest,
		http.StatusBadRequest,
	)
}

func (h *handler) ForbiddenError(
	loadFileContent func(ctx context.Context, path string) (string, error),
	err error,
) {
	// Get configuration
	cfg := h.cfgManager.GetConfig()

	// Variable to save target template configuration item override
	var tplCfgItem *config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if have a template override
		if targetCfg != nil &&
			targetCfg.Templates != nil &&
			targetCfg.Templates.Forbidden != nil {
			// Save override
			tplCfgItem = targetCfg.Templates.Forbidden
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		cfg.Templates.Forbidden,
		http.StatusForbidden,
	)
}

func (h *handler) NotFoundError(
	loadFileContent func(ctx context.Context, path string) (string, error),
) {
	// Get configuration
	cfg := h.cfgManager.GetConfig()

	// Create specific error
	err := errors.New("Not Found")

	// Variable to save target template configuration item override
	var tplCfgItem *config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if have a template override
		if targetCfg != nil &&
			targetCfg.Templates != nil &&
			targetCfg.Templates.NotFound != nil {
			// Save override
			tplCfgItem = targetCfg.Templates.NotFound
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		cfg.Templates.NotFound,
		http.StatusNotFound,
	)
}

func (h *handler) UnauthorizedError(
	loadFileContent func(ctx context.Context, path string) (string, error),
	err error,
) {
	// Get configuration
	cfg := h.cfgManager.GetConfig()

	// Variable to save target template configuration item override
	var tplCfgItem *config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if have a template override
		if targetCfg != nil &&
			targetCfg.Templates != nil &&
			targetCfg.Templates.Unauthorized != nil {
			// Save override
			tplCfgItem = targetCfg.Templates.Unauthorized
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		cfg.Templates.Unauthorized,
		http.StatusUnauthorized,
	)
}

func (h *handler) handleGenericErrorTemplate(
	loadFileContent func(ctx context.Context, path string) (string, error),
	err error,
	tplCfgItem *config.TargetTemplateConfigItem,
	baseTemplateFilePath string,
	statusCode int,
) {
	// Initialize content
	tplContent := ""
	// Get request context
	ctx := h.req.Context()
	// Get logger from request
	logger := log.GetLoggerFromContext(h.req.Context())

	// Log error
	logger.Error(err)

	// Check if a target template configuration exists
	if tplCfgItem != nil {
		// Put error err2 to avoid erase of err
		var err2 error
		// Load template content
		tplContent, err2 = h.loadTemplateContent(
			ctx,
			loadFileContent,
			tplCfgItem,
		)
		// Check if error exists
		if err2 != nil {
			// Return an internal server error
			h.InternalServerError(
				loadFileContent,
				err2,
			)

			return
		}
	}

	// Check if content isn't already filled
	if tplContent == "" {
		// Put error err2 to avoid erase of err
		var err2 error
		// Get template from general configuration
		tplContent, err2 = loadLocalFileContent(baseTemplateFilePath)
		// Check if error exists
		if err2 != nil {
			// Return an internal server error
			h.InternalServerError(
				loadFileContent,
				err2,
			)

			return
		}
	}

	// Create data
	// TODO Add user
	// TODO Add full request
	data := errorData{
		Request: h.generateRequestData(),
		Path:    h.req.URL.RequestURI(),
		Error:   err,
	}

	// Execute template
	err2 := h.templateExecution(tplContent, data, statusCode)
	// Check error
	if err2 != nil {
		// Return an internal server error
		h.InternalServerError(
			loadFileContent,
			err2,
		)
	}
}

func (h *handler) InternalServerError(
	loadFileContent func(ctx context.Context, path string) (string, error),
	err error,
) {
	// Initialize content
	tplContent := ""
	// Get request context
	ctx := h.req.Context()
	// Get config
	cfg := h.cfgManager.GetConfig()
	// Get logger from request
	logger := log.GetLoggerFromContext(h.req.Context())

	// Log error
	logger.Error(err)

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if file is in bucket
		if targetCfg != nil &&
			targetCfg.Templates != nil &&
			targetCfg.Templates.InternalServerError != nil {
			// Put error err2 to avoid erase of err
			var err2 error
			// Load template content
			tplContent, err2 = h.loadTemplateContent(
				ctx,
				loadFileContent,
				targetCfg.Templates.InternalServerError,
			)
			// Check if error exists
			if err2 != nil {
				// This is a particular case. In this case, remove old error and manage new one
				err = err2

				// Log error
				logger.Error(err)
			}
		}
	}

	// Check if content isn't already filled
	if tplContent == "" {
		// Put error err2 to avoid erase of err
		var err2 error
		// Get template from general configuration
		tplContent, err2 = loadLocalFileContent(cfg.Templates.InternalServerError)
		// Check if error exists
		if err2 != nil {
			// This is a particular case. In this case, remove old error and manage new one
			err = err2

			// Log error
			logger.Error(err)
		}
	}

	// Create data
	data := errorData{
		Request: h.generateRequestData(),
		Path:    h.req.URL.RequestURI(),
		Error:   err,
	}

	// Execute template
	err2 := h.templateExecution(tplContent, data, http.StatusInternalServerError)
	// Check error
	if err2 != nil {
		// New error
		logger.Error(err2)
		// Template error
		res := fmt.Sprintf(`
<!DOCTYPE html>
<html>
  <body>
	<h1>Internal Server Error</h1>
	<p>%s</p>
  </body>
</html>
`, err2)

		// Set the header
		h.res.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Set status code
		h.res.WriteHeader(http.StatusInternalServerError)
		// Write the buffer to the http.ResponseWriter
		_, _ = h.res.Write([]byte(res))
	}
}
