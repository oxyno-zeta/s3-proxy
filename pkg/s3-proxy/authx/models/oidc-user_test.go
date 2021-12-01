//go:build unit

package models

import (
	"reflect"
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
			want: "username",
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

func TestOIDCUser_GetUsername(t *testing.T) {
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
			name: "valid",
			fields: fields{
				PreferredUsername: "username",
				Email:             "email",
			},
			want: "username",
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
			if got := u.GetUsername(); got != tt.want {
				t.Errorf("OIDCUser.GetUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCUser_GetName(t *testing.T) {
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
			name: "valid",
			fields: fields{
				PreferredUsername: "username",
				Email:             "email",
				Name:              "name",
			},
			want: "name",
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
			if got := u.GetName(); got != tt.want {
				t.Errorf("OIDCUser.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCUser_GetGroups(t *testing.T) {
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
		want   []string
	}{
		{
			name: "valid",
			fields: fields{
				PreferredUsername: "username",
				Email:             "email",
				Groups:            []string{"fake"},
			},
			want: []string{"fake"},
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
			if got := u.GetGroups(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OIDCUser.GetGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCUser_GetGivenName(t *testing.T) {
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
			name: "valid",
			fields: fields{
				PreferredUsername: "username",
				Email:             "email",
				GivenName:         "given name",
			},
			want: "given name",
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
			if got := u.GetGivenName(); got != tt.want {
				t.Errorf("OIDCUser.GetGivenName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCUser_GetFamilyName(t *testing.T) {
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
			name: "valid",
			fields: fields{
				PreferredUsername: "username",
				Email:             "email",
				FamilyName:        "family name",
			},
			want: "family name",
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
			if got := u.GetFamilyName(); got != tt.want {
				t.Errorf("OIDCUser.GetFamilyName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCUser_GetEmail(t *testing.T) {
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
			name: "valid",
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
			if got := u.GetEmail(); got != tt.want {
				t.Errorf("OIDCUser.GetEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCUser_IsEmailVerified(t *testing.T) {
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
		want   bool
	}{
		{
			name: "valid",
			fields: fields{
				PreferredUsername: "username",
				Email:             "email",
				EmailVerified:     true,
			},
			want: true,
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
			if got := u.IsEmailVerified(); got != tt.want {
				t.Errorf("OIDCUser.IsEmailVerified() = %v, want %v", got, tt.want)
			}
		})
	}
}
