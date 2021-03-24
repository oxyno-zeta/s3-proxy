package s3client

import (
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/thoas/go-funk"
)

type manager struct {
	targetClient map[string]Client
	cfgManager   config.Manager
	metricCl     metrics.Client
}

func (m *manager) GetClientForTarget(name string) Client {
	return m.targetClient[name]
}

func (m *manager) Load() error {
	// Get configuration
	cfg := m.cfgManager.GetConfig()

	// Store target keys
	tgtKeys := make([]string, 0)

	// Loop over all targets
	for key, tgt := range cfg.Targets {
		// Store key
		tgtKeys = append(tgtKeys, key)

		// Create new client
		cl, err := newClient(tgt, m.metricCl)
		// Check error
		if err != nil {
			return err
		}
		// Store client
		m.targetClient[key] = cl
	}

	// Get all keys from current object
	actualKeysInt := funk.Keys(m.targetClient)
	// Check if result exists or not
	if actualKeysInt != nil {
		// Cast it to string array
		actualKeys := actualKeysInt.([]string)
		// Get difference between those 2 array
		subtract := funk.SubtractString(actualKeys, tgtKeys)
		// Loop over subtract keys
		for _, key := range subtract {
			// Delete key inside actual object
			delete(m.targetClient, key)
		}
	}

	// Default
	return nil
}
