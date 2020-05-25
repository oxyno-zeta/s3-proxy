// +build unit

package config

import (
	"testing"
)

func TestBucketConfig_GetRootPrefix(t *testing.T) {
	type fields struct {
		Name        string
		Prefix      string
		Region      string
		S3Endpoint  string
		Credentials *BucketCredentialConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Must add a / at the end",
			fields: fields{
				Name:   "bucket",
				Prefix: "test",
			},
			want: "test/",
		},
		{
			name: "Must let prefix as provided",
			fields: fields{
				Name:   "bucket",
				Prefix: "test/",
			},
			want: "test/",
		},
		{
			name: "Must let empty prefix",
			fields: fields{
				Name:   "bucket",
				Prefix: "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bcfg := &BucketConfig{
				Name:        tt.fields.Name,
				Prefix:      tt.fields.Prefix,
				Region:      tt.fields.Region,
				S3Endpoint:  tt.fields.S3Endpoint,
				Credentials: tt.fields.Credentials,
			}
			if got := bcfg.GetRootPrefix(); got != tt.want {
				t.Errorf("BucketConfig.GetRootPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func TestConfigureLogger(t *testing.T) {
// 	type args struct {
// 		logger    *logrus.Logger
// 		logConfig *LogConfig
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name: "Cannot parse log level",
// 			args: args{
// 				logger: logrus.New(),
// 				logConfig: &LogConfig{
// 					Level: "fake",
// 				},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "Parse log level ok",
// 			args: args{
// 				logger: logrus.New(),
// 				logConfig: &LogConfig{
// 					Level: "info",
// 				},
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name: "Format json ok",
// 			args: args{
// 				logger: logrus.New(),
// 				logConfig: &LogConfig{
// 					Format: "json",
// 					Level:  "info",
// 				},
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if err := ConfigureLogger(tt.args.logger, tt.args.logConfig); (err != nil) != tt.wantErr {
// 				t.Errorf("ConfigureLogger() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }
