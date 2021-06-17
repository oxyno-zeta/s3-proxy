package bucket

import (
	"context"
	"path"
)

func (rctx *requestContext) HandleInternalServerError(ctx context.Context, err error, requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.InternalServerError != nil {
		// Put error err2 to avoid erase of err
		var err2 error
		content, err2 = rctx.loadTemplateContent(ctx, rctx.targetCfg.Templates.InternalServerError)
		// Check if error exists
		if err2 != nil {
			// This is a particular case. In this case, remove old error and manage new one
			err = err2
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleInternalServerErrorWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath, err)
}

func (rctx *requestContext) HandleNotFound(ctx context.Context, requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.NotFound != nil {
		// Declare error
		var err error
		// Try to get file from bucket
		content, err = rctx.loadTemplateContent(ctx, rctx.targetCfg.Templates.NotFound)
		if err != nil {
			rctx.HandleInternalServerError(ctx, err, requestPath)

			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleNotFoundWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath)
}

func (rctx *requestContext) HandleForbidden(ctx context.Context, requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.Forbidden != nil {
		// Declare error
		var err error
		// Try to get file from bucket
		content, err = rctx.loadTemplateContent(ctx, rctx.targetCfg.Templates.Forbidden)
		if err != nil {
			rctx.HandleInternalServerError(ctx, err, requestPath)

			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleForbiddenWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath)
}

func (rctx *requestContext) HandleBadRequest(ctx context.Context, err error, requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.BadRequest != nil {
		// Declare error
		var err2 error
		// Try to get file from bucket
		content, err2 = rctx.loadTemplateContent(ctx, rctx.targetCfg.Templates.BadRequest)
		if err2 != nil {
			rctx.HandleInternalServerError(ctx, err2, requestPath)

			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleBadRequestWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath, err)
}

func (rctx *requestContext) HandleUnauthorized(ctx context.Context, requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.Unauthorized != nil {
		// Declare error
		var err error
		// Try to get file from bucket
		content, err = rctx.loadTemplateContent(ctx, rctx.targetCfg.Templates.Unauthorized)
		if err != nil {
			rctx.HandleInternalServerError(ctx, err, requestPath)

			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleUnauthorizedWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath)
}
