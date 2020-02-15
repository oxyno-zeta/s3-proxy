package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/server"
	"github.com/oxyno-zeta/s3-proxy/pkg/version"
	"github.com/sirupsen/logrus"
)

// Main package

func main() {
	// Create logger
	logger := logrus.New()
	// Set JSON as default for the moment
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Load configuration from file
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(err)
		os.Exit(1)
	}

	// Configure logger
	err = config.ConfigureLogger(logger, cfg.Log)
	if err != nil {
		logger.Fatal(err)
		os.Exit(1)
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

func internalServe(logger logrus.FieldLogger, cfg *config.Config, metricsCtx metrics.Client) {
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

func serve(logger logrus.FieldLogger, cfg *config.Config, metricsCtx metrics.Client) {
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
