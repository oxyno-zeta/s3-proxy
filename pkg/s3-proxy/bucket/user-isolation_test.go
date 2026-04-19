//go:build unit

package bucket

import (
	"context"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	responsehandlermocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/mocks"
	responsehandlermodels "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	s3clientmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook"
	wmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_isUserIsolationEnabled(t *testing.T) {
	tests := []struct {
		name      string
		targetCfg *config.TargetConfig
		want      bool
	}{
		{
			name:      "should return false when actions is nil",
			targetCfg: &config.TargetConfig{},
			want:      false,
		},
		{
			name: "should return false when GET action is nil",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{},
			},
			want: false,
		},
		{
			name: "should return false when GET config is nil",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{},
				},
			},
			want: false,
		},
		{
			name: "should return false when userIsolation is not set",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{},
					},
				},
			},
			want: false,
		},
		{
			name: "should return true when userIsolation is enabled",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bri := &bucketReqImpl{
				targetCfg: tt.targetCfg,
			}
			got := bri.isUserIsolationEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_isUserIsolationAdmin(t *testing.T) {
	tests := []struct {
		name      string
		targetCfg *config.TargetConfig
		username  string
		want      bool
	}{
		{
			name:      "should return false when actions is nil",
			targetCfg: &config.TargetConfig{},
			username:  "admin",
			want:      false,
		},
		{
			name: "should return false when admin list is empty",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{},
						},
					},
				},
			},
			username: "admin",
			want:     false,
		},
		{
			name: "should return false when user is not in admin list",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin1", "admin2"},
						},
					},
				},
			},
			username: "regularuser",
			want:     false,
		},
		{
			name: "should return true for first admin in list",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin1", "admin2"},
						},
					},
				},
			},
			username: "admin1",
			want:     true,
		},
		{
			name: "should return true for second admin in list",
			targetCfg: &config.TargetConfig{
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin1", "admin2"},
						},
					},
				},
			},
			username: "admin2",
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bri := &bucketReqImpl{
				targetCfg: tt.targetCfg,
			}
			got := bri.isUserIsolationAdmin(tt.username)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_generateStartKey_UserIsolation verifies that S3 keys are built correctly
// under the transparent-injection model: the authenticated username is inserted
// after the bucket root prefix for non-admin users; admins get the bare key.
func Test_generateStartKey_UserIsolation(t *testing.T) {
	tests := []struct {
		name        string
		targetCfg   *config.TargetConfig
		user        models.GenericUser
		requestPath string
		wantKey     string
		wantErr     bool
	}{
		{
			name: "should not inject when isolation disabled",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{Config: &config.GetActionConfigConfig{}},
				},
			},
			user:        &models.BasicAuthUser{Username: "alice"},
			requestPath: "/file.txt",
			wantKey:     "data/file.txt",
		},
		{
			name: "should inject username for non-admin alice",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user:        &models.BasicAuthUser{Username: "alice"},
			requestPath: "/file.txt",
			wantKey:     "data/alice/file.txt",
		},
		{
			name: "should inject username for non-admin bob",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user:        &models.BasicAuthUser{Username: "bob"},
			requestPath: "/sub/deep.txt",
			wantKey:     "data/bob/sub/deep.txt",
		},
		{
			name: "should inject username for non-admin charlie",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user:        &models.BasicAuthUser{Username: "charlie"},
			requestPath: "/",
			wantKey:     "data/charlie/",
		},
		{
			name: "should not inject username for admin user",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin"},
						},
					},
				},
			},
			user:        &models.BasicAuthUser{Username: "admin"},
			requestPath: "/bob/file.txt",
			wantKey:     "data/bob/file.txt",
		},
		{
			name: "should not inject for a secondary admin (superuser)",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin", "superuser"},
						},
					},
				},
			},
			user:        &models.BasicAuthUser{Username: "superuser"},
			requestPath: "/alice/secret.txt",
			wantKey:     "data/alice/secret.txt",
		},
		{
			name: "should work with empty bucket prefix",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: ""},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user:        &models.BasicAuthUser{Username: "alice"},
			requestPath: "/file.txt",
			wantKey:     "alice/file.txt",
		},
		{
			name: "should error when isolation enabled but no user in context",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user:        nil,
			requestPath: "/file.txt",
			wantErr:     true,
		},
		{
			name: "should ignore path-traversal-like inputs by still confining under username",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user:        &models.BasicAuthUser{Username: "alice"},
			requestPath: "/bob/../file.txt",
			wantKey:     "data/alice/bob/../file.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			bri := &bucketReqImpl{targetCfg: tt.targetCfg}
			got, err := bri.generateStartKey(ctx, tt.requestPath)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantKey, got)
		})
	}
}

// Test_displayPrefix_UserIsolation verifies that the prefix used to build
// user-facing paths hides the injected username for non-admin users while
// admins see only the bucket prefix stripped.
func Test_displayPrefix_UserIsolation(t *testing.T) {
	tests := []struct {
		name      string
		targetCfg *config.TargetConfig
		user      models.GenericUser
		want      string
	}{
		{
			name: "should return bucket prefix when isolation disabled",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{Config: &config.GetActionConfigConfig{}},
				},
			},
			user: &models.BasicAuthUser{Username: "alice"},
			want: "data/",
		},
		{
			name: "should include username for non-admin alice",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user: &models.BasicAuthUser{Username: "alice"},
			want: "data/alice/",
		},
		{
			name: "should include username for non-admin bob",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user: &models.BasicAuthUser{Username: "bob"},
			want: "data/bob/",
		},
		{
			name: "should not include username for admin",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin"},
						},
					},
				},
			},
			user: &models.BasicAuthUser{Username: "admin"},
			want: "data/",
		},
		{
			name: "should return bucket prefix when user missing and isolation enabled",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{UserIsolation: true},
					},
				},
			},
			user: nil,
			want: "data/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			bri := &bucketReqImpl{targetCfg: tt.targetCfg}
			got := bri.displayPrefix(ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_requestContext_Delete_UserIsolation verifies DELETE keys are rewritten
// transparently so users can only delete files under their own folder, and
// admins can delete anything.
func Test_requestContext_Delete_UserIsolation(t *testing.T) {
	type responseHandlerDeleteMockResult struct {
		input *responsehandlermodels.DeleteInput
		times int
	}
	type responseHandlerErrorsMockResult struct {
		input2 error
		times  int
	}
	type s3ClientDeleteObjectMockResult struct {
		input2 string
		res    *s3client.ResultInfo
		err    error
		times  int
	}
	type webhookManagerManageDeleteHooksMockResult struct {
		input2 string
		input3 string
		input4 *webhook.S3Metadata
		times  int
	}
	type fields struct {
		targetCfg *config.TargetConfig
		mountPath string
	}
	type args struct {
		requestPath string
	}
	tests := []struct {
		name                                         string
		fields                                       fields
		args                                         args
		user                                         models.GenericUser
		s3clManagerClientForTargetMockInput          string
		responseHandlerDeleteMockResultTimes         responseHandlerDeleteMockResult
		responseHandlerInternalServerErrorMockResult responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult      responseHandlerErrorsMockResult
		s3ClientDeleteObjectMockResult               s3ClientDeleteObjectMockResult
		webhookManagerManageDeleteHooksMockResult    webhookManagerManageDeleteHooksMockResult
	}{
		{
			name: "alice should delete only under her own folder via injected key",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                                args{requestPath: "/file.txt"},
			user:                                &models.BasicAuthUser{Username: "alice"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "data/alice/file.txt",
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageDeleteHooksMockResult: webhookManagerManageDeleteHooksMockResult{
				input2: "bucket",
				input3: "/file.txt",
				input4: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerDeleteMockResultTimes: responseHandlerDeleteMockResult{
				input: &responsehandlermodels.DeleteInput{Key: "data/alice/file.txt"},
				times: 1,
			},
		},
		{
			name: "bob requesting /charlie/file.txt stays confined to bob's folder",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                                args{requestPath: "/charlie/file.txt"},
			user:                                &models.BasicAuthUser{Username: "bob"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "data/bob/charlie/file.txt",
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageDeleteHooksMockResult: webhookManagerManageDeleteHooksMockResult{
				input2: "bucket",
				input3: "/charlie/file.txt",
				input4: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerDeleteMockResultTimes: responseHandlerDeleteMockResult{
				input: &responsehandlermodels.DeleteInput{Key: "data/bob/charlie/file.txt"},
				times: 1,
			},
		},
		{
			name: "admin should delete in bob's folder using the bob-prefixed path",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{
								UserIsolation:       true,
								UserIsolationAdmins: []string{"admin"},
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                                args{requestPath: "/bob/file.txt"},
			user:                                &models.BasicAuthUser{Username: "admin"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "data/bob/file.txt",
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageDeleteHooksMockResult: webhookManagerManageDeleteHooksMockResult{
				input2: "bucket",
				input3: "/bob/file.txt",
				input4: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerDeleteMockResultTimes: responseHandlerDeleteMockResult{
				input: &responsehandlermodels.DeleteInput{Key: "data/bob/file.txt"},
				times: 1,
			},
		},
		{
			name: "should forbid delete when isolation is on and no user is set",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{requestPath: "/file.txt"},
			user: nil,
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errUserIsolationForbidden,
				times:  1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)

			ctx := context.TODO()
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			resHandlerMock.EXPECT().
				InternalServerError(gomock.Any(), tt.responseHandlerInternalServerErrorMockResult.input2).
				Times(tt.responseHandlerInternalServerErrorMockResult.times)
			resHandlerMock.EXPECT().
				ForbiddenError(gomock.Any(), tt.responseHandlerForbiddenErrorMockResult.input2).
				Times(tt.responseHandlerForbiddenErrorMockResult.times)
			resHandlerMock.EXPECT().Delete(
				gomock.Any(),
				tt.responseHandlerDeleteMockResultTimes.input,
			).Times(tt.responseHandlerDeleteMockResultTimes.times)

			s3ClientMock.EXPECT().
				DeleteObject(ctx, tt.s3ClientDeleteObjectMockResult.input2).
				Return(tt.s3ClientDeleteObjectMockResult.res, tt.s3ClientDeleteObjectMockResult.err).
				Times(tt.s3ClientDeleteObjectMockResult.times)

			s3clManagerMock.EXPECT().
				GetClientForTarget(tt.s3clManagerClientForTargetMockInput).
				AnyTimes().
				Return(s3ClientMock)

			webhookManagerMock.EXPECT().
				ManageDELETEHooks(
					ctx,
					tt.webhookManagerManageDeleteHooksMockResult.input2,
					tt.webhookManagerManageDeleteHooksMockResult.input3,
					tt.webhookManagerManageDeleteHooksMockResult.input4,
				).
				Times(tt.webhookManagerManageDeleteHooksMockResult.times)

			rctx := &bucketReqImpl{
				s3ClientManager: s3clManagerMock,
				webhookManager:  webhookManagerMock,
				targetCfg:       tt.fields.targetCfg,
				mountPath:       tt.fields.mountPath,
			}
			rctx.Delete(ctx, tt.args.requestPath)
		})
	}
}

// Test_requestContext_Put_UserIsolation verifies PUT uploads go to the
// injected user folder transparently and that admins keep bucket-wide access.
func Test_requestContext_Put_UserIsolation(t *testing.T) {
	type responseHandlerPutMockResult struct {
		input *responsehandlermodels.PutInput
		times int
	}
	type responseHandlerErrorsMockResult struct {
		input2 error
		times  int
	}
	type s3ClientPutObjectMockResult struct {
		input2 *s3client.PutInput
		res    *s3client.ResultInfo
		err    error
		times  int
	}
	type webhookManagerManagePutHooksMockResult struct {
		input2 string
		input3 string
		input4 *webhook.PutInputMetadata
		input5 *webhook.S3Metadata
		times  int
	}
	type fields struct {
		targetCfg *config.TargetConfig
		mountPath string
	}
	type args struct {
		inp *PutInput
	}
	tests := []struct {
		name                                         string
		fields                                       fields
		args                                         args
		user                                         models.GenericUser
		responseHandlerInternalServerErrorMockResult responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult      responseHandlerErrorsMockResult
		responseHandlerPutMockResultTimes            responseHandlerPutMockResult
		s3clManagerClientForTargetMockInput          string
		s3ClientPutObjectMockResult                  s3ClientPutObjectMockResult
		webhookManagerManagePutHooksMockResult       webhookManagerManagePutHooksMockResult
	}{
		{
			name: "alice PUT at root should be stored under data/alice/",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
			},
			user:                                &models.BasicAuthUser{Username: "alice"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "data/alice/file.txt",
					ContentType: "text/plain",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "target",
				input3: "/",
				input4: &webhook.PutInputMetadata{
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerPutMockResultTimes: responseHandlerPutMockResult{
				input: &responsehandlermodels.PutInput{
					Key:         "data/alice/file.txt",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
				times: 1,
			},
		},
		{
			name: "bob PUT with sub-path /charlie should land under data/bob/charlie/",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/charlie",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
			},
			user:                                &models.BasicAuthUser{Username: "bob"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "data/bob/charlie/file.txt",
					ContentType: "text/plain",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "target",
				input3: "/charlie",
				input4: &webhook.PutInputMetadata{
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerPutMockResultTimes: responseHandlerPutMockResult{
				input: &responsehandlermodels.PutInput{
					Key:         "data/bob/charlie/file.txt",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
				times: 1,
			},
		},
		{
			name: "admin PUT under /alice should hit data/alice/ unchanged",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{
								UserIsolation:       true,
								UserIsolationAdmins: []string{"admin"},
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/alice",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
			},
			user:                                &models.BasicAuthUser{Username: "admin"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "data/alice/file.txt",
					ContentType: "text/plain",
				},
				res: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManagePutHooksMockResult: webhookManagerManagePutHooksMockResult{
				input2: "target",
				input3: "/alice",
				input4: &webhook.PutInputMetadata{
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerPutMockResultTimes: responseHandlerPutMockResult{
				input: &responsehandlermodels.PutInput{
					Key:         "data/alice/file.txt",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
				times: 1,
			},
		},
		{
			name: "should forbid PUT when isolation is on and no user is set",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
			},
			user: nil,
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errUserIsolationForbidden,
				times:  1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)

			ctx := context.TODO()
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			resHandlerMock.EXPECT().
				InternalServerError(gomock.Any(), tt.responseHandlerInternalServerErrorMockResult.input2).
				Times(tt.responseHandlerInternalServerErrorMockResult.times)
			resHandlerMock.EXPECT().
				ForbiddenError(gomock.Any(), tt.responseHandlerForbiddenErrorMockResult.input2).
				Times(tt.responseHandlerForbiddenErrorMockResult.times)
			resHandlerMock.EXPECT().Put(
				gomock.Any(),
				tt.responseHandlerPutMockResultTimes.input,
			).Times(tt.responseHandlerPutMockResultTimes.times)

			s3ClientMock.EXPECT().
				HeadObject(ctx, gomock.Any()).
				Return(nil, nil, nil).
				AnyTimes()
			s3ClientMock.EXPECT().
				PutObject(ctx, tt.s3ClientPutObjectMockResult.input2).
				Return(tt.s3ClientPutObjectMockResult.res, tt.s3ClientPutObjectMockResult.err).
				Times(tt.s3ClientPutObjectMockResult.times)

			s3clManagerMock.EXPECT().
				GetClientForTarget(tt.s3clManagerClientForTargetMockInput).
				AnyTimes().
				Return(s3ClientMock)

			webhookManagerMock.EXPECT().
				ManagePUTHooks(
					ctx,
					tt.webhookManagerManagePutHooksMockResult.input2,
					tt.webhookManagerManagePutHooksMockResult.input3,
					tt.webhookManagerManagePutHooksMockResult.input4,
					tt.webhookManagerManagePutHooksMockResult.input5,
				).
				Times(tt.webhookManagerManagePutHooksMockResult.times)

			rctx := &bucketReqImpl{
				s3ClientManager: s3clManagerMock,
				webhookManager:  webhookManagerMock,
				targetCfg:       tt.fields.targetCfg,
				mountPath:       tt.fields.mountPath,
			}
			rctx.Put(ctx, tt.args.inp)
		})
	}
}

// Test_requestContext_Get_UserIsolation covers GET listings and direct-file
// GETs under user isolation: non-admin users list their own folder with
// username-free Paths; admins keep bucket-wide access; missing user yields 403.
func Test_requestContext_Get_UserIsolation(t *testing.T) {
	fakeDate := time.Date(1990, time.December, 25, 1, 1, 1, 1, time.UTC)

	type responseHandlerErrorsMockResult struct {
		input2 error
		times  int
	}
	type responseHandlerFoldersFilesListMockResult struct {
		input2 []*responsehandlermodels.Entry
		times  int
	}
	type s3ClientListFilesAndDirectoriesMockResult struct {
		input2 string
		res    []*s3client.ListElementOutput
		res2   *s3client.ResultInfo
		err    error
		times  int
	}
	type webhookManagerManageGetHooksMockResult struct {
		input2 string
		input3 string
		input4 *webhook.GetInputMetadata
		input5 *webhook.S3Metadata
		times  int
	}
	type fields struct {
		targetCfg *config.TargetConfig
		mountPath string
	}
	type args struct {
		input *GetInput
	}
	tests := []struct {
		name                                         string
		fields                                       fields
		args                                         args
		user                                         models.GenericUser
		responseHandlerInternalServerErrorMockResult responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult      responseHandlerErrorsMockResult
		responseHandlerFoldersFilesListMockResult    responseHandlerFoldersFilesListMockResult
		s3ClientListFilesAndDirectoriesMockResult    s3ClientListFilesAndDirectoriesMockResult
		s3clManagerClientForTargetMockInput          string
		webhookManagerManageGetHooksMockResult       webhookManagerManageGetHooksMockResult
	}{
		{
			name: "alice listing / should list data/alice/ and strip username from Path",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Name: "bucket1", Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                                args{input: &GetInput{RequestPath: "/"}},
			user:                                &models.BasicAuthUser{Username: "alice"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "data/alice/",
				res: []*s3client.ListElementOutput{
					{Name: "file1.txt", Type: s3client.FileType, Key: "data/alice/file1.txt", LastModified: fakeDate},
					{Name: "sub", Type: s3client.FolderType, Key: "data/alice/sub/", LastModified: fakeDate},
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerFoldersFilesListMockResult: responseHandlerFoldersFilesListMockResult{
				input2: []*responsehandlermodels.Entry{
					{
						Type:         s3client.FileType,
						LastModified: fakeDate,
						Name:         "file1.txt",
						Key:          "data/alice/file1.txt",
						Path:         "/mount/file1.txt",
					},
					{
						Type:         s3client.FolderType,
						LastModified: fakeDate,
						Name:         "sub",
						Key:          "data/alice/sub/",
						Path:         "/mount/sub/",
					},
				},
				times: 1,
			},
		},
		{
			name: "bob listing /sub/ should list data/bob/sub/ and show Path without username",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Name: "bucket1", Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                                args{input: &GetInput{RequestPath: "/sub/"}},
			user:                                &models.BasicAuthUser{Username: "bob"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "data/bob/sub/",
				res: []*s3client.ListElementOutput{
					{Name: "deep.txt", Type: s3client.FileType, Key: "data/bob/sub/deep.txt", LastModified: fakeDate},
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/sub/",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerFoldersFilesListMockResult: responseHandlerFoldersFilesListMockResult{
				input2: []*responsehandlermodels.Entry{
					{
						Type:         s3client.FileType,
						LastModified: fakeDate,
						Name:         "deep.txt",
						Key:          "data/bob/sub/deep.txt",
						Path:         "/mount/sub/deep.txt",
					},
				},
				times: 1,
			},
		},
		{
			name: "charlie listing /alice/ stays confined to data/charlie/alice/ (empty, no leak)",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Name: "bucket1", Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                                args{input: &GetInput{RequestPath: "/alice/"}},
			user:                                &models.BasicAuthUser{Username: "charlie"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "data/charlie/alice/",
				res:    []*s3client.ListElementOutput{},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/alice/",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerFoldersFilesListMockResult: responseHandlerFoldersFilesListMockResult{
				input2: []*responsehandlermodels.Entry{},
				times:  1,
			},
		},
		{
			name: "admin listing / sees raw bucket and Path stripped of bucket prefix only",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Name: "bucket1", Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{
								UserIsolation:       true,
								UserIsolationAdmins: []string{"admin"},
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                                args{input: &GetInput{RequestPath: "/"}},
			user:                                &models.BasicAuthUser{Username: "admin"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "data/",
				res: []*s3client.ListElementOutput{
					{Name: "alice", Type: s3client.FolderType, Key: "data/alice/", LastModified: fakeDate},
					{Name: "bob", Type: s3client.FolderType, Key: "data/bob/", LastModified: fakeDate},
					{Name: "charlie", Type: s3client.FolderType, Key: "data/charlie/", LastModified: fakeDate},
				},
				res2: &s3client.ResultInfo{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			webhookManagerManageGetHooksMockResult: webhookManagerManageGetHooksMockResult{
				input2: "target",
				input3: "/",
				input4: &webhook.GetInputMetadata{},
				input5: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerFoldersFilesListMockResult: responseHandlerFoldersFilesListMockResult{
				input2: []*responsehandlermodels.Entry{
					{
						Type:         s3client.FolderType,
						LastModified: fakeDate,
						Name:         "alice",
						Key:          "data/alice/",
						Path:         "/mount/alice/",
					},
					{
						Type:         s3client.FolderType,
						LastModified: fakeDate,
						Name:         "bob",
						Key:          "data/bob/",
						Path:         "/mount/bob/",
					},
					{
						Type:         s3client.FolderType,
						LastModified: fakeDate,
						Name:         "charlie",
						Key:          "data/charlie/",
						Path:         "/mount/charlie/",
					},
				},
				times: 1,
			},
		},
		{
			name: "should forbid GET when isolation is on and no user is set",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Name: "bucket1", Prefix: "data/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{UserIsolation: true},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{input: &GetInput{RequestPath: "/secret.txt"}},
			user: nil,
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errUserIsolationForbidden,
				times:  1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)

			ctx := context.TODO()
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			resHandlerMock.EXPECT().
				InternalServerError(gomock.Any(), tt.responseHandlerInternalServerErrorMockResult.input2).
				Times(tt.responseHandlerInternalServerErrorMockResult.times)
			resHandlerMock.EXPECT().
				ForbiddenError(gomock.Any(), tt.responseHandlerForbiddenErrorMockResult.input2).
				Times(tt.responseHandlerForbiddenErrorMockResult.times)
			resHandlerMock.EXPECT().NotFoundError(gomock.Any()).Times(0)
			resHandlerMock.EXPECT().StreamFile(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			resHandlerMock.EXPECT().NotModified().Times(0)
			resHandlerMock.EXPECT().PreconditionFailed().Times(0)
			resHandlerMock.EXPECT().
				FoldersFilesList(gomock.Any(), tt.responseHandlerFoldersFilesListMockResult.input2).
				Times(tt.responseHandlerFoldersFilesListMockResult.times)

			s3ClientMock.EXPECT().HeadObject(ctx, gomock.Any()).Return(nil, nil, nil).AnyTimes()
			s3ClientMock.EXPECT().GetObject(ctx, gomock.Any()).Return(nil, nil, nil).AnyTimes()
			s3ClientMock.EXPECT().
				ListFilesAndDirectories(ctx, tt.s3ClientListFilesAndDirectoriesMockResult.input2).
				Return(
					tt.s3ClientListFilesAndDirectoriesMockResult.res,
					tt.s3ClientListFilesAndDirectoriesMockResult.res2,
					tt.s3ClientListFilesAndDirectoriesMockResult.err,
				).
				Times(tt.s3ClientListFilesAndDirectoriesMockResult.times)

			s3clManagerMock.EXPECT().
				GetClientForTarget(tt.s3clManagerClientForTargetMockInput).
				AnyTimes().
				Return(s3ClientMock)

			webhookManagerMock.EXPECT().
				ManageGETHooks(
					ctx,
					tt.webhookManagerManageGetHooksMockResult.input2,
					tt.webhookManagerManageGetHooksMockResult.input3,
					tt.webhookManagerManageGetHooksMockResult.input4,
					tt.webhookManagerManageGetHooksMockResult.input5,
				).
				Times(tt.webhookManagerManageGetHooksMockResult.times)

			rctx := &bucketReqImpl{
				s3ClientManager: s3clManagerMock,
				webhookManager:  webhookManagerMock,
				targetCfg:       tt.fields.targetCfg,
				mountPath:       tt.fields.mountPath,
			}
			rctx.Get(ctx, tt.args.input)
		})
	}
}
