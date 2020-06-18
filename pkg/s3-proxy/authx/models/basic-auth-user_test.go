// +build unit

package models

import (
	"testing"
)

func TestBasicAuthUser_GetIdentifier(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "empty",
			fields: fields{
				Username: "",
			},
			want: "",
		},
		{
			name: "valid",
			fields: fields{
				Username: "valid",
			},
			want: "valid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetIdentifier(); got != tt.want {
				t.Errorf("BasicAuthUser.GetIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuthUser_GetType(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "valid",
			fields: fields{},
			want:   BasicAuthUserType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetType(); got != tt.want {
				t.Errorf("BasicAuthUser.GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}
