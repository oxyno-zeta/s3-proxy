// +build unit

package models

import (
	"testing"
)

func TestOIDCUser_GetType(t *testing.T) {
	type fields struct {
		PreferredUsername string
		Name              string
		Groups            []string
		GivenName         string
		FamilyName        string
		Email             string
		EmailVerified     bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "valid",
			fields: fields{},
			want:   OIDCUserType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &OIDCUser{
				PreferredUsername: tt.fields.PreferredUsername,
				Name:              tt.fields.Name,
				Groups:            tt.fields.Groups,
				GivenName:         tt.fields.GivenName,
				FamilyName:        tt.fields.FamilyName,
				Email:             tt.fields.Email,
				EmailVerified:     tt.fields.EmailVerified,
			}
			if got := u.GetType(); got != tt.want {
				t.Errorf("OIDCUser.GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCUser_GetIdentifier(t *testing.T) {
	type fields struct {
		PreferredUsername string
		Name              string
		Groups            []string
		GivenName         string
		FamilyName        string
		Email             string
		EmailVerified     bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "all empty",
			fields: fields{
				PreferredUsername: "",
				Email:             "",
			},
			want: "",
		},
		{
			name: "empty email",
			fields: fields{
				PreferredUsername: "username",
				Email:             "",
			},
			want: "username",
		},
		{
			name: "empty username",
			fields: fields{
				PreferredUsername: "",
				Email:             "email",
			},
			want: "email",
		},
		{
			name: "all set",
			fields: fields{
				PreferredUsername: "username",
				Email:             "email",
			},
			want: "email",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &OIDCUser{
				PreferredUsername: tt.fields.PreferredUsername,
				Name:              tt.fields.Name,
				Groups:            tt.fields.Groups,
				GivenName:         tt.fields.GivenName,
				FamilyName:        tt.fields.FamilyName,
				Email:             tt.fields.Email,
				EmailVerified:     tt.fields.EmailVerified,
			}
			if got := u.GetIdentifier(); got != tt.want {
				t.Errorf("OIDCUser.GetIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}
