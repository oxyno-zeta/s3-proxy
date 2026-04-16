//go:build unit

package s3client

import (
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	"go.uber.org/mock/gomock"
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
	s3Manager := NewManager(cfgManagerMock, nil).(*manager)

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

func Test_newClient_S3ForcePathStyleDefaultAndOverrides(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name         string
		forcePath    *bool
		expectedPath bool
	}{
		{
			name:         "uses default when omitted",
			forcePath:    nil,
			expectedPath: config.DefaultBucketS3ForcePathStyle,
		},
		{
			name:         "keeps explicit true",
			forcePath:    &trueValue,
			expectedPath: true,
		},
		{
			name:         "keeps explicit false",
			forcePath:    &falseValue,
			expectedPath: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl, err := newClient(&config.TargetConfig{
				Bucket: &config.BucketConfig{
					Name:                "bucket1",
					Region:              "us-east-1",
					S3MaxUploadParts:    config.DefaultS3MaxUploadParts,
					S3UploadPartSize:    config.DefaultS3UploadPartSize,
					S3UploadConcurrency: config.DefaultS3UploadConcurrency,
					S3ForcePathStyle:    tt.forcePath,
				},
			}, nil)
			if !assert.NoError(t, err) {
				return
			}

			s3cl, ok := cl.(*s3client)
			if !assert.True(t, ok) {
				return
			}
			svc, ok := s3cl.svcClient.(*s3.S3)
			if !assert.True(t, ok) {
				return
			}

			assert.NotNil(t, svc.Config.S3ForcePathStyle)
			assert.Equal(t, tt.expectedPath, *svc.Config.S3ForcePathStyle)
		})
	}
}
