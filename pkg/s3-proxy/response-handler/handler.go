package responsehandler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
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
	tplContent, err := h.loadAndConcatTemplateContents(
		context.TODO(), // Empty context
		nil,
		nil,
		// Here we can use the shortcut to append the template at the end
		// Because target list won't use any template from a bucket.
		append(cfg.Templates.Helpers, cfg.Templates.TargetList),
	)
	// Check error
	if err != nil {
		h.InternalServerError(nil, err)

		return
	}

	// Create data structure
	data := targetListData{
		Request: h.req,
		User:    models.GetAuthenticatedUserFromContext(h.req.Context()),
		Targets: cfg.Targets,
	}

	// Execute template
	err = h.templateExecution(tplContent, data, http.StatusOK)
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

func (h *handler) StreamFile(output *StreamInput) error {
	// Set headers from object
	setHeadersFromObjectOutput(h.res, output)
	// Copy data stream to output stream
	_, err := io.Copy(h.res, output.Body)

	return err
}

func (h *handler) FoldersFilesList(
	loadFileContent func(ctx context.Context, path string) (string, error),
	entries []*Entry,
) {
	// Get context
	ctx := h.req.Context()

	// Get config
	cfg := h.cfgManager.GetConfig()

	// Get target configuration
	targetCfg := cfg.Targets[h.targetKey]

	// Helpers list
	var helpersCfgList []*config.TargetTemplateConfigItem

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
	content, err := h.loadAndConcatTemplateContents(
		ctx,
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

	// Check if per target template is declared
	// Note: Done like this and not with list to avoid creating list of 1 element
	// and to avoid loops etc to save potential memory and cpu
	if tplConfigItem != nil {
		// Load template content
		tpl, err2 := h.loadTemplateContent(
			ctx,
			loadFileContent,
			tplConfigItem,
		)
		// Concat
		content = content + "\n" + tpl
		// Save error
		err = err2
	} else {
		// Load template
		tpl, err2 := loadLocalFileContent(cfg.Templates.FolderList)
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

	// Create bucket list data for templating
	data := &bucketListingData{
		Request:    h.req,
		User:       models.GetAuthenticatedUserFromContext(h.req.Context()),
		Entries:    entries,
		BucketName: targetCfg.Bucket.Name,
		Name:       targetCfg.Name,
		Path:       h.req.URL.RequestURI(),
	}

	// Execute template
	err = h.templateExecution(content, data, http.StatusOK)
	// Check error
	if err != nil {
		h.InternalServerError(loadFileContent, err)

		return
	}
}
