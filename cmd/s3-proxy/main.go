package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
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

	// Generate router
	r := server.GenerateRouter(logger, cfg)

	// Create server
	addr := cfg.Server.ListenAddr + ":" + strconv.Itoa(cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	logger.Infof("Server listening on %s", addr)
	err = server.ListenAndServe()
	// Listen
	if err != nil {
		logger.Fatalf("Unable to start http server: %v", err)
		os.Exit(1)
	}
}
