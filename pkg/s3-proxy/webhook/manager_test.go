//go:build unit

package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	mmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_manager_createRestClients(t *testing.T) {
	type testedContent struct {
		RetryWaitTime    time.Duration
		RetryMaxWaitTime time.Duration
		RetryCount       int
	}
	type args struct {
		list []*config.WebhookConfig
	}
	tests := []struct {
		name    string
		args    args
		want    []*testedContent
		wantErr bool
	}{
		{
			name: "should be ok to create one",
			args: args{
				list: []*config.WebhookConfig{{
					RetryCount:      1,
					MaxWaitTime:     "10s",
					DefaultWaitTime: "1s",
				}},
			},
			want: []*testedContent{{
				RetryCount:       1,
				RetryWaitTime:    time.Second,
				RetryMaxWaitTime: 10 * time.Second,
			}},
		},
		{
			name: "should be ok to create two",
			args: args{
				list: []*config.WebhookConfig{
					{
						RetryCount:      1,
						MaxWaitTime:     "10s",
						DefaultWaitTime: "1s",
					},
					{},
				},
			},
			want: []*testedContent{{
				RetryCount:       1,
				RetryWaitTime:    time.Second,
				RetryMaxWaitTime: 10 * time.Second,
			}, {
				RetryWaitTime:    100 * time.Millisecond,
				RetryMaxWaitTime: 2 * time.Second,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				storageMap: map[string]*hooksCfgStorage{},
			}
			got, err := m.createRestClients(tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("manager.createRestClients() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Len(t, got, len(tt.args.list))

			for i, w := range tt.want {
				assert.Equal(t, w.RetryCount, got[i].Client.RetryCount)
				assert.Equal(t, w.RetryWaitTime, got[i].Client.RetryWaitTime)
				assert.Equal(t, w.RetryMaxWaitTime, got[i].Client.RetryMaxWaitTime)
				assert.Equal(t, tt.args.list[i], got[i].Config)
			}
		})
	}
}

func Test_manager_Load(t *testing.T) {
	type storageTestContent struct {
		getLen               int
		putLen               int
		deleteLen            int
		getHookStorageCfg    []*config.WebhookConfig
		putHookStorageCfg    []*config.WebhookConfig
		deleteHookStorageCfg []*config.WebhookConfig
	}
	type fields struct {
		storageMap map[string]*hooksCfgStorage
	}
	tests := []struct {
		name           string
		fields         fields
		cfg            *config.Config
		wantErr        bool
		storageContent map[string]storageTestContent
	}{
		{
			name:           "should be ok without any config",
			cfg:            &config.Config{},
			fields:         fields{storageMap: map[string]*hooksCfgStorage{}},
			storageContent: map[string]storageTestContent{},
		},
		{
			name: "should create hooks",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{RetryCount: 1},
										{DefaultWaitTime: "1s"},
									},
								},
							},
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{RetryCount: 15},
									},
								},
							},
							DELETE: &config.DeleteActionConfig{
								Config: &config.DeleteActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{MaxWaitTime: "13s"},
									},
								},
							},
						},
					},
					"tgt2": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{{
										RetryCount: 160,
									}},
								},
							},
						},
					},
				},
			},
			fields: fields{
				storageMap: map[string]*hooksCfgStorage{},
			},
			storageContent: map[string]storageTestContent{
				"tgt1": {
					getLen:    2,
					putLen:    1,
					deleteLen: 1,
					getHookStorageCfg: []*config.WebhookConfig{
						{RetryCount: 1},
						{DefaultWaitTime: "1s"},
					},
					putHookStorageCfg: []*config.WebhookConfig{
						{RetryCount: 15},
					},
					deleteHookStorageCfg: []*config.WebhookConfig{
						{MaxWaitTime: "13s"},
					},
				},
				"tgt2": {
					getLen: 1,
					getHookStorageCfg: []*config.WebhookConfig{
						{RetryCount: 160},
					},
				},
			},
		},
		{
			name: "should update target hooks",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{RetryCount: 1},
										{DefaultWaitTime: "1s"},
									},
								},
							},
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{RetryCount: 15},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				storageMap: map[string]*hooksCfgStorage{
					"tgt1": {
						Delete: []*hookStorage{
							{Client: nil, Config: &config.WebhookConfig{MaxWaitTime: "13s"}},
						},
						Get: []*hookStorage{
							{Client: nil, Config: &config.WebhookConfig{MaxWaitTime: "130s"}},
						},
					},
				},
			},
			storageContent: map[string]storageTestContent{
				"tgt1": {
					getLen:    2,
					putLen:    1,
					deleteLen: 0,
					getHookStorageCfg: []*config.WebhookConfig{
						{RetryCount: 1},
						{DefaultWaitTime: "1s"},
					},
					putHookStorageCfg: []*config.WebhookConfig{
						{RetryCount: 15},
					},
					deleteHookStorageCfg: []*config.WebhookConfig{},
				},
			},
		},
		{
			name: "should flush target hooks",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{RetryCount: 1},
										{DefaultWaitTime: "1s"},
									},
								},
							},
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{RetryCount: 15},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				storageMap: map[string]*hooksCfgStorage{
					"tgt2": {
						Delete: []*hookStorage{
							{Client: nil, Config: &config.WebhookConfig{MaxWaitTime: "13s"}},
						},
						Get: []*hookStorage{
							{Client: nil, Config: &config.WebhookConfig{MaxWaitTime: "130s"}},
						},
					},
				},
			},
			storageContent: map[string]storageTestContent{
				"tgt1": {
					getLen:    2,
					putLen:    1,
					deleteLen: 0,
					getHookStorageCfg: []*config.WebhookConfig{
						{RetryCount: 1},
						{DefaultWaitTime: "1s"},
					},
					putHookStorageCfg: []*config.WebhookConfig{
						{RetryCount: 15},
					},
					deleteHookStorageCfg: []*config.WebhookConfig{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			cfgManagerMock := cmocks.NewMockManager(ctrl)

			cfgManagerMock.EXPECT().GetConfig().Return(tt.cfg)

			m := &manager{
				cfgManager: cfgManagerMock,
				storageMap: tt.fields.storageMap,
			}
			if err := m.Load(); (err != nil) != tt.wantErr {
				t.Errorf("manager.Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			for k, v := range tt.fields.storageMap {
				assert.Len(t, v.Get, tt.storageContent[k].getLen)
				assert.Len(t, v.Put, tt.storageContent[k].putLen)
				assert.Len(t, v.Delete, tt.storageContent[k].deleteLen)

				for i, v2 := range v.Get {
					assert.NotNil(t, v2.Client)
					assert.Equal(t, tt.storageContent[k].getHookStorageCfg[i], v2.Config)
				}
				for i, v2 := range v.Put {
					assert.NotNil(t, v2.Client)
					assert.Equal(t, tt.storageContent[k].putHookStorageCfg[i], v2.Config)
				}
				for i, v2 := range v.Delete {
					assert.NotNil(t, v2.Client)
					assert.Equal(t, tt.storageContent[k].deleteHookStorageCfg[i], v2.Config)
				}
			}

			storageKeys := make([]string, 0)
			for k := range tt.fields.storageMap {
				storageKeys = append(storageKeys, k)
			}
			sort.Strings(storageKeys)

			expectedStorageKeys := make([]string, 0)
			for k := range tt.storageContent {
				expectedStorageKeys = append(expectedStorageKeys, k)
			}
			sort.Strings(expectedStorageKeys)

			assert.Equal(t, expectedStorageKeys, storageKeys)
		})
	}
}

func Test_manager_manageDELETEHooksInternal(t *testing.T) {
	type responseMock struct {
		statusCode int
		body       string
	}
	type requestResult struct {
		method  string
		body    string
		headers map[string]string
	}
	type args struct {
		targetKey   string
		requestPath string
		s3Metadata  *S3Metadata
	}
	type metricsIncXWebhooksMockResult struct {
		input1 string
		input2 string
		times  int
	}
	tests := []struct {
		name                                string
		args                                args
		cfg                                 *config.Config
		injectMockServers                   bool
		metricsIncFailedWebhooksMockResult  metricsIncXWebhooksMockResult
		metricsIncSucceedWebhooksMockResult metricsIncXWebhooksMockResult
		responseMockList                    []responseMock
		requestResult                       []requestResult
	}{
		{
			name: "no storage for target",
			cfg:  &config.Config{},
			args: args{targetKey: "tgt1"},
		},
		{
			name: "empty storage for target",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {},
				},
			},
			args: args{targetKey: "tgt1"},
		},
		{
			name: "should fail to call url",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							DELETE: &config.DeleteActionConfig{
								Config: &config.DeleteActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											URL:    "http://not-an-url",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
			},
			injectMockServers: false,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "DELETE",
				times:  1,
			},
		},
		{
			name: "should fail when bad request is present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							DELETE: &config.DeleteActionConfig{
								Config: &config.DeleteActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "DELETE",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 400,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
		{
			"action":"DELETE",
			"requestPath":"/fake",
			"outputMetadata":{
				"bucket":"bucket",
				"region":"region",
				"s3Endpoint":"s3endpoint",
				"key":"key"
			},
			"target": {"name":"tgt1"}
		}
							`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should fail when internal server is present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							DELETE: &config.DeleteActionConfig{
								Config: &config.DeleteActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "DELETE",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 500,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
		{
			"action":"DELETE",
			"requestPath":"/fake",
			"outputMetadata":{
				"bucket":"bucket",
				"region":"region",
				"s3Endpoint":"s3endpoint",
				"key":"key"
			},
			"target": {"name":"tgt1"}
		}
							`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should fail when internal server and a method not allowed error are present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							DELETE: &config.DeleteActionConfig{
								Config: &config.DeleteActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "DELETE",
				times:  2,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 500,
				}, {
					body:       `{"error":"method not allowed"}`,
					statusCode: 405,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"DELETE",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"DELETE",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should be ok with 2 success",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							DELETE: &config.DeleteActionConfig{
								Config: &config.DeleteActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
											Headers: map[string]string{
												"h1": "v1",
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
			},
			injectMockServers: true,
			metricsIncSucceedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "DELETE",
				times:  2,
			},
			responseMockList: []responseMock{
				{
					body:       `{}`,
					statusCode: 200,
				}, {
					body:       `{}`,
					statusCode: 201,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"DELETE",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
						"h1":            "v1",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"DELETE",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should be ok with 1 success and 1 fail",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							DELETE: &config.DeleteActionConfig{
								Config: &config.DeleteActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
			},
			injectMockServers: true,
			metricsIncSucceedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "DELETE",
				times:  1,
			},
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "DELETE",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":"forbidden"}`,
					statusCode: 403,
				}, {
					body:       `{}`,
					statusCode: 201,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"DELETE",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"DELETE",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			cfgManagerMock := cmocks.NewMockManager(ctrl)
			metricsSvcMock := mmocks.NewMockClient(ctrl)

			cfgManagerMock.EXPECT().GetConfig().Return(tt.cfg)

			metricsSvcMock.EXPECT().IncFailedWebhooks(
				tt.metricsIncFailedWebhooksMockResult.input1,
				tt.metricsIncFailedWebhooksMockResult.input2,
			).Times(
				tt.metricsIncFailedWebhooksMockResult.times,
			)

			metricsSvcMock.EXPECT().IncSucceedWebhooks(
				tt.metricsIncSucceedWebhooksMockResult.input1,
				tt.metricsIncSucceedWebhooksMockResult.input2,
			).Times(
				tt.metricsIncSucceedWebhooksMockResult.times,
			)

			m := &manager{
				cfgManager: cfgManagerMock,
				metricsSvc: metricsSvcMock,
				storageMap: map[string]*hooksCfgStorage{},
			}

			// Create ctx
			ctx := context.TODO()
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())
			ctx = opentracing.ContextWithSpan(ctx, opentracing.StartSpan("fake"))

			// Save request
			reqs := make([]*struct {
				Body    string
				Method  string
				Headers http.Header
			}, 0)

			if tt.injectMockServers {
				// Create mock servers
				for _, v := range tt.cfg.Targets {
					for i, v2 := range v.Actions.DELETE.Config.Webhooks {
						// Get mock
						m := tt.responseMockList[i]

						s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
							by, err := io.ReadAll(r.Body)
							assert.NoError(t, err)

							reqs = append(reqs, &struct {
								Body    string
								Method  string
								Headers http.Header
							}{
								Method:  r.Method,
								Headers: r.Header,
								Body:    string(by),
							})

							rw.WriteHeader(m.statusCode)
							rw.Write([]byte(m.body))
						}))

						defer s.Close()

						v2.URL = s.URL
					}
				}
			}

			// Load clients
			err := m.Load()
			assert.NoError(t, err)

			m.manageDELETEHooksInternal(ctx, tt.args.targetKey, tt.args.requestPath, tt.args.s3Metadata)

			// Test
			for i, v := range tt.requestResult {
				assert.JSONEq(t, v.body, reqs[i].Body)

				assert.Equal(t, v.method, reqs[i].Method)

				for key, val := range v.headers {
					assert.Equal(t, val, reqs[i].Headers.Get(key))
				}
			}
		})
	}
}

func Test_manager_managePUTHooksInternal(t *testing.T) {
	type responseMock struct {
		statusCode int
		body       string
	}
	type requestResult struct {
		method  string
		body    string
		headers map[string]string
	}
	type args struct {
		targetKey   string
		requestPath string
		metadata    *PutInputMetadata
		s3Metadata  *S3Metadata
	}
	type metricsIncXWebhooksMockResult struct {
		input1 string
		input2 string
		times  int
	}
	tests := []struct {
		name                                string
		args                                args
		cfg                                 *config.Config
		injectMockServers                   bool
		metricsIncFailedWebhooksMockResult  metricsIncXWebhooksMockResult
		metricsIncSucceedWebhooksMockResult metricsIncXWebhooksMockResult
		responseMockList                    []responseMock
		requestResult                       []requestResult
	}{
		{
			name: "no storage for target",
			cfg:  &config.Config{},
			args: args{targetKey: "tgt1"},
		},
		{
			name: "empty storage for target",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {},
				},
			},
			args: args{targetKey: "tgt1"},
		},
		{
			name: "should fail to call url",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											URL:    "http://not-an-url",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &PutInputMetadata{
					Filename:    "filename",
					ContentType: "contenttype",
					ContentSize: 1,
				},
			},
			injectMockServers: false,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "PUT",
				times:  1,
			},
		},
		{
			name: "should fail when bad request is present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &PutInputMetadata{
					Filename:    "filename",
					ContentType: "contenttype",
					ContentSize: 1,
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "PUT",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 400,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
		{
			"action":"PUT",
			"requestPath":"/fake",
			"outputMetadata":{
				"bucket":"bucket",
				"region":"region",
				"s3Endpoint":"s3endpoint",
				"key":"key"
			},
			"inputMetadata":{
				"contentSize":1,
				"contentType":"contenttype",
				"filename":"filename"
			},
			"target": {"name":"tgt1"}
		}
							`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should fail when internal server is present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &PutInputMetadata{
					Filename:    "filename",
					ContentType: "contenttype",
					ContentSize: 1,
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "PUT",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 500,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
		{
			"action":"PUT",
			"requestPath":"/fake",
			"outputMetadata":{
				"bucket":"bucket",
				"region":"region",
				"s3Endpoint":"s3endpoint",
				"key":"key"
			},
			"inputMetadata":{
				"contentSize":1,
				"contentType":"contenttype",
				"filename":"filename"
			},
			"target": {"name":"tgt1"}
		}
							`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should fail when internal server and a method not allowed error are present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &PutInputMetadata{
					Filename:    "filename",
					ContentType: "contenttype",
					ContentSize: 1,
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "PUT",
				times:  2,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 500,
				}, {
					body:       `{"error":"method not allowed"}`,
					statusCode: 405,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"PUT",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"contentSize":1,
		"contentType":"contenttype",
		"filename":"filename"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"PUT",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"contentSize":1,
		"contentType":"contenttype",
		"filename":"filename"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should be ok with 2 success",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
											Headers: map[string]string{
												"h1": "v1",
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &PutInputMetadata{
					Filename:    "filename",
					ContentType: "contenttype",
					ContentSize: 1,
				},
			},
			injectMockServers: true,
			metricsIncSucceedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "PUT",
				times:  2,
			},
			responseMockList: []responseMock{
				{
					body:       `{}`,
					statusCode: 200,
				}, {
					body:       `{}`,
					statusCode: 201,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"PUT",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"contentSize":1,
		"contentType":"contenttype",
		"filename":"filename"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
						"h1":            "v1",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"PUT",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"contentSize":1,
		"contentType":"contenttype",
		"filename":"filename"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should be ok with 1 success and 1 fail",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							PUT: &config.PutActionConfig{
								Config: &config.PutActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &PutInputMetadata{
					Filename:    "filename",
					ContentType: "contenttype",
					ContentSize: 1,
				},
			},
			injectMockServers: true,
			metricsIncSucceedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "PUT",
				times:  1,
			},
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "PUT",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":"forbidden"}`,
					statusCode: 403,
				}, {
					body:       `{}`,
					statusCode: 201,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"PUT",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"contentSize":1,
		"contentType":"contenttype",
		"filename":"filename"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"PUT",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"contentSize":1,
		"contentType":"contenttype",
		"filename":"filename"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			cfgManagerMock := cmocks.NewMockManager(ctrl)
			metricsSvcMock := mmocks.NewMockClient(ctrl)

			cfgManagerMock.EXPECT().GetConfig().Return(tt.cfg)

			metricsSvcMock.EXPECT().IncFailedWebhooks(
				tt.metricsIncFailedWebhooksMockResult.input1,
				tt.metricsIncFailedWebhooksMockResult.input2,
			).Times(
				tt.metricsIncFailedWebhooksMockResult.times,
			)

			metricsSvcMock.EXPECT().IncSucceedWebhooks(
				tt.metricsIncSucceedWebhooksMockResult.input1,
				tt.metricsIncSucceedWebhooksMockResult.input2,
			).Times(
				tt.metricsIncSucceedWebhooksMockResult.times,
			)

			m := &manager{
				cfgManager: cfgManagerMock,
				metricsSvc: metricsSvcMock,
				storageMap: map[string]*hooksCfgStorage{},
			}

			// Create ctx
			ctx := context.TODO()
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())
			ctx = opentracing.ContextWithSpan(ctx, opentracing.StartSpan("fake"))

			// Save request
			reqs := make([]*struct {
				Body    string
				Method  string
				Headers http.Header
			}, 0)

			if tt.injectMockServers {
				// Create mock servers
				for _, v := range tt.cfg.Targets {
					for i, v2 := range v.Actions.PUT.Config.Webhooks {
						// Get mock
						m := tt.responseMockList[i]

						s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
							by, err := io.ReadAll(r.Body)
							assert.NoError(t, err)

							reqs = append(reqs, &struct {
								Body    string
								Method  string
								Headers http.Header
							}{
								Method:  r.Method,
								Headers: r.Header,
								Body:    string(by),
							})

							rw.WriteHeader(m.statusCode)
							rw.Write([]byte(m.body))
						}))

						defer s.Close()

						v2.URL = s.URL
					}
				}
			}

			// Load clients
			err := m.Load()
			assert.NoError(t, err)

			m.managePUTHooksInternal(
				ctx,
				tt.args.targetKey,
				tt.args.requestPath,
				tt.args.metadata,
				tt.args.s3Metadata,
			)

			// Test
			for i, v := range tt.requestResult {
				assert.JSONEq(t, v.body, reqs[i].Body)

				assert.Equal(t, v.method, reqs[i].Method)

				for key, val := range v.headers {
					assert.Equal(t, val, reqs[i].Headers.Get(key))
				}
			}
		})
	}
}

func Test_manager_manageGETHooksInternal(t *testing.T) {
	type responseMock struct {
		statusCode int
		body       string
	}
	type requestResult struct {
		method  string
		body    string
		headers map[string]string
	}
	type args struct {
		targetKey   string
		requestPath string
		metadata    *GetInputMetadata
		s3Metadata  *S3Metadata
	}
	type metricsIncXWebhooksMockResult struct {
		input1 string
		input2 string
		times  int
	}
	tests := []struct {
		name                                string
		args                                args
		cfg                                 *config.Config
		injectMockServers                   bool
		metricsIncFailedWebhooksMockResult  metricsIncXWebhooksMockResult
		metricsIncSucceedWebhooksMockResult metricsIncXWebhooksMockResult
		responseMockList                    []responseMock
		requestResult                       []requestResult
	}{
		{
			name: "no storage for target",
			cfg:  &config.Config{},
			args: args{targetKey: "tgt1"},
		},
		{
			name: "empty storage for target",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {},
				},
			},
			args: args{targetKey: "tgt1"},
		},
		{
			name: "should fail to call url",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											URL:    "http://not-an-url",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &GetInputMetadata{
					Range:       "range",
					IfMatch:     "ifmatch",
					IfNoneMatch: "ifnonematch",
				},
			},
			injectMockServers: false,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "GET",
				times:  1,
			},
		},
		{
			name: "should fail when bad request is present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &GetInputMetadata{
					Range:       "range",
					IfMatch:     "ifmatch",
					IfNoneMatch: "ifnonematch",
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "GET",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 400,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
		{
			"action":"GET",
			"requestPath":"/fake",
			"outputMetadata":{
				"bucket":"bucket",
				"region":"region",
				"s3Endpoint":"s3endpoint",
				"key":"key"
			},
			"inputMetadata":{
				"ifMatch":"ifmatch",
				"ifModifiedSince":"",
				"ifNoneMatch":"ifnonematch",
				"ifUnmodifiedSince":"",
				"range":"range"
			},
			"target": {"name":"tgt1"}
		}
							`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should fail when internal server is present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &GetInputMetadata{
					Range:       "range",
					IfMatch:     "ifmatch",
					IfNoneMatch: "ifnonematch",
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "GET",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 500,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
		{
			"action":"GET",
			"requestPath":"/fake",
			"outputMetadata":{
				"bucket":"bucket",
				"region":"region",
				"s3Endpoint":"s3endpoint",
				"key":"key"
			},
			"inputMetadata":{
				"ifMatch":"ifmatch",
				"ifModifiedSince":"",
				"ifNoneMatch":"ifnonematch",
				"ifUnmodifiedSince":"",
				"range":"range"
			},
			"target": {"name":"tgt1"}
		}
							`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should fail when internal server and a method not allowed error are present",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &GetInputMetadata{
					Range:       "range",
					IfMatch:     "ifmatch",
					IfNoneMatch: "ifnonematch",
				},
			},
			injectMockServers: true,
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "GET",
				times:  2,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":true}`,
					statusCode: 500,
				}, {
					body:       `{"error":"method not allowed"}`,
					statusCode: 405,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"GET",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"ifMatch":"ifmatch",
		"ifModifiedSince":"",
		"ifNoneMatch":"ifnonematch",
		"ifUnmodifiedSince":"",
		"range":"range"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"GET",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"ifMatch":"ifmatch",
		"ifModifiedSince":"",
		"ifNoneMatch":"ifnonematch",
		"ifUnmodifiedSince":"",
		"range":"range"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should be ok with 2 success",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
											Headers: map[string]string{
												"h1": "v1",
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &GetInputMetadata{
					Range:       "range",
					IfMatch:     "ifmatch",
					IfNoneMatch: "ifnonematch",
				},
			},
			injectMockServers: true,
			metricsIncSucceedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "GET",
				times:  2,
			},
			responseMockList: []responseMock{
				{
					body:       `{}`,
					statusCode: 200,
				}, {
					body:       `{}`,
					statusCode: 201,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"GET",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"ifMatch":"ifmatch",
		"ifModifiedSince":"",
		"ifNoneMatch":"ifnonematch",
		"ifUnmodifiedSince":"",
		"range":"range"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
						"h1":            "v1",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"GET",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"ifMatch":"ifmatch",
		"ifModifiedSince":"",
		"ifNoneMatch":"ifnonematch",
		"ifUnmodifiedSince":"",
		"range":"range"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
		{
			name: "should be ok with 1 success and 1 fail",
			cfg: &config.Config{
				Targets: map[string]*config.TargetConfig{
					"tgt1": {
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{
								Config: &config.GetActionConfigConfig{
									Webhooks: []*config.WebhookConfig{
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret",
												},
											},
										},
										{
											Method: "POST",
											SecretHeaders: map[string]*config.CredentialConfig{
												"authorization": {
													Value: "secret2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				targetKey:   "tgt1",
				requestPath: "/fake",
				s3Metadata: &S3Metadata{
					Bucket:     "bucket",
					Region:     "region",
					S3Endpoint: "s3endpoint",
					Key:        "key",
				},
				metadata: &GetInputMetadata{
					Range:       "range",
					IfMatch:     "ifmatch",
					IfNoneMatch: "ifnonematch",
				},
			},
			injectMockServers: true,
			metricsIncSucceedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "GET",
				times:  1,
			},
			metricsIncFailedWebhooksMockResult: metricsIncXWebhooksMockResult{
				input1: "tgt1",
				input2: "GET",
				times:  1,
			},
			responseMockList: []responseMock{
				{
					body:       `{"error":"forbidden"}`,
					statusCode: 403,
				}, {
					body:       `{}`,
					statusCode: 201,
				},
			},
			requestResult: []requestResult{
				{
					method: "POST",
					body: `
{
	"action":"GET",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"ifMatch":"ifmatch",
		"ifModifiedSince":"",
		"ifNoneMatch":"ifnonematch",
		"ifUnmodifiedSince":"",
		"range":"range"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret",
						"Content-Type":  "application/json",
					},
				}, {
					method: "POST",
					body: `
{
	"action":"GET",
	"requestPath":"/fake",
	"outputMetadata":{
		"bucket":"bucket",
		"region":"region",
		"s3Endpoint":"s3endpoint",
		"key":"key"
	},
	"inputMetadata":{
		"ifMatch":"ifmatch",
		"ifModifiedSince":"",
		"ifNoneMatch":"ifnonematch",
		"ifUnmodifiedSince":"",
		"range":"range"
	},
	"target": {"name":"tgt1"}
}
					`,
					headers: map[string]string{
						"Authorization": "secret2",
						"Content-Type":  "application/json",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			cfgManagerMock := cmocks.NewMockManager(ctrl)
			metricsSvcMock := mmocks.NewMockClient(ctrl)

			cfgManagerMock.EXPECT().GetConfig().Return(tt.cfg)

			metricsSvcMock.EXPECT().IncFailedWebhooks(
				tt.metricsIncFailedWebhooksMockResult.input1,
				tt.metricsIncFailedWebhooksMockResult.input2,
			).Times(
				tt.metricsIncFailedWebhooksMockResult.times,
			)

			metricsSvcMock.EXPECT().IncSucceedWebhooks(
				tt.metricsIncSucceedWebhooksMockResult.input1,
				tt.metricsIncSucceedWebhooksMockResult.input2,
			).Times(
				tt.metricsIncSucceedWebhooksMockResult.times,
			)

			m := &manager{
				cfgManager: cfgManagerMock,
				metricsSvc: metricsSvcMock,
				storageMap: map[string]*hooksCfgStorage{},
			}

			// Create ctx
			ctx := context.TODO()
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())
			ctx = opentracing.ContextWithSpan(ctx, opentracing.StartSpan("fake"))

			// Save request
			reqs := make([]*struct {
				Body    string
				Method  string
				Headers http.Header
			}, 0)

			if tt.injectMockServers {
				// Create mock servers
				for _, v := range tt.cfg.Targets {
					for i, v2 := range v.Actions.GET.Config.Webhooks {
						// Get mock
						m := tt.responseMockList[i]

						s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
							by, err := io.ReadAll(r.Body)
							assert.NoError(t, err)

							reqs = append(reqs, &struct {
								Body    string
								Method  string
								Headers http.Header
							}{
								Method:  r.Method,
								Headers: r.Header,
								Body:    string(by),
							})

							rw.WriteHeader(m.statusCode)
							rw.Write([]byte(m.body))
						}))

						defer s.Close()

						v2.URL = s.URL
					}
				}
			}

			// Load clients
			err := m.Load()
			assert.NoError(t, err)

			m.manageGETHooksInternal(
				ctx,
				tt.args.targetKey,
				tt.args.requestPath,
				tt.args.metadata,
				tt.args.s3Metadata,
			)

			// Test
			for i, v := range tt.requestResult {
				assert.JSONEq(t, v.body, reqs[i].Body)

				assert.Equal(t, v.method, reqs[i].Method)

				for key, val := range v.headers {
					assert.Equal(t, val, reqs[i].Headers.Get(key))
				}
			}
		})
	}
}
