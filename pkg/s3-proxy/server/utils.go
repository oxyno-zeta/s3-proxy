package server

import (
	"net/http"
	"time"

	"emperror.dev/errors"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

func injectServerTimeout(svr *http.Server, cfg *config.ServerTimeoutsConfig) error {
	// Check if configuration is empty
	if cfg == nil {
		// Ignore
		return nil
	}

	// Check if read timeout is set
	if cfg.ReadTimeout != "" {
		// Parse timeout
		dur, err := time.ParseDuration(cfg.ReadTimeout)
		// Check error
		if err != nil {
			return errors.WithStack(err)
		}

		// Inject
		svr.ReadTimeout = dur
	}

	// Check if read header timeout is set
	if cfg.ReadHeaderTimeout != "" {
		// Parse timeout
		dur, err := time.ParseDuration(cfg.ReadHeaderTimeout)
		// Check error
		if err != nil {
			return errors.WithStack(err)
		}

		// Inject
		svr.ReadHeaderTimeout = dur
	}

	// Check if write timeout is set
	if cfg.WriteTimeout != "" {
		// Parse timeout
		dur, err := time.ParseDuration(cfg.WriteTimeout)
		// Check error
		if err != nil {
			return errors.WithStack(err)
		}

		// Inject
		svr.WriteTimeout = dur
	}

	// Check if idle timeout is set
	if cfg.IdleTimeout != "" {
		// Parse timeout
		dur, err := time.ParseDuration(cfg.IdleTimeout)
		// Check error
		if err != nil {
			return errors.WithStack(err)
		}

		// Inject
		svr.IdleTimeout = dur
	}

	// Default
	return nil
}
