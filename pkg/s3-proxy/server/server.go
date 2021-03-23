package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httptracer"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/authentication"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/authorization"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/version"
	"github.com/thoas/go-funk"
)

type Server struct {
	logger     log.Logger
	cfgManager config.Manager
	metricsCl  metrics.Client
	server     *http.Server
	tracingSvc tracing.Service
}

func NewServer(logger log.Logger, cfgManager config.Manager, metricsCl metrics.Client, tracingSvc tracing.Service) *Server {
	return &Server{
		logger:     logger,
		cfgManager: cfgManager,
		metricsCl:  metricsCl,
		tracingSvc: tracingSvc,
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

	// Create authentication service
	authenticationSvc := authentication.NewAuthenticationService(cfg, svr.metricsCl)

	// Create router
	r := chi.NewRouter()

	// Check if we need to enabled the compress middleware
	if *cfg.Server.Compress.Enabled {
		r.Use(middleware.Compress(
			cfg.Server.Compress.Level,
			cfg.Server.Compress.Types...,
		))
	}

	// Check if no cache is enabled or not
	if cfg.Server.Cache == nil || cfg.Server.Cache.NoCacheEnabled {
		// Apply no cache
		r.Use(middleware.NoCache)
	} else {
		// Apply S3 proxy cache management middleware
		r.Use(middlewares.CacheManagement(cfg.Server.Cache))
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// Manage tracing
	// Create http tracer configuration
	httptraCfg := httptracer.Config{
		ServiceName:    "s3-proxy",
		ServiceVersion: version.GetVersion().Version,
		SampleRate:     1,
		OperationName:  "http.request",
		Tags:           cfg.Tracing.FixedTags,
	}
	// Put tracing middlewares
	r.Use(httptracer.Tracer(svr.tracingSvc.GetTracer(), httptraCfg))
	r.Use(middlewares.ImproveTracing())
	r.Use(log.NewStructuredLogger(
		svr.logger,
		tracing.GetTraceIDFromRequest,
		utils.ClientIP,
		utils.GetRequestURI,
	))
	r.Use(log.HTTPAddLoggerToContextMiddleware())
	r.Use(svr.metricsCl.Instrument("business"))
	// Recover panic
	r.Use(middleware.Recoverer)

	// Check if cors is enabled
	if cfg.Server != nil && cfg.Server.CORS != nil && cfg.Server.CORS.Enabled {
		// Generate CORS
		cc := generateCors(cfg.Server, svr.logger.GetCorsLogger())
		// Apply CORS handler
		r.Use(cc.Handler)
	}

	// Check if auth if enabled and oidc enabled
	if cfg.AuthProviders != nil && cfg.AuthProviders.OIDC != nil {
		for k, v := range cfg.AuthProviders.OIDC {
			err := authenticationSvc.OIDCEndpoints(k, v, r)
			if err != nil {
				return nil, err
			}
		}
	}

	notFoundHandler := func(w http.ResponseWriter, r *http.Request) {
		// Get logger
		logger := log.GetLoggerFromContext(r.Context())
		// Get request URI
		requestURI := r.URL.RequestURI()
		utils.HandleNotFound(logger, w, cfg.Templates, requestURI)
	}

	internalServerHandlerGen := func(err error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get logger
			logger := log.GetLoggerFromContext(r.Context())
			// Get request URI
			requestURI := r.URL.RequestURI()
			utils.HandleInternalServerError(logger, w, cfg.Templates, requestURI, err)
		}
	}

	// Create host router
	hr := NewHostRouter(notFoundHandler, internalServerHandlerGen)

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
				rt2 = rt2.With(authenticationSvc.Middleware(resources))

				// Add authorization middleware to router
				rt2 = rt2.With(authorization.Middleware(cfg, svr.metricsCl))

				rt2.Get("/", func(rw http.ResponseWriter, req *http.Request) {
					logEntry := log.GetLoggerFromContext(req.Context())
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
	for _, tgt := range cfg.Targets {
		// Manage domain
		domain := tgt.Mount.Host
		if domain == "" {
			domain = "*"
		}
		// Get router from hostrouter if exists
		rt := hr.Get(domain)
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
				rt2.Use(authenticationSvc.Middleware(tgt.Resources))

				// Add authorization middleware to router
				rt2.Use(authorization.Middleware(cfg, svr.metricsCl))

				// Check if GET action is enabled
				if tgt.Actions.GET != nil && tgt.Actions.GET.Enabled {
					// Add GET method to router
					rt2.Get("/*", func(rw http.ResponseWriter, req *http.Request) {
						// Get bucket request context
						brctx := middlewares.GetBucketRequestContext(req)

						// Get request path
						requestPath := chi.URLParam(req, "*")

						// Get ETag headers

						// Get If-Modified-Since as string
						ifModifiedSinceStr := req.Header.Get("If-Modified-Since")
						// Create result
						var ifModifiedSince *time.Time
						// Check if content exists
						if ifModifiedSinceStr != "" {
							// Parse time
							ifModifiedSinceTime, err := http.ParseTime(ifModifiedSinceStr)
							// Check error
							if err != nil {
								brctx.HandleBadRequest(req.Context(), err, requestPath)

								return
							}
							// Save result
							ifModifiedSince = &ifModifiedSinceTime
						}

						// Get Range
						byteRange := req.Header.Get("Range")

						// Get If-Match
						ifMatch := req.Header.Get("If-Match")

						// Get If-None-Match
						ifNoneMatch := req.Header.Get("If-None-Match")

						// Get If-Unmodified-Since as string
						ifUnmodifiedSinceStr := req.Header.Get("If-Unmodified-Since")
						// Create result
						var ifUnmodifiedSince *time.Time
						// Check if content exists
						if ifUnmodifiedSinceStr != "" {
							// Parse time
							ifUnmodifiedSinceTime, err := http.ParseTime(ifUnmodifiedSinceStr)
							// Check error
							if err != nil {
								brctx.HandleBadRequest(req.Context(), err, requestPath)

								return
							}
							// Save result
							ifUnmodifiedSince = &ifUnmodifiedSinceTime
						}

						// Proxy GET Request
						brctx.Get(req.Context(), &bucket.GetInput{
							RequestPath:       requestPath,
							IfModifiedSince:   ifModifiedSince,
							IfMatch:           ifMatch,
							IfNoneMatch:       ifNoneMatch,
							IfUnmodifiedSince: ifUnmodifiedSince,
							Range:             byteRange,
						})
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
						logEntry := log.GetLoggerFromContext(req.Context())
						if err := req.ParseForm(); err != nil {
							logEntry.Error(err)
							brctx.HandleInternalServerError(req.Context(), err, path)

							return
						}
						// Parse multipart form
						err := req.ParseMultipartForm(0)
						if err != nil {
							logEntry.Error(err)
							brctx.HandleInternalServerError(req.Context(), err, path)

							return
						}
						// Get file from form
						file, fileHeader, err := req.FormFile("file")
						if err != nil {
							logEntry.Error(err)
							brctx.HandleInternalServerError(req.Context(), err, path)

							return
						}
						// Create input for put request
						inp := &bucket.PutInput{
							RequestPath: requestPath,
							Filename:    fileHeader.Filename,
							Body:        file,
							ContentType: fileHeader.Header.Get("Content-Type"),
							ContentSize: fileHeader.Size,
						}
						brctx.Put(req.Context(), inp)
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
						brctx.Delete(req.Context(), requestPath)
					})
				}
			})
		})
		// Mount domain from target
		hr.Map(domain, rt)
	}

	// Mount host router
	r.Mount("/", hr)

	return r, nil
}

// Generate CORS.
func generateCors(cfg *config.ServerConfig, logger log.CorsLogger) *cors.Cors {
	// Check if allow all is enabled
	if cfg.CORS.AllowAll {
		cc := cors.AllowAll()
		// Add logger
		cc.Log = logger
		// Return
		return cc
	}

	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	corsOpt := cors.Options{}
	// Check if allowed origins exist
	if cfg.CORS.AllowOrigins != nil {
		corsOpt.AllowedOrigins = cfg.CORS.AllowOrigins
	}
	// Check if allowed methods exist
	if cfg.CORS.AllowMethods != nil {
		corsOpt.AllowedMethods = cfg.CORS.AllowMethods
	}
	// Check if allowed headers exist
	if cfg.CORS.AllowHeaders != nil {
		corsOpt.AllowedHeaders = cfg.CORS.AllowHeaders
	}
	// Check if exposed headers exist
	if cfg.CORS.ExposeHeaders != nil {
		corsOpt.ExposedHeaders = cfg.CORS.ExposeHeaders
	}
	// Check if allow credentials exist
	if cfg.CORS.AllowCredentials != nil {
		corsOpt.AllowCredentials = *cfg.CORS.AllowCredentials
	}
	// Check if max age exists
	// 300 = Maximum value not ignored by any of major browsers
	if cfg.CORS.MaxAge != nil {
		corsOpt.MaxAge = *cfg.CORS.MaxAge
	}
	// Check if debug option exists
	if cfg.CORS.Debug != nil {
		corsOpt.Debug = *cfg.CORS.Debug
	}
	// Check if Options Passthrough exists
	if cfg.CORS.OptionsPassthrough != nil {
		corsOpt.OptionsPassthrough = *cfg.CORS.OptionsPassthrough
	}

	cc := cors.New(corsOpt)
	// Add logger
	cc.Log = logger
	// Return
	return cc
}

func generateTargetList(rw http.ResponseWriter, path string, logger log.Logger, cfg *config.Config) {
	err := utils.TemplateExecution(cfg.Templates.TargetList, "", logger, rw, struct {
		Targets map[string]*config.TargetConfig
	}{Targets: cfg.Targets}, 200)
	if err != nil {
		logger.Error(err)
		// ! In this case, use default default local files for error
		utils.HandleInternalServerError(logger, rw, cfg.Templates, path, err)
		// Stop here
		return
	}
}
