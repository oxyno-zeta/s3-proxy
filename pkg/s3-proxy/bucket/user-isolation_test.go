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
			name: "should return true when user is in admin list",
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

func Test_checkUserIsolationAccess(t *testing.T) {
	tests := []struct {
		name      string
		targetCfg *config.TargetConfig
		user      models.GenericUser
		s3Key     string
		want      bool
	}{
		{
			name: "should allow access when isolation is disabled",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{},
					},
				},
			},
			user:  &models.BasicAuthUser{Username: "alice"},
			s3Key: "/bob/file.txt",
			want:  true,
		},
		{
			name: "should deny access when user is nil",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					},
				},
			},
			user:  nil,
			s3Key: "/alice/file.txt",
			want:  false,
		},
		{
			name: "should allow access to own folder",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					},
				},
			},
			user:  &models.BasicAuthUser{Username: "alice"},
			s3Key: "/alice/file.txt",
			want:  true,
		},
		{
			name: "should deny access to another user's folder",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					},
				},
			},
			user:  &models.BasicAuthUser{Username: "alice"},
			s3Key: "/bob/file.txt",
			want:  false,
		},
		{
			name: "should allow admin to access another user's folder",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin"},
						},
					},
				},
			},
			user:  &models.BasicAuthUser{Username: "admin"},
			s3Key: "/bob/file.txt",
			want:  true,
		},
		{
			name: "should work with bucket prefix",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					},
				},
			},
			user:  &models.BasicAuthUser{Username: "alice"},
			s3Key: "/data/alice/file.txt",
			want:  true,
		},
		{
			name: "should deny access with bucket prefix to another user's folder",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/data/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					},
				},
			},
			user:  &models.BasicAuthUser{Username: "alice"},
			s3Key: "/data/bob/file.txt",
			want:  false,
		},
		{
			name: "should deny access when username is a prefix of folder name",
			targetCfg: &config.TargetConfig{
				Bucket: &config.BucketConfig{Prefix: "/"},
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					},
				},
			},
			user:  &models.BasicAuthUser{Username: "ali"},
			s3Key: "/alice/file.txt",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			bri := &bucketReqImpl{
				targetCfg: tt.targetCfg,
			}
			got := bri.checkUserIsolationAccess(ctx, tt.s3Key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_filterS3EntriesByUser(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name                string
		user                models.GenericUser
		s3Entries           []*s3client.ListElementOutput
		bucketRootPrefixKey string
		adminUsers          []string
		want                []*s3client.ListElementOutput
	}{
		{
			name:                "should return empty when user is nil",
			user:                nil,
			s3Entries:           []*s3client.ListElementOutput{{Key: "/alice/file.txt"}},
			bucketRootPrefixKey: "/",
			adminUsers:          nil,
			want:                []*s3client.ListElementOutput{},
		},
		{
			name: "should filter entries to show only user's files",
			user: &models.BasicAuthUser{Username: "alice"},
			s3Entries: []*s3client.ListElementOutput{
				{Key: "/alice/file1.txt", Name: "file1.txt", Type: s3client.FileType, LastModified: now, Size: 100},
				{Key: "/bob/file2.txt", Name: "file2.txt", Type: s3client.FileType, LastModified: now, Size: 200},
				{Key: "/alice/file3.txt", Name: "file3.txt", Type: s3client.FileType, LastModified: now, Size: 300},
			},
			bucketRootPrefixKey: "/",
			adminUsers:          nil,
			want: []*s3client.ListElementOutput{
				{Key: "/alice/file1.txt", Name: "file1.txt", Type: s3client.FileType, LastModified: now, Size: 100},
				{Key: "/alice/file3.txt", Name: "file3.txt", Type: s3client.FileType, LastModified: now, Size: 300},
			},
		},
		{
			name: "should return all entries for admin user",
			user: &models.BasicAuthUser{Username: "admin"},
			s3Entries: []*s3client.ListElementOutput{
				{Key: "/alice/file1.txt", Name: "file1.txt", Type: s3client.FileType, LastModified: now, Size: 100},
				{Key: "/bob/file2.txt", Name: "file2.txt", Type: s3client.FileType, LastModified: now, Size: 200},
			},
			bucketRootPrefixKey: "/",
			adminUsers:          []string{"admin"},
			want: []*s3client.ListElementOutput{
				{Key: "/alice/file1.txt", Name: "file1.txt", Type: s3client.FileType, LastModified: now, Size: 100},
				{Key: "/bob/file2.txt", Name: "file2.txt", Type: s3client.FileType, LastModified: now, Size: 200},
			},
		},
		{
			name:                "should return empty when no entries match user prefix",
			user:                &models.BasicAuthUser{Username: "charlie"},
			s3Entries:           []*s3client.ListElementOutput{{Key: "/alice/file.txt"}, {Key: "/bob/file.txt"}},
			bucketRootPrefixKey: "/",
			adminUsers:          nil,
			want:                []*s3client.ListElementOutput{},
		},
		{
			name:                "should handle empty entries list",
			user:                &models.BasicAuthUser{Username: "alice"},
			s3Entries:           []*s3client.ListElementOutput{},
			bucketRootPrefixKey: "/",
			adminUsers:          nil,
			want:                []*s3client.ListElementOutput{},
		},
		{
			name: "should work with bucket root prefix",
			user: &models.BasicAuthUser{Username: "alice"},
			s3Entries: []*s3client.ListElementOutput{
				{Key: "data/alice/file.txt", Name: "file.txt", Type: s3client.FileType},
				{Key: "data/bob/file.txt", Name: "file.txt", Type: s3client.FileType},
			},
			bucketRootPrefixKey: "data/",
			adminUsers:          nil,
			want: []*s3client.ListElementOutput{
				{Key: "data/alice/file.txt", Name: "file.txt", Type: s3client.FileType},
			},
		},
		{
			name: "should not match partial username prefix",
			user: &models.BasicAuthUser{Username: "ali"},
			s3Entries: []*s3client.ListElementOutput{
				{Key: "/alice/file.txt", Name: "file.txt", Type: s3client.FileType},
			},
			bucketRootPrefixKey: "/",
			adminUsers:          nil,
			want:                []*s3client.ListElementOutput{},
		},
		{
			name: "should include folder entries for user",
			user: &models.BasicAuthUser{Username: "alice"},
			s3Entries: []*s3client.ListElementOutput{
				{Key: "/alice/subfolder/", Name: "subfolder", Type: s3client.FolderType},
				{Key: "/bob/subfolder/", Name: "subfolder", Type: s3client.FolderType},
			},
			bucketRootPrefixKey: "/",
			adminUsers:          nil,
			want: []*s3client.ListElementOutput{
				{Key: "/alice/subfolder/", Name: "subfolder", Type: s3client.FolderType},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			got := filterS3EntriesByUser(ctx, tt.s3Entries, tt.bucketRootPrefixKey, tt.adminUsers)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
		name                                            string
		fields                                          fields
		args                                            args
		user                                            models.GenericUser
		s3clManagerClientForTargetMockInput             string
		responseHandlerDeleteMockResultTimes            responseHandlerDeleteMockResult
		responseHandlerInternalServerErrorMockResult    responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult         responseHandlerErrorsMockResult
		s3ClientDeleteObjectMockResult                  s3ClientDeleteObjectMockResult
		webhookManagerManageDeleteHooksMockResult       webhookManagerManageDeleteHooksMockResult
	}{
		{
			name: "should block delete to another user's file when isolation is enabled",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{
								UserIsolation: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{requestPath: "/bob/file.txt"},
			user: &models.BasicAuthUser{Username: "alice"},
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errUserIsolationForbidden,
				times:  1,
			},
		},
		{
			name: "should allow delete to own file when isolation is enabled",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{
								UserIsolation: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args:                               args{requestPath: "/alice/file.txt"},
			user:                               &models.BasicAuthUser{Username: "alice"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "/alice/file.txt",
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
				input3: "/alice/file.txt",
				input4: &webhook.S3Metadata{
					Bucket:     "bucket",
					Key:        "key",
					Region:     "region",
					S3Endpoint: "s3endpoint",
				},
				times: 1,
			},
			responseHandlerDeleteMockResultTimes: responseHandlerDeleteMockResult{
				input: &responsehandlermodels.DeleteInput{Key: "/alice/file.txt"},
				times: 1,
			},
		},
		{
			name: "should allow admin to delete another user's file",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "bucket",
					Bucket: &config.BucketConfig{Prefix: "/"},
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
			args:                               args{requestPath: "/bob/file.txt"},
			user:                               &models.BasicAuthUser{Username: "admin"},
			s3clManagerClientForTargetMockInput: "bucket",
			s3ClientDeleteObjectMockResult: s3ClientDeleteObjectMockResult{
				input2: "/bob/file.txt",
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
				input: &responsehandlermodels.DeleteInput{Key: "/bob/file.txt"},
				times: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)

			// Create mocks
			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)

			// Create context
			ctx := context.TODO()

			// Add response handler to context
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)

			// Add logger to context
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())

			// Add user to context
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
				Return(
					tt.s3ClientDeleteObjectMockResult.res,
					tt.s3ClientDeleteObjectMockResult.err,
				).
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
		name                                        string
		fields                                      fields
		args                                        args
		user                                        models.GenericUser
		responseHandlerInternalServerErrorMockResult responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult      responseHandlerErrorsMockResult
		responseHandlerPutMockResultTimes            responseHandlerPutMockResult
		s3clManagerClientForTargetMockInput          string
		s3ClientPutObjectMockResult                  s3ClientPutObjectMockResult
		webhookManagerManagePutHooksMockResult       webhookManagerManagePutHooksMockResult
	}{
		{
			name: "should block put to another user's folder when isolation is enabled",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{
								UserIsolation: true,
							},
						},
					},
				},
				mountPath: "/mount",
			},
			args: args{
				inp: &PutInput{
					RequestPath: "/bob",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
			},
			user: &models.BasicAuthUser{Username: "alice"},
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errUserIsolationForbidden,
				times:  1,
			},
		},
		{
			name: "should allow put to own folder when isolation is enabled",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name:   "target",
					Bucket: &config.BucketConfig{Prefix: "/"},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{
							Config: &config.GetActionConfigConfig{
								UserIsolation: true,
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
			user:                               &models.BasicAuthUser{Username: "alice"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientPutObjectMockResult: s3ClientPutObjectMockResult{
				input2: &s3client.PutInput{
					Key:         "/alice/file.txt",
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
					Key:         "/alice/file.txt",
					Filename:    "file.txt",
					ContentType: "text/plain",
				},
				times: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)

			// Create mocks
			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)

			// Create context
			ctx := context.TODO()

			// Add response handler to context
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)

			// Add logger to context
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())

			// Add user to context
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
				Return(
					tt.s3ClientPutObjectMockResult.res,
					tt.s3ClientPutObjectMockResult.err,
				).
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
				Times(
					tt.webhookManagerManagePutHooksMockResult.times,
				)

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
		name                                        string
		fields                                      fields
		args                                        args
		user                                        models.GenericUser
		responseHandlerInternalServerErrorMockResult responseHandlerErrorsMockResult
		responseHandlerForbiddenErrorMockResult      responseHandlerErrorsMockResult
		responseHandlerFoldersFilesListMockResult    responseHandlerFoldersFilesListMockResult
		s3ClientListFilesAndDirectoriesMockResult    s3ClientListFilesAndDirectoriesMockResult
		s3clManagerClientForTargetMockInput          string
		webhookManagerManageGetHooksMockResult       webhookManagerManageGetHooksMockResult
	}{
		{
			name: "should filter directory listing to show only user's entries",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/"},
			},
			user:                               &models.BasicAuthUser{Username: "alice"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "/",
				res: []*s3client.ListElementOutput{
					{Name: "alice", Type: s3client.FolderType, Key: "/alice/", LastModified: fakeDate},
					{Name: "bob", Type: s3client.FolderType, Key: "/bob/", LastModified: fakeDate},
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
						Key:          "/alice/",
						Path:         "/mount/alice/",
					},
				},
				times: 1,
			},
		},
		{
			name: "should block subdirectory listing to another user's folder",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/bob/subdir/"},
			},
			user: &models.BasicAuthUser{Username: "alice"},
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errUserIsolationForbidden,
				times:  1,
			},
		},
		{
			name: "should allow admin to list another user's subdirectory",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: []string{"admin"},
						},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/bob/subdir/"},
			},
			user:                               &models.BasicAuthUser{Username: "admin"},
			s3clManagerClientForTargetMockInput: "target",
			s3ClientListFilesAndDirectoriesMockResult: s3ClientListFilesAndDirectoriesMockResult{
				input2: "/bob/subdir/",
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
				input3: "/bob/subdir/",
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
			name: "should block direct file access to another user's file",
			fields: fields{
				targetCfg: &config.TargetConfig{
					Name: "target",
					Bucket: &config.BucketConfig{
						Name:   "bucket1",
						Prefix: "/",
					},
					Actions: &config.ActionsConfig{GET: &config.GetActionConfig{
						Config: &config.GetActionConfigConfig{
							UserIsolation: true,
						},
					}},
				},
				mountPath: "/mount",
			},
			args: args{
				input: &GetInput{RequestPath: "/bob/secret.txt"},
			},
			user: &models.BasicAuthUser{Username: "alice"},
			responseHandlerForbiddenErrorMockResult: responseHandlerErrorsMockResult{
				input2: errUserIsolationForbidden,
				times:  1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)

			// Create mocks
			resHandlerMock := responsehandlermocks.NewMockResponseHandler(ctrl)
			s3ClientMock := s3clientmocks.NewMockClient(ctrl)
			s3clManagerMock := s3clientmocks.NewMockManager(ctrl)
			webhookManagerMock := wmocks.NewMockManager(ctrl)

			// Create context
			ctx := context.TODO()

			// Add response handler to context
			ctx = responsehandler.SetResponseHandlerInContext(ctx, resHandlerMock)

			// Add logger to context
			ctx = log.SetLoggerInContext(ctx, log.NewLogger())

			// Add user to context
			if tt.user != nil {
				ctx = models.SetAuthenticatedUserInContext(ctx, tt.user)
			}

			resHandlerMock.EXPECT().
				InternalServerError(gomock.Any(), tt.responseHandlerInternalServerErrorMockResult.input2).
				Times(tt.responseHandlerInternalServerErrorMockResult.times)
			resHandlerMock.EXPECT().
				ForbiddenError(gomock.Any(), tt.responseHandlerForbiddenErrorMockResult.input2).
				Times(tt.responseHandlerForbiddenErrorMockResult.times)
			resHandlerMock.EXPECT().
				NotFoundError(gomock.Any()).
				Times(0)
			resHandlerMock.EXPECT().
				StreamFile(gomock.Any(), gomock.Any()).
				Return(nil).
				AnyTimes()
			resHandlerMock.EXPECT().
				NotModified().
				Times(0)
			resHandlerMock.EXPECT().
				PreconditionFailed().
				Times(0)
			resHandlerMock.EXPECT().
				FoldersFilesList(gomock.Any(), tt.responseHandlerFoldersFilesListMockResult.input2).
				Times(tt.responseHandlerFoldersFilesListMockResult.times)

			s3ClientMock.EXPECT().
				HeadObject(ctx, gomock.Any()).
				Return(nil, nil, nil).
				AnyTimes()
			s3ClientMock.EXPECT().
				GetObject(ctx, gomock.Any()).
				Return(nil, nil, nil).
				AnyTimes()
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
				Times(
					tt.webhookManagerManageGetHooksMockResult.times,
				)

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
