package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/version"
)

// Main package

func main() {
	// Create new logger
	logger := log.NewLogger()

	// Create configuration manager
	cfgManager := config.NewManager(logger)

	// Load configuration
	err := cfgManager.Load()
	if err != nil {
		logger.Fatal(err)
	}

	// Get configuration
	cfg := cfgManager.GetConfig()
	// Configure logger
	err = logger.Configure(cfg.Log.Level, cfg.Log.Format, cfg.Log.FilePath)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Debug("Configuration successfully loaded and logger configured")

	// Getting version
	v := version.GetVersion()
	logger.Infof("Starting s3-proxy version: %s (git commit: %s) built on %s", v.Version, v.GitCommit, v.BuildDate)

	// Generate metrics instance
	metricsCtx := metrics.NewClient()

	// Listen
	go internalServe(logger, cfg, metricsCtx)
	serve(logger, cfg, metricsCtx)
}

func internalServe(logger log.Logger, cfg *config.Config, metricsCtx metrics.Client) {
	r := server.GenerateInternalRouter(logger, metricsCtx)
	// Create server
	addr := cfg.InternalServer.ListenAddr + ":" + strconv.Itoa(cfg.InternalServer.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	logger.Infof("Server listening on %s", addr)

	err := server.ListenAndServe()
	// Check if error exists
	if err != nil {
		logger.Fatalf("Unable to start http server: %v", err)
		os.Exit(1)
	}
}

func serve(logger log.Logger, cfg *config.Config, metricsCtx metrics.Client) {
	// Generate router
	r, err := server.GenerateRouter(logger, cfg, metricsCtx)
	if err != nil {
		logger.Fatalf("Unable to setup http server: %v", err)
		os.Exit(1)
	}

	// Create server
	addr := cfg.Server.ListenAddr + ":" + strconv.Itoa(cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	logger.Infof("Server listening on %s", addr)

	err = server.ListenAndServe()
	// Check if error exists
	if err != nil {
		logger.Fatalf("Unable to start http server: %v", err)
		os.Exit(1)
	}
}
