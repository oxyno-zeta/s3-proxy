package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/version"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook"
)

// Main package

func startServer(mainConfDir string) {
	// Create new logger
	logger := log.NewLogger()

	// Create configuration manager
	cfgManager := config.NewManager(logger)

	// Load configuration
	err := cfgManager.Load(mainConfDir)
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

	// Watch change for logger (special case)
	cfgManager.AddOnChangeHook(func() {
		// Get configuration
		cfg := cfgManager.GetConfig()
		// Configure logger
		err = logger.Configure(cfg.Log.Level, cfg.Log.Format, cfg.Log.FilePath)
		if err != nil {
			logger.Fatal(err)
		}
	})

	logger.Debug("Configuration successfully loaded and logger configured")

	// Getting version
	v := version.GetVersion()
	logger.Infof("Starting s3-proxy version: %s (git commit: %s) built on %s", v.Version, v.GitCommit, v.BuildDate)

	// Generate metrics instance
	metricsCtx := metrics.NewClient()

	// Generate tracing service instance
	tracingSvc, err := tracing.New(cfgManager, logger)
	// Check error
	if err != nil {
		logger.Fatal(err)
	}
	// Prepare on reload hook
	cfgManager.AddOnChangeHook(func() {
		err2 := tracingSvc.Reload()
		if err2 != nil {
			logger.Fatal(err2)
		}
	})

	// Create S3 client manager
	s3clientManager := s3client.NewManager(cfgManager, metricsCtx)
	// Log
	logger.Info("Load S3 clients for all targets")
	// Load configuration
	err = s3clientManager.Load()
	// Check error
	if err != nil {
		logger.Fatal(err)
	}
	// Prepare on reload hook
	cfgManager.AddOnChangeHook(func() {
		logger.Info("Reload S3 clients for all targets")
		// Load
		err2 := s3clientManager.Load()
		// Check error
		if err2 != nil {
			logger.Fatal(err2)
		}
	})

	// Create webhook manager
	webhookManager := webhook.NewManager(cfgManager, metricsCtx)
	// Load
	err = webhookManager.Load()
	// Check error
	if err != nil {
		logger.Fatal(err)
	}
	// Prepare on reload hook
	cfgManager.AddOnChangeHook(func() {
		logger.Info("Reload webhook clients for all targets")
		// Load
		err2 := webhookManager.Load()
		// Check error
		if err2 != nil {
			logger.Fatal(err2)
		}
	})

	// Create internal server
	intSvr := server.NewInternalServer(logger, cfgManager, metricsCtx)
	// Generate server
	err = intSvr.GenerateServer()
	if err != nil {
		logger.Fatal(err)
	}
	// Create server
	svr := server.NewServer(logger, cfgManager, metricsCtx, tracingSvc, s3clientManager, webhookManager)
	// Generate server
	err = svr.GenerateServer()
	if err != nil {
		logger.Fatal(err)
	}

	var g errgroup.Group

	g.Go(svr.Listen)
	g.Go(intSvr.Listen)

	if err := g.Wait(); err != nil {
		logger.Fatal(err)
	}
}

func main() {
	var configFolder string

	rootCmd := &cobra.Command{
		Use:   "s3-proxy",
		Short: "S3 Reverse Proxy",
		Long:  "S3 Reverse Proxy with GET, PUT and DELETE methods and authentication (OpenID Connect and Basic Auth)",
		Run: func(_ *cobra.Command, _ []string) {
			startServer(configFolder)
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of s3-proxy",
		Run: func(_ *cobra.Command, _ []string) {
			v := version.GetVersion()
			fmt.Printf("version: %s (git commit: %s) built on %s", v.Version, v.GitCommit, v.BuildDate)
		},
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVar(&configFolder, "config", "conf/", "Config folder (default is <Current Working Directory>/conf/)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
