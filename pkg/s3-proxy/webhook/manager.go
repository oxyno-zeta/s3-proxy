package webhook

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"emperror.dev/errors"
	"github.com/go-resty/resty/v2"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/thoas/go-funk"
)

// HookNumberOfRedirect will contains the number of redirect that a hook can follow.
const HookNumberOfRedirect = 20

type manager struct {
	cfgManager config.Manager
	storageMap map[string]*hooksCfgStorage
	metricsSvc metrics.Client
}

type hooksCfgStorage struct {
	Get    []*hookStorage
	Put    []*hookStorage
	Delete []*hookStorage
}

type hookStorage struct {
	Client *resty.Client
	Config *config.WebhookConfig
}

func (m *manager) Load() error {
	// Get configuration
	cfg := m.cfgManager.GetConfig()

	// Store target keys
	tgtKeys := []string{}

	// Loop over the target map
	for k, targetCfg := range cfg.Targets {
		// Store target key
		tgtKeys = append(tgtKeys, k)

		// Create storage structure
		entry := &hooksCfgStorage{
			Get:    []*hookStorage{},
			Put:    []*hookStorage{},
			Delete: []*hookStorage{},
		}

		// Check if actions are present
		if targetCfg.Actions != nil {
			// Check if GET action is present and have a config
			if targetCfg.Actions.GET != nil && targetCfg.Actions.GET.Config != nil {
				// Create list
				list, err := m.createRestClients(targetCfg.Actions.GET.Config.Webhooks)
				// Check error
				if err != nil {
					return err
				}
				// Store
				entry.Get = list
			}

			// Check if PUT action is present and have a config
			if targetCfg.Actions.PUT != nil && targetCfg.Actions.PUT.Config != nil {
				// Create list
				list, err := m.createRestClients(targetCfg.Actions.PUT.Config.Webhooks)
				// Check error
				if err != nil {
					return err
				}
				// Store
				entry.Put = list
			}

			// Check if DELETE action is present and have a config
			if targetCfg.Actions.DELETE != nil && targetCfg.Actions.DELETE.Config != nil {
				// Create list
				list, err := m.createRestClients(targetCfg.Actions.DELETE.Config.Webhooks)
				// Check error
				if err != nil {
					return err
				}
				// Store
				entry.Delete = list
			}
		}

		// Save new entry
		m.storageMap[k] = entry
	}

	// Get all keys from current object
	actualKeysInt := funk.Keys(m.storageMap)
	// Check if result exists or not
	if actualKeysInt != nil {
		// Cast it to string array
		actualKeys, _ := actualKeysInt.([]string)
		// Get difference between those 2 array
		subtract := funk.SubtractString(actualKeys, tgtKeys)
		// Loop over subtract keys
		for _, key := range subtract {
			// Delete key inside actual object
			delete(m.storageMap, key)
		}
	}

	// Default return
	return nil
}

func (*manager) createRestClients(list []*config.WebhookConfig) ([]*hookStorage, error) {
	// Create result
	res := []*hookStorage{}

	// Loop over the list
	for _, it := range list {
		// Create client
		cli := resty.New()

		// Manage wait time
		if it.DefaultWaitTime != "" {
			// Parse duration
			dur, err := time.ParseDuration(it.DefaultWaitTime)
			// Check error
			if err != nil {
				return nil, errors.WithStack(err)
			}
			// Add it
			cli = cli.SetRetryWaitTime(dur)
		}

		// Manage max wait time
		if it.MaxWaitTime != "" {
			// Parse duration
			dur, err := time.ParseDuration(it.MaxWaitTime)
			// Check error
			if err != nil {
				return nil, errors.WithStack(err)
			}
			// Add it
			cli = cli.SetRetryMaxWaitTime(dur)
		}

		// Manage retry count
		if it.RetryCount != 0 {
			// Add
			cli = cli.SetRetryCount(it.RetryCount)
		}

		// Set redirect policy
		cli = cli.SetRedirectPolicy(resty.FlexibleRedirectPolicy(HookNumberOfRedirect))

		// Append
		res = append(res, &hookStorage{
			Client: cli,
			Config: it,
		})
	}

	return res, nil
}

func (m *manager) ManageDELETEHooks(
	ctx context.Context,
	targetKey, requestPath string,
	s3Metadata *S3Metadata,
) {
	// Separate functions to test logic without routine
	go m.manageDELETEHooksInternal(ctx, targetKey, requestPath, s3Metadata)
}

func (m *manager) manageDELETEHooksInternal(
	ctx context.Context,
	targetKey, requestPath string,
	s3Metadata *S3Metadata,
) {
	// Get logger
	logger := log.GetLoggerFromContext(ctx)

	// Get target storage
	sto := m.storageMap[targetKey]

	// Check if storage is empty
	if sto == nil || len(sto.Delete) == 0 {
		// Stop here
		logger.Debugf("No DELETE hook declared for target %s", targetKey)

		return
	}

	// Get hooks declared
	hookClients := sto.Delete

	// Create output metadata
	outputMetadata := &OutputMetadataHookBody{
		Bucket:     s3Metadata.Bucket,
		Region:     s3Metadata.Region,
		S3Endpoint: s3Metadata.S3Endpoint,
		Key:        s3Metadata.Key,
	}

	// Run hooks
	m.runHooks(
		ctx,
		requestPath,
		nil,
		outputMetadata,
		DELETEAction,
		targetKey,
		hookClients,
	)
}

func (m *manager) ManagePUTHooks(
	ctx context.Context,
	targetKey, requestPath string,
	metadata *PutInputMetadata,
	s3Metadata *S3Metadata,
) {
	// Separate functions to test logic without routine
	go m.managePUTHooksInternal(ctx, targetKey, requestPath, metadata, s3Metadata)
}

func (m *manager) managePUTHooksInternal(
	ctx context.Context,
	targetKey, requestPath string,
	metadata *PutInputMetadata,
	s3Metadata *S3Metadata,
) {
	// Get logger
	logger := log.GetLoggerFromContext(ctx)

	// Get target storage
	sto := m.storageMap[targetKey]

	// Check if storage is empty
	if sto == nil || len(sto.Put) == 0 {
		// Stop here
		logger.Debugf("No PUT hook declared for target %s", targetKey)

		return
	}

	// Get hooks declared
	hookClients := sto.Put

	// Create input metadata
	inputMetadata := &PutInputMetadataHookBody{
		Filename:    metadata.Filename,
		ContentType: metadata.ContentType,
		ContentSize: metadata.ContentSize,
	}

	// Create output metadata
	outputMetadata := &OutputMetadataHookBody{
		Bucket:     s3Metadata.Bucket,
		Region:     s3Metadata.Region,
		S3Endpoint: s3Metadata.S3Endpoint,
		Key:        s3Metadata.Key,
	}

	// Run hooks
	m.runHooks(
		ctx,
		requestPath,
		inputMetadata,
		outputMetadata,
		PUTAction,
		targetKey,
		hookClients,
	)
}

func (m *manager) ManageGETHooks(
	ctx context.Context,
	targetKey, requestPath string,
	metadata *GetInputMetadata,
	s3Metadata *S3Metadata,
) {
	// Separate functions to test logic without routine
	go m.manageGETHooksInternal(ctx, targetKey, requestPath, metadata, s3Metadata)
}

func (m *manager) manageGETHooksInternal(
	ctx context.Context,
	targetKey, requestPath string,
	metadata *GetInputMetadata,
	s3Metadata *S3Metadata,
) {
	// Get logger
	logger := log.GetLoggerFromContext(ctx)

	// Get target storage
	sto := m.storageMap[targetKey]

	// Check if storage is empty
	if sto == nil || len(sto.Get) == 0 {
		// Stop here
		logger.Debugf("No GET hook declared for target %s", targetKey)

		return
	}

	// Get hooks declared
	hookClients := sto.Get

	// Create input metadata
	inputMetadata := &GetInputMetadataHookBody{
		IfMatch:     metadata.IfMatch,
		IfNoneMatch: metadata.IfNoneMatch,
		Range:       metadata.Range,
	}
	// Manage if modified since
	if metadata.IfModifiedSince != nil {
		inputMetadata.IfModifiedSince = metadata.IfModifiedSince.Format(time.RFC3339)
	}
	// Manage if unmodified since
	if metadata.IfUnmodifiedSince != nil {
		inputMetadata.IfUnmodifiedSince = metadata.IfUnmodifiedSince.Format(time.RFC3339)
	}

	// Create output metadata
	outputMetadata := &OutputMetadataHookBody{
		Bucket:     s3Metadata.Bucket,
		Region:     s3Metadata.Region,
		S3Endpoint: s3Metadata.S3Endpoint,
		Key:        s3Metadata.Key,
	}

	// Run hooks
	m.runHooks(
		ctx,
		requestPath,
		inputMetadata,
		outputMetadata,
		GETAction,
		targetKey,
		hookClients,
	)
}

func (m *manager) runHooks(
	ctx context.Context,
	requestPath string,
	inputMetadata interface{},
	outputMetadata interface{},
	action, targetName string,
	hookClients []*hookStorage,
) {
	// Get logger
	logger := log.GetLoggerFromContext(ctx)

	// Get parent trace
	parentTrace := tracing.GetTraceFromContext(ctx)

	// Create body
	body := &HookBody{
		Action:         action,
		RequestPath:    requestPath,
		InputMetadata:  inputMetadata,
		OutputMetadata: outputMetadata,
		Target: &TargetHookBody{
			Name: targetName,
		},
	}

	// Need to create an intermediate function to manage defer properly
	executeOne := func(i int, st *hookStorage) {
		// Create specific logger
		spLogger := logger.WithFields(map[string]interface{}{
			"webhook_action": action,
			"webhook_number": i,
		})

		// Create child trace
		childTrace := parentTrace.GetChildTrace("webhook")
		childTrace.SetTag("webhook-url", st.Config.URL)
		childTrace.SetTag("webhook-method", st.Config.Method)

		defer childTrace.Finish()

		// Save client
		cl := st.Client.R()
		// Add all fixed headers
		for k, val := range st.Config.Headers {
			// Add header
			cl = cl.SetHeader(k, val)
		}
		// Add all secret headers
		for k, val := range st.Config.SecretHeaders {
			// Add header
			cl = cl.SetHeader(k, val.Value)
		}
		// Add content-type
		cl = cl.SetHeader("Content-Type", "application/json")
		// Add body
		cl = cl.SetBody(body)
		// Add trace to http header for forwarding
		err := childTrace.InjectInHTTPHeader(cl.Header)
		// Check error
		if err != nil {
			spLogger.Error(errors.WithStack(err))

			// Stop here
			return
		}
		// Log
		spLogger.Info("Executing webhook")
		// Execute request
		res, err := cl.Execute(st.Config.Method, st.Config.URL)
		// Check error
		if err != nil {
			// Log
			spLogger.Error(errors.WithStack(err))

			// Increase failed webhooks
			m.metricsSvc.IncFailedWebhooks(targetName, action)

			// Stop here
			return
		}
		// Add status code to logger
		spLogger = spLogger.WithField("webhook_status_code", strconv.Itoa(res.StatusCode()))
		// Check status code
		if res.StatusCode() >= http.StatusBadRequest {
			// Create error
			err := fmt.Errorf("%d - %s", res.StatusCode(), string(res.Body()))
			// Log
			spLogger.Error(errors.WithStack(err))

			// Increase failed webhooks
			m.metricsSvc.IncFailedWebhooks(targetName, action)

			// Stop here
			return
		}

		spLogger.Info("Webhook succeed")

		// Increase failed webhooks
		m.metricsSvc.IncSucceedWebhooks(targetName, action)
	}

	// Loop over clients to perform requests
	for i, st := range hookClients {
		executeOne(i, st)
	}
}
