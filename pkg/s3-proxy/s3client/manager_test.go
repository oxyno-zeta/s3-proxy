//go:build unit

package s3client

import (
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
)

func Test_manager_Load_Cleanup(t *testing.T) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())

	// Create go mock controller
	ctrl := gomock.NewController(t)
	cfgManagerMock := cmocks.NewMockManager(ctrl)

	cfgManagerMock.EXPECT().GetConfig().Times(1).Return(&config.Config{
		Targets: map[string]*config.TargetConfig{
			"t1": {
				Bucket: &config.BucketConfig{
					S3Endpoint: ts.URL,
					Credentials: &config.BucketCredentialConfig{
						AccessKey: &config.CredentialConfig{Value: "access"},
						SecretKey: &config.CredentialConfig{Value: "secret"},
					},
				},
			},
			"t2": {
				Bucket: &config.BucketConfig{
					S3Endpoint: ts.URL,
					Credentials: &config.BucketCredentialConfig{
						AccessKey: &config.CredentialConfig{Value: "access"},
						SecretKey: &config.CredentialConfig{Value: "secret"},
					},
				},
			},
		},
	})

	// create manager
	s3Manager := NewManager(cfgManagerMock, nil, nil).(*manager)

	// Load
	err := s3Manager.Load()

	if !assert.NoError(t, err) {
		return
	}

	assert.Contains(t, funk.Keys(s3Manager.targetClient), "t1")
	assert.Contains(t, funk.Keys(s3Manager.targetClient), "t2")
	assert.Len(t, funk.Keys(s3Manager.targetClient), 2)

	cfgManagerMock.EXPECT().GetConfig().Times(1).Return(&config.Config{
		Targets: map[string]*config.TargetConfig{
			"t1": {
				Bucket: &config.BucketConfig{
					S3Endpoint: ts.URL,
					Credentials: &config.BucketCredentialConfig{
						AccessKey: &config.CredentialConfig{Value: "access"},
						SecretKey: &config.CredentialConfig{Value: "secret"},
					},
				},
			},
		},
	})

	// Load
	err = s3Manager.Load()

	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []string{"t1"}, funk.Keys(s3Manager.targetClient))
}
