//go:build unit

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

func TestPutActionConfigConfig_GetFormFieldName(t *testing.T) {
	tests := []struct {
		name   string
		config *PutActionConfigConfig
		want   string
	}{
		{
			name:   "Must return default form field name when config is not set",
			config: nil,
			want:   "file",
		},
		{
			name:   "Must return default form field name when not set",
			config: &PutActionConfigConfig{StorageClass: "STANDARD"},
			want:   "file",
		},
		{
			name:   "Must return configured form field name",
			config: &PutActionConfigConfig{FormFieldName: new("upload")},
			want:   "upload",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetFormFieldName(); got != tt.want {
				t.Errorf("PutActionConfigConfig.GetFormFieldName() = %v, want %v", got, tt.want)
			}
		})
	}
}
