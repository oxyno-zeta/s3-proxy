package responsehandler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils"
	"github.com/pkg/errors"
)

type handler struct {
	req        *http.Request
	res        http.ResponseWriter
	cfgManager config.Manager
	targetKey  string
}

func (h *handler) UpdateRequestAndResponse(req *http.Request, res http.ResponseWriter) {
	// Update request
	h.req = req
	// Update response
	h.res = res
}

func (h *handler) PreconditionFailed() {
	h.res.WriteHeader(http.StatusPreconditionFailed)
}

func (h *handler) NotModified() {
	h.res.WriteHeader(http.StatusNotModified)
}

func (h *handler) NoContent() {
	h.res.WriteHeader(http.StatusNoContent)
}

func (h *handler) TargetList() {
	// Get configuration
	cfg := h.cfgManager.GetConfig()

	// Get template content
	helpersTpl, err := h.loadAllHelpersContent(
		nil,
		nil,
		cfg.Templates.Helpers,
	)
	// Check error
	if err != nil {
		h.InternalServerError(nil, err)

		return
	}

	// Create header data
	hData := &genericHeaderData{
		Request: h.req,
		User:    models.GetAuthenticatedUserFromContext(h.req.Context()),
	}

	// Manage headers
	headers, err := h.manageHeaders(
		helpersTpl,
		cfg.Templates.TargetList.Headers,
		hData,
	)
	// Check if error exists
	if err != nil {
		// Return an internal server error
		h.InternalServerError(nil, err)

		return
	}

	// Load main template content
	tpl, err := loadLocalFileContent(cfg.Templates.TargetList.Path)
	// Check error
	if err != nil {
		h.InternalServerError(nil, err)

		return
	}
	// Concat
	tplContent := helpersTpl + "\n" + tpl

	// Create data structure
	data := targetListData{
		Request: h.req,
		User:    models.GetAuthenticatedUserFromContext(h.req.Context()),
		Targets: cfg.Targets,
	}

	// Execute main template
	bodyBuf, err := utils.ExecuteTemplate(tplContent, data)
	// Check error
	if err != nil {
		h.InternalServerError(nil, err)

		return
	}

	// Manage status code
	statusCode, err := h.manageStatus(helpersTpl, nil, cfg.Templates.TargetList.Status, data)
	// Check error
	if err != nil {
		h.InternalServerError(nil, err)

		return
	}

	// Send
	err = h.send(bodyBuf, headers, statusCode)
	// Check error
	if err != nil {
		h.InternalServerError(nil, err)

		return
	}
}

func (h *handler) RedirectWithTrailingSlash() {
	//  Get path
	p := h.req.URL.RequestURI()
	// Check if path doesn't start with /
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// Check if path doesn't end with /
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	// Redirect
	http.Redirect(h.res, h.req, p, http.StatusFound)
}

func (h *handler) StreamFile(
	loadFileContent func(ctx context.Context, path string) (string, error),
	input *StreamInput,
) error {
	// Get configuration
	cfg := h.cfgManager.GetConfig()
	// Get target configuration (exists in this case)
	targetCfg := cfg.Targets[h.targetKey]

	// Set headers from object
	setHeadersFromObjectOutput(h.res, input)

	// Check if headers templates are defined in the GET configuration
	if targetCfg.Actions != nil &&
		targetCfg.Actions.GET != nil &&
		targetCfg.Actions.GET.Config != nil &&
		targetCfg.Actions.GET.Config.StreamedFileHeaders != nil {
		// Target template helpers
		var tplHelpers []*config.TargetHelperConfigItem
		// Check if target templates are defined
		if targetCfg.Templates != nil {
			tplHelpers = targetCfg.Templates.Helpers
		}

		// Get template content
		helpersTpl, err := h.loadAllHelpersContent(
			loadFileContent,
			tplHelpers,
			cfg.Templates.Helpers,
		)
		// Check error
		if err != nil {
			return err
		}

		// Create data structure
		data := &streamFileHeaderData{
			Request:    h.req,
			User:       models.GetAuthenticatedUserFromContext(h.req.Context()),
			StreamFile: input,
		}
		// Manage headers
		headers, err := h.manageHeaders(helpersTpl, targetCfg.Actions.GET.Config.StreamedFileHeaders, data)
		// Check error
		if err != nil {
			return err
		}

		// Loop over them to add them to response
		for k, v := range headers {
			// Add them
			h.res.Header().Set(k, v)
		}
	}

	// Copy data stream to output stream
	_, err := io.Copy(h.res, input.Body)

	return errors.WithStack(err)
}

func (h *handler) FoldersFilesList(
	loadFileContent func(ctx context.Context, path string) (string, error),
	entries []*Entry,
) {
	// Get config
	cfg := h.cfgManager.GetConfig()

	// Get target configuration
	targetCfg := cfg.Targets[h.targetKey]

	// Helpers list
	var helpersCfgList []*config.TargetHelperConfigItem

	// Target template config item
	var tplConfigItem *config.TargetTemplateConfigItem

	// Get helpers template configs
	if targetCfg != nil && targetCfg.Templates != nil {
		// Save
		helpersCfgList = targetCfg.Templates.Helpers
		tplConfigItem = targetCfg.Templates.FolderList
	}

	// Get content from helpers
	// Note: separated because helpers and template are 2 different things and can be mixed
	helpersContent, err := h.loadAllHelpersContent(
		loadFileContent,
		helpersCfgList,
		cfg.Templates.Helpers,
	)
	// Check error
	if err != nil {
		h.InternalServerError(loadFileContent, err)
		// Stop
		return
	}

	// Store headers
	var headers map[string]string

	// Create header data
	hData := &genericHeaderData{
		Request: h.req,
		User:    models.GetAuthenticatedUserFromContext(h.req.Context()),
	}

	// Check if target config item exists
	if tplConfigItem != nil {
		// Manage headers
		headers, err = h.manageHeaders(
			helpersContent,
			tplConfigItem.Headers,
			hData,
		)
	} else {
		// Manage headers
		headers, err = h.manageHeaders(
			helpersContent,
			cfg.Templates.FolderList.Headers,
			hData,
		)
	}

	// Check if error exists
	if err != nil {
		// Return an internal server error
		h.InternalServerError(
			loadFileContent,
			err,
		)

		return
	}

	// Create bucket list data for templating
	data := &folderListingData{
		Request:    h.req,
		User:       models.GetAuthenticatedUserFromContext(h.req.Context()),
		Entries:    entries,
		BucketName: targetCfg.Bucket.Name,
		Name:       targetCfg.Name,
	}

	// Create main content
	content := helpersContent

	// Check if per target template is declared
	// Note: Done like this and not with list to avoid creating list of 1 element
	// and to avoid loops etc to save potential memory and cpu
	if tplConfigItem != nil {
		// Load template content
		tpl, err2 := h.loadTemplateContent(
			loadFileContent,
			tplConfigItem,
		)
		// Concat
		content = content + "\n" + tpl
		// Save error
		err = err2
	} else {
		// Load template
		tpl, err2 := loadLocalFileContent(cfg.Templates.FolderList.Path)
		// Concat
		content = content + "\n" + tpl
		// Save error
		err = err2
	}

	// Check error
	if err != nil {
		h.InternalServerError(loadFileContent, err)
		// Stop
		return
	}

	// Execute main template
	bodyBuf, err := utils.ExecuteTemplate(content, data)
	// Check error
	if err != nil {
		h.InternalServerError(loadFileContent, err)

		return
	}

	// Manage status code
	statusCode, err := h.manageStatus(helpersContent, tplConfigItem, cfg.Templates.FolderList.Status, data)
	// Check error
	if err != nil {
		h.InternalServerError(loadFileContent, err)

		return
	}

	// Send
	err = h.send(bodyBuf, headers, statusCode)
	// Check error
	if err != nil {
		h.InternalServerError(loadFileContent, err)

		return
	}
}
