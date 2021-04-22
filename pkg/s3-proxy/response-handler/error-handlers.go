package responsehandler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
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

	// Store helpers template configs
	var helpersCfgItems []*config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if have a template override
		if targetCfg != nil &&
			targetCfg.Templates != nil {
			// Save override
			tplCfgItem = targetCfg.Templates.BadRequest
			helpersCfgItems = targetCfg.Templates.Helpers
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		helpersCfgItems,
		cfg.Templates.BadRequest,
		cfg.Templates.Helpers,
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

	// Store helpers template configs
	var helpersCfgItems []*config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if have a template override
		if targetCfg != nil &&
			targetCfg.Templates != nil {
			// Save override
			tplCfgItem = targetCfg.Templates.Forbidden
			helpersCfgItems = targetCfg.Templates.Helpers
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		helpersCfgItems,
		cfg.Templates.Forbidden,
		cfg.Templates.Helpers,
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

	// Store helpers template configs
	var helpersCfgItems []*config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg := cfg.Targets[h.targetKey]
		// Check if have a template override
		if targetCfg != nil &&
			targetCfg.Templates != nil {
			// Save override
			tplCfgItem = targetCfg.Templates.NotFound
			helpersCfgItems = targetCfg.Templates.Helpers
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		helpersCfgItems,
		cfg.Templates.NotFound,
		cfg.Templates.Helpers,
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

	// Store helpers template configs
	var helpersCfgItems []*config.TargetTemplateConfigItem

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
			helpersCfgItems = targetCfg.Templates.Helpers
		}
	}

	// Call generic template handler
	h.handleGenericErrorTemplate(
		loadFileContent,
		err,
		tplCfgItem,
		helpersCfgItems,
		cfg.Templates.Unauthorized,
		cfg.Templates.Helpers,
		http.StatusUnauthorized,
	)
}

func (h *handler) handleGenericErrorTemplate(
	loadFileContent func(ctx context.Context, path string) (string, error),
	err error,
	tplCfgItem *config.TargetTemplateConfigItem,
	helpersTplCfgItems []*config.TargetTemplateConfigItem,
	baseTplFilePath string,
	helpersTplFilePathList []string,
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

	// Put error err2 to avoid erase of err
	var err2 error

	// Get helpers template content
	tplContent, err2 = h.loadAndConcatTemplateContents(
		ctx,
		loadFileContent,
		helpersTplCfgItems,
		helpersTplFilePathList,
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

	// Check if a target template configuration exists
	// Note: Done like this and not with list to avoid creating list of 1 element
	// and to avoid loops etc to save potential memory and cpu
	if tplCfgItem != nil {
		// Load template content
		tpl, err3 := h.loadTemplateContent(
			ctx,
			loadFileContent,
			tplCfgItem,
		)
		// Concat
		tplContent = tplContent + "\n" + tpl
		// Save error
		err2 = err3
	} else {
		// Get template from general configuration
		tpl, err3 := loadLocalFileContent(baseTplFilePath)
		// Concat
		tplContent = tplContent + "\n" + tpl
		// Save error
		err2 = err3
	}

	// Check if error exists
	if err2 != nil {
		// Return an internal server error
		h.InternalServerError(
			loadFileContent,
			err2,
		)

		return
	}

	// Create data
	data := errorData{
		Request: h.req,
		User:    models.GetAuthenticatedUserFromContext(h.req.Context()),
		Error:   err,
	}

	// Execute template
	err2 = h.templateExecution(tplContent, data, statusCode)
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

	// Put error err2 to avoid erase of err
	var err2 error

	// Store target config
	var targetCfg *config.TargetConfig

	// Store internal server error template config
	var tplCfgItem *config.TargetTemplateConfigItem

	// Store helpers template configs
	var helpersCfgItems []*config.TargetTemplateConfigItem

	// Check if a target has been involve in this request
	if h.targetKey != "" {
		// Get target from key
		targetCfg = cfg.Targets[h.targetKey]
		// Check if target config and templates exist
		if targetCfg != nil && targetCfg.Templates != nil {
			// Save data
			tplCfgItem = targetCfg.Templates.InternalServerError
			helpersCfgItems = targetCfg.Templates.Helpers
		}
	}

	// Get helpers template content
	tplContent, err2 = h.loadAndConcatTemplateContents(
		ctx,
		loadFileContent,
		helpersCfgItems,
		cfg.Templates.Helpers,
	)

	// Check if error 2 doesn't exist
	if err2 == nil {
		// Check if target config and template exists
		if tplCfgItem != nil {
			// Load template content
			tplContent, err2 = h.loadTemplateContent(
				ctx,
				loadFileContent,
				targetCfg.Templates.InternalServerError,
			)
		} else {
			// Get template from general configuration
			tplContent, err2 = loadLocalFileContent(cfg.Templates.InternalServerError)
		}
	}

	// Check if error 2 doesn't exist
	// Note: this second if will manage second loading phase
	if err2 == nil {
		// Create data
		data := errorData{
			Request: h.req,
			User:    models.GetAuthenticatedUserFromContext(h.req.Context()),
			Error:   err,
		}

		// Execute template
		err2 = h.templateExecution(tplContent, data, http.StatusInternalServerError)
	}

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
