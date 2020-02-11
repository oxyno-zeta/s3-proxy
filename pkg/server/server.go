package server

import (
	"net/http"

	"github.com/dimiro1/health"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/hostrouter"
	"github.com/oxyno-zeta/s3-proxy/pkg/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/utils"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

// GenerateInternalRouter Generate internal router
func GenerateInternalRouter(logger logrus.FieldLogger, metricsCtx metrics.Client) http.Handler {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.NewStructuredLogger(logger))
	r.Use(metricsCtx.Instrument())
	r.Use(middleware.Recoverer)

	healthHandler := health.NewHandler()
	// Listen path
	r.Handle("/metrics", metricsCtx.GetExposeHandler())
	r.Handle("/health", healthHandler)

	return r
}

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

// GenerateRouter Generate router
func GenerateRouter(logger logrus.FieldLogger, cfg *config.Config, metricsCtx metrics.Client) (http.Handler, error) {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.NewStructuredLogger(logger))
	r.Use(metricsCtx.Instrument())
	r.Use(middleware.Recoverer)
	// Check if auth if enabled and oidc enabled
	if cfg.AuthProviders != nil && cfg.AuthProviders.OIDC != nil {
		for _, v := range cfg.AuthProviders.OIDC {
			err := middlewares.OIDCEndpoints(v, cfg.Templates, r)
			if err != nil {
				return nil, err
			}
		}
	}

	r.NotFound(func(rw http.ResponseWriter, req *http.Request) {
		logEntry := middlewares.GetLogEntry(req)
		path := req.URL.RequestURI()
		utils.HandleNotFound(rw, path, logEntry, cfg.Templates)
	})

	// Create host router
	hr := hostrouter.New()

	// Load main route only if main bucket path support option isn't enabled
	if cfg.ListTargets.Enabled {
		// Create new router
		rt := chi.NewRouter()
		// Make list of resources from resource
		resources := make([]*config.Resource, 0)
		if cfg.ListTargets.Resource != nil {
			resources = append(resources, cfg.ListTargets.Resource)
		}
		// Manage path for list targets feature
		// Loop over path list
		funk.ForEach(cfg.ListTargets.Mount.Path, func(path string) {
			rt.Route(path, func(rt2 chi.Router) {
				rt2 = rt2.With(middlewares.AuthMiddleware(cfg, resources))
				rt2.Get("/", func(rw http.ResponseWriter, req *http.Request) {
					logEntry := middlewares.GetLogEntry(req)
					generateTargetList(rw, logEntry, cfg)
				})
			})
		})
		// Create domain
		domain := cfg.ListTargets.Mount.Host
		if domain == "" {
			domain = "*"
		}
		// Mount domain from configuration
		hr.Map(domain, rt)
	}

	// Load all targets routes
	funk.ForEach(cfg.Targets, func(tgt *config.TargetConfig) {
		// Manage domain
		domain := tgt.Mount.Host
		if domain == "" {
			domain = "*"
		}
		// Get router from hostrouter if exists
		rt := hr[domain]
		if rt == nil {
			// Create a new router
			rt = chi.NewRouter()
		}
		// Loop over path list
		funk.ForEach(tgt.Mount.Path, func(path string) {
			rt.Route(path, func(rt2 chi.Router) {
				// Add auth middleware to router
				rt2 = rt2.With(middlewares.AuthMiddleware(cfg, tgt.Resources))

				// Check if GET action is enabled
				if tgt.Actions.GET != nil && tgt.Actions.GET.Enabled {
					// Add GET method to router
					rt2.Get("/*", func(rw http.ResponseWriter, req *http.Request) {
						// Get request path
						requestPath := chi.URLParam(req, "*")
						// Get logger
						logEntry := middlewares.GetLogEntry(req)
						// Generate new bucket client
						brctx, err := bucket.NewClient(tgt, cfg.Templates, logEntry, path, rw, utils.HandleNotFound,
							utils.HandleInternalServerError, utils.HandleForbidden, metricsCtx)
						if err != nil {
							logEntry.Error(err)
							utils.HandleInternalServerError(rw, err, requestPath, logEntry, cfg.Templates)
							// Stop
							return
						}
						// Proxy GET Request
						brctx.Get(requestPath)
					})
				}

				// Check if PUT action is enabled
				if tgt.Actions.PUT != nil && tgt.Actions.PUT.Enabled {
					// Add PUT method to router
					rt2.Put("/*", func(rw http.ResponseWriter, req *http.Request) {
						// Get request path
						requestPath := chi.URLParam(req, "*")
						// Get logger
						logEntry := middlewares.GetLogEntry(req)
						if err := req.ParseForm(); err != nil {
							logEntry.Error(err)
							utils.HandleInternalServerError(rw, err, requestPath, logEntry, cfg.Templates)
							return
						}
						// Parse multipart form
						err := req.ParseMultipartForm(defaultMaxMemory)
						if err != nil {
							logEntry.Error(err)
							utils.HandleInternalServerError(rw, err, requestPath, logEntry, cfg.Templates)
							return
						}
						// Get file from form
						file, fileHeader, err := req.FormFile("file")
						if err != nil {
							logEntry.Error(err)
							utils.HandleInternalServerError(rw, err, requestPath, logEntry, cfg.Templates)
							return
						}
						// Generate new bucket client
						brctx, err := bucket.NewClient(tgt, cfg.Templates, logEntry, path, rw, utils.HandleNotFound,
							utils.HandleInternalServerError, utils.HandleForbidden, metricsCtx)
						if err != nil {
							logEntry.Error(err)
							utils.HandleInternalServerError(rw, err, requestPath, logEntry, cfg.Templates)
							// Stop
							return
						}
						// Create input for put request
						inp := &bucket.PutInput{
							RequestPath: requestPath,
							Filename:    fileHeader.Filename,
							Body:        file,
							ContentType: fileHeader.Header.Get("Content-Type"),
						}
						brctx.Put(inp)
					})
				}

				// Check if DELETE action is enabled
				if tgt.Actions.DELETE != nil && tgt.Actions.DELETE.Enabled {
					// Add DELETE method to router
					rt2.Delete("/*", func(rw http.ResponseWriter, req *http.Request) {
						// Get request path
						requestPath := chi.URLParam(req, "*")
						// Get logger
						logEntry := middlewares.GetLogEntry(req)
						// Generate new bucket client
						brctx, err := bucket.NewClient(tgt, cfg.Templates, logEntry, path, rw, utils.HandleNotFound,
							utils.HandleInternalServerError, utils.HandleForbidden, metricsCtx)
						if err != nil {
							logEntry.Error(err)
							utils.HandleInternalServerError(rw, err, requestPath, logEntry, cfg.Templates)
							// Stop
							return
						}
						// Proxy GET Request
						brctx.Delete(requestPath)
					})
				}
			})
		})
		// Mount domain from target
		hr.Map(domain, rt)
	})

	// Mount host router
	r.Mount("/", hr)

	return r, nil
}

func generateTargetList(rw http.ResponseWriter, logger logrus.FieldLogger, cfg *config.Config) {
	err := utils.TemplateExecution(cfg.Templates.TargetList, logger, rw, struct{ Targets []*config.TargetConfig }{Targets: cfg.Targets}, 200)
	if err != nil {
		logger.Error(err)
		utils.HandleInternalServerError(rw, err, "/", logger, cfg.Templates)
		// Stop here
		return
	}
}
