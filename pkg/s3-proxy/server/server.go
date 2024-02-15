package server

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"emperror.dev/errors"
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
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/version"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook"
	"github.com/thoas/go-funk"
)

type Server struct {
	logger          log.Logger
	cfgManager      config.Manager
	metricsCl       metrics.Client
	server          *http.Server
	tracingSvc      tracing.Service
	s3clientManager s3client.Manager
	webhookManager  webhook.Manager
}

func NewServer(
	logger log.Logger,
	cfgManager config.Manager,
	metricsCl metrics.Client,
	tracingSvc tracing.Service,
	s3clientManager s3client.Manager,
	webhookManager webhook.Manager,
) *Server {
	return &Server{
		logger:          logger,
		cfgManager:      cfgManager,
		metricsCl:       metricsCl,
		tracingSvc:      tracingSvc,
		s3clientManager: s3clientManager,
		webhookManager:  webhookManager,
	}
}

func (svr *Server) Listen() error {
	svr.logger.Infof("Server listening on %s", svr.server.Addr)

	var err error

	// Listen (either HTTPS or HTTP)
	if svr.server.TLSConfig != nil {
		err = svr.server.ListenAndServeTLS("", "")
	} else {
		err = svr.server.ListenAndServe()
	}

	// Check error
	if err != nil {
		return errors.WithStack(err)
	}

	// Default
	return nil
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
	server := &http.Server{ //nolint: gosec // Set after
		Addr:    addr,
		Handler: r,
	}

	// Inject timeouts
	err = injectServerTimeout(server, cfg.Server.Timeouts)
	// Check error
	if err != nil {
		return err
	}

	// Get the TLS configuration (if necessary).
	tlsConfig, err := generateTLSConfig(cfg.Server.SSL, svr.logger)
	if err != nil {
		return errors.Wrap(err, "failed to create TLS configuration for server")
	}

	server.TLSConfig = tlsConfig

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
	authenticationSvc := authentication.NewAuthenticationService(cfg, svr.cfgManager, svr.metricsCl)

	// Create router
	r := chi.NewRouter()

	// Check if we need to enabled the compress middleware
	if cfg.Server.Compress != nil && *cfg.Server.Compress.Enabled {
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
	))
	r.Use(log.HTTPAddLoggerToContextMiddleware())
	r.Use(svr.metricsCl.Instrument("business", cfg.Metrics))
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
			// Add oidc endpoints
			err := authenticationSvc.OIDCEndpoints(k, v, r)
			// Check error
			if err != nil {
				return nil, err
			}
		}
	}

	notFoundHandler := func(w http.ResponseWriter, r *http.Request) {
		// Answer with general not found handler
		responsehandler.GeneralNotFoundError(r, w, svr.cfgManager)
	}

	internalServerHandlerGen := func(err error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Answer with general internal server error handler
			responsehandler.GeneralInternalServerError(
				r,
				w,
				svr.cfgManager,
				errors.WithStack(err),
			)
		}
	}

	// Create host router
	hr := NewHostRouter(notFoundHandler, internalServerHandlerGen)

	// Load main route only if main bucket path support option isn't enabled
	if cfg.ListTargets.Enabled {
		// Create new router
		rt := chi.NewRouter()
		// Add middleware in order to add response handler
		rt.Use(responsehandler.HTTPMiddleware(svr.cfgManager, ""))
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
				rt2 = rt2.With(authorization.Middleware(svr.cfgManager, svr.metricsCl))

				rt2.Get("/", func(_ http.ResponseWriter, req *http.Request) {
					// Get response handler
					resHan := responsehandler.GetResponseHandlerFromContext(req.Context())

					// Handle target list
					resHan.TargetList()
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
	for targetKey, tgt := range cfg.Targets {
		// Manage domain
		domain := tgt.Mount.Host
		if domain == "" {
			domain = "*"
		}
		// Get router from hostrouter if exists
		rt := hr.Get(domain)
		// Check nil
		if rt == nil {
			// Create a new router
			rt = chi.NewRouter()
		}
		// Loop over path list
		funk.ForEach(tgt.Mount.Path, func(path string) {
			rt.Route(path, func(rt2 chi.Router) {
				// Add middleware in order to add response handler
				rt2.Use(responsehandler.HTTPMiddleware(svr.cfgManager, targetKey))

				// Add Bucket request context middleware to initialize it
				rt2.Use(bucket.HTTPMiddleware(tgt, path, svr.s3clientManager, svr.webhookManager))

				// Add authentication middleware to router
				rt2.Use(authenticationSvc.Middleware(tgt.Resources))

				// Add authorization middleware to router
				rt2.Use(authorization.Middleware(svr.cfgManager, svr.metricsCl))

				// Check if GET action is enabled
				if tgt.Actions.GET != nil && tgt.Actions.GET.Enabled {
					// Add GET method to router
					rt2.Get("/*", func(_ http.ResponseWriter, req *http.Request) {
						// Get bucket request context
						brctx := bucket.GetBucketRequestContextFromContext(req.Context())
						// Get response handler
						resHan := responsehandler.GetResponseHandlerFromContext(req.Context())

						// Get request path
						requestPath := chi.URLParam(req, "*")

						// Unescape it
						// Found a bug where sometimes the request path isn't unescaped
						requestPath, err := url.PathUnescape(requestPath)
						// Check error
						if err != nil {
							resHan.InternalServerError(brctx.LoadFileContent, errors.WithStack(err))

							return
						}

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
								resHan.BadRequestError(brctx.LoadFileContent, errors.WithStack(err))

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
								resHan.BadRequestError(brctx.LoadFileContent, errors.WithStack(err))

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
					rt2.Put("/*", func(_ http.ResponseWriter, req *http.Request) {
						// Get bucket request context
						brctx := bucket.GetBucketRequestContextFromContext(req.Context())
						// Get response handler
						resHan := responsehandler.GetResponseHandlerFromContext(req.Context())

						// Get request path
						requestPath := chi.URLParam(req, "*")
						// Unescape it
						// Found a bug where sometimes the request path isn't unescaped
						requestPath, err := url.PathUnescape(requestPath)
						// Check error
						if err != nil {
							resHan.InternalServerError(brctx.LoadFileContent, errors.WithStack(err))

							return
						}

						// Parse form
						err = req.ParseForm()
						// Check error
						if err != nil {
							resHan.InternalServerError(brctx.LoadFileContent, errors.WithStack(err))

							return
						}

						// Parse multipart form
						err = req.ParseMultipartForm(0)
						if err != nil {
							resHan.InternalServerError(brctx.LoadFileContent, errors.WithStack(err))

							return
						}
						// Get file from form
						file, fileHeader, err := req.FormFile("file")
						if err != nil {
							resHan.InternalServerError(brctx.LoadFileContent, errors.WithStack(err))

							return
						}
						// Defer close file
						defer file.Close()
						// Defer remove all form
						defer req.MultipartForm.RemoveAll() //nolint: errcheck // Ignored

						// Create input for put request
						inp := &bucket.PutInput{
							RequestPath: requestPath,
							Filename:    fileHeader.Filename,
							Body:        file,
							ContentType: fileHeader.Header.Get("Content-Type"),
							ContentSize: fileHeader.Size,
						}
						// Action
						brctx.Put(req.Context(), inp)
					})
				}

				// Check if DELETE action is enabled
				if tgt.Actions.DELETE != nil && tgt.Actions.DELETE.Enabled {
					// Add DELETE method to router
					rt2.Delete("/*", func(_ http.ResponseWriter, req *http.Request) {
						// Get bucket request context
						brctx := bucket.GetBucketRequestContextFromContext(req.Context())
						// Get response handler
						resHan := responsehandler.GetResponseHandlerFromContext(req.Context())
						// Get request path
						requestPath := chi.URLParam(req, "*")
						// Unescape it
						// Found a bug where sometimes the request path isn't unescaped
						requestPath, err := url.PathUnescape(requestPath)
						// Check error
						if err != nil {
							resHan.InternalServerError(brctx.LoadFileContent, errors.WithStack(err))

							return
						}
						// Proxy DELETE Request
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
