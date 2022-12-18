package s3client

import (
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awsv2credentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsv2s3 "github.com/aws/aws-sdk-go-v2/service/s3"
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
	// TODO Find a better way
	// ctx := context.TODO()

	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	// cfg, err := awsconfig.LoadDefaultConfig(
	// 	ctx,
	// 	awsconfig.WithRegion(tgt.Bucket.Region),
	// 	awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)),
	// 	awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
	// 		return aws.Endpoint{
	// 			PartitionID:       "aws",
	// 			URL:               "http://127.0.0.1:9000",
	// 			SigningRegion:     region,
	// 			HostnameImmutable: true,
	// 		}, nil
	// 	})),
	// )
	// if err != nil {
	// 	return nil, errors.WithStack(err)
	// }

	cfg := awsv2.Config{
		Region: tgt.Bucket.Region,
	}

	// Load credentials if they exists
	if tgt.Bucket.Credentials != nil && tgt.Bucket.Credentials.AccessKey != nil && tgt.Bucket.Credentials.SecretKey != nil {
		cfg.Credentials = awsv2credentials.NewStaticCredentialsProvider(
			tgt.Bucket.Credentials.AccessKey.Value,
			tgt.Bucket.Credentials.SecretKey.Value,
			"",
		)
	}
	// Load custom endpoint if it exists
	if tgt.Bucket.S3Endpoint != "" {
		cfg.EndpointResolverWithOptions = awsv2.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (awsv2.Endpoint, error) {
			return awsv2.Endpoint{
				PartitionID:       "aws",
				URL:               tgt.Bucket.S3Endpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		})
	}
	// Create s3 aws configuration
	s3cl := awsv2s3.NewFromConfig(cfg, func(o *awsv2s3.Options) {
		o.UsePathStyle = tgt.Bucket.S3Endpoint != ""
	})

	return &s3Context{
		svcClient:  s3cl,
		target:     tgt,
		metricsCtx: metricsCtx,
	}, nil
}
