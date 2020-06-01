package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/hostrouter"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/authentication"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/authorization"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/thoas/go-funk"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

type Server struct {
	logger     log.Logger
	cfgManager config.Manager
	metricsCl  metrics.Client
	server     *http.Server
}

func NewServer(logger log.Logger, cfgManager config.Manager, metricsCl metrics.Client) *Server {
	return &Server{
		logger:     logger,
		cfgManager: cfgManager,
		metricsCl:  metricsCl,
	}
}

func (svr *Server) Listen() error {
	svr.logger.Infof("Server listening on %s", svr.server.Addr)
	err := svr.server.ListenAndServe()

	return err
}

func (svr *Server) GenerateServer() error {
	// Get configuration
	cfg := svr.cfgManager.GetConfig()
	// Generate router
	r, err := svr.generateRouter()
	if err != nil {
		return err
	}

	// Create server
	addr := cfg.Server.ListenAddr + ":" + strconv.Itoa(cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Prepare for configuration onChange
	svr.cfgManager.AddOnChangeHook(func() {
		// Generate router
		r, err2 := svr.generateRouter()
		if err2 != nil {
			svr.logger.Fatal(err2)
		}
		// Change server handler
		server.Handler = r
		svr.logger.Info("Server handler reloaded")
	})

	// Store server
	svr.server = server

	return nil
}

func (svr *Server) generateRouter() (http.Handler, error) {
	// Get configuration
	cfg := svr.cfgManager.GetConfig()
	// Create router
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.Compress(
		5,
		"text/html",
		"text/css",
		"text/plain",
		"text/javascript",
		"application/javascript",
		"application/x-javascript",
		"application/json",
		"application/atom+xml",
		"application/rss+xml",
		"image/svg+xml",
	))
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.NewStructuredLogger(svr.logger))
	r.Use(svr.metricsCl.Instrument())
	r.Use(middleware.Recoverer)
	// Check if auth if enabled and oidc enabled
	if cfg.AuthProviders != nil && cfg.AuthProviders.OIDC != nil {
		for _, v := range cfg.AuthProviders.OIDC {
			err := authentication.OIDCEndpoints(v, cfg.Templates, r)
			if err != nil {
				return nil, err
			}
		}
	}

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
				// Add authentication middleware to router
				rt2 = rt2.With(authentication.Middleware(cfg, resources))

				// Add authorization middleware to router
				rt2 = rt2.With(authorization.Middleware(cfg, cfg.Templates))

				rt2.Get("/", func(rw http.ResponseWriter, req *http.Request) {
					logEntry := middlewares.GetLogEntry(req)
					generateTargetList(rw, req.RequestURI, logEntry, cfg)
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
				// Add Bucket request context middleware to initialize it
				rt2.Use(middlewares.BucketRequestContext(tgt, cfg.Templates, path, svr.metricsCl))

				// Add authentication middleware to router
				rt2.Use(authentication.Middleware(cfg, tgt.Resources))

				// Add authorization middleware to router
				rt2.Use(authorization.Middleware(cfg, cfg.Templates))

				// Check if GET action is enabled
				if tgt.Actions.GET != nil && tgt.Actions.GET.Enabled {
					// Add GET method to router
					rt2.Get("/*", func(rw http.ResponseWriter, req *http.Request) {
						// Get bucket request context
						brctx := middlewares.GetBucketRequestContext(req)
						// Get request path
						requestPath := chi.URLParam(req, "*")
						// Proxy GET Request
						brctx.Get(requestPath)
					})
				}

				// Check if PUT action is enabled
				if tgt.Actions.PUT != nil && tgt.Actions.PUT.Enabled {
					// Add PUT method to router
					rt2.Put("/*", func(rw http.ResponseWriter, req *http.Request) {
						// Get bucket request context
						brctx := middlewares.GetBucketRequestContext(req)
						// Get request path
						requestPath := chi.URLParam(req, "*")
						// Get logger
						logEntry := middlewares.GetLogEntry(req)
						if err := req.ParseForm(); err != nil {
							logEntry.Error(err)
							brctx.HandleInternalServerError(err, path)
							return
						}
						// Parse multipart form
						err := req.ParseMultipartForm(defaultMaxMemory)
						if err != nil {
							logEntry.Error(err)
							brctx.HandleInternalServerError(err, path)
							return
						}
						// Get file from form
						file, fileHeader, err := req.FormFile("file")
						if err != nil {
							logEntry.Error(err)
							brctx.HandleInternalServerError(err, path)
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
						// Get bucket request context
						brctx := middlewares.GetBucketRequestContext(req)
						// Get request path
						requestPath := chi.URLParam(req, "*")
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

func generateTargetList(rw http.ResponseWriter, path string, logger log.Logger, cfg *config.Config) {
	err := utils.TemplateExecution(cfg.Templates.TargetList, "", logger, rw, struct{ Targets []*config.TargetConfig }{Targets: cfg.Targets}, 200)
	if err != nil {
		logger.Error(err)
		// ! In this case, use default default local files for error
		utils.HandleInternalServerError(logger, rw, cfg.Templates, path, err)
		// Stop here
		return
	}
}
