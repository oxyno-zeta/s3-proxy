package responsehandler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

type handler struct {
	req        *http.Request
	res        http.ResponseWriter
	cfgManager config.Manager
	targetKey  string
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

	// Load template
	tplContent, err := loadLocalFileContent(cfg.Templates.TargetList)
	// Check error
	if err != nil {
		h.InternalServerError(nil, err)

		return
	}

	// Create data structure
	// TODO Add user
	data := targetListData{
		Request: h.req,
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

	// Get template content
	var content string

	// Store error
	var err error

	// Check if per target template is declared
	if targetCfg != nil && targetCfg.Templates != nil &&
		targetCfg.Templates.FolderList != nil {
		// Load template content
		content, err = h.loadTemplateContent(
			ctx,
			loadFileContent,
			targetCfg.Templates.FolderList,
		)
	} else {
		// Load template
		content, err = loadLocalFileContent(cfg.Templates.FolderList)
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
