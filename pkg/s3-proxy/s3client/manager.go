package s3client

import (
	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
		actualKeys, _ := actualKeysInt.([]string)
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

func newClient(tgt *config.TargetConfig, metricsCtx metrics.Client) (Client, error) {
	sessionConfig := &aws.Config{
		Region: aws.String(tgt.Bucket.Region),
	}
	// Load credentials if they exists
	if tgt.Bucket.Credentials != nil && tgt.Bucket.Credentials.AccessKey != nil && tgt.Bucket.Credentials.SecretKey != nil {
		sessionConfig.Credentials = credentials.NewStaticCredentials(tgt.Bucket.Credentials.AccessKey.Value, tgt.Bucket.Credentials.SecretKey.Value, "")
	}
	// Load custom endpoint if it exists
	if tgt.Bucket.S3Endpoint != "" {
		sessionConfig.Endpoint = aws.String(tgt.Bucket.S3Endpoint)
		sessionConfig.S3ForcePathStyle = aws.Bool(true)
	}
	// Check if ssl needs to be disabled
	if tgt.Bucket.DisableSSL {
		sessionConfig.DisableSSL = aws.Bool(true)
	}
	// Create session
	sess, err := session.NewSession(sessionConfig)
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// Create s3 client
	svcClient := s3.New(sess)

	return &s3Context{
		svcClient:  svcClient,
		target:     tgt,
		metricsCtx: metricsCtx,
	}, nil
}
