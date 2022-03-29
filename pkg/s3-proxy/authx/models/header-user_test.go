//go:build unit

package models

import (
	"reflect"
	"testing"
)

func TestHeaderUser_GetType(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "valid",
			fields: fields{},
			want:   HeaderUserType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetType(); got != tt.want {
				t.Errorf("HeaderUser.GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_GetIdentifier(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "all empty",
			fields: fields{
				Username: "",
				Email:    "",
			},
			want: "",
		},
		{
			name: "empty email",
			fields: fields{
				Username: "username",
				Email:    "",
			},
			want: "username",
		},
		{
			name: "empty username",
			fields: fields{
				Username: "",
				Email:    "email",
			},
			want: "email",
		},
		{
			name: "all set",
			fields: fields{
				Username: "username",
				Email:    "email",
			},
			want: "username",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetIdentifier(); got != tt.want {
				t.Errorf("HeaderUser.GetIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_GetUsername(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "username",
				Email:    "email",
			},
			want: "username",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetUsername(); got != tt.want {
				t.Errorf("HeaderUser.GetUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_GetName(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "username",
				Email:    "email",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetName(); got != tt.want {
				t.Errorf("HeaderUser.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_GetGroups(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "username",
				Email:    "email",
				Groups:   []string{"fake"},
			},
			want: []string{"fake"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetGroups(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HeaderUser.GetGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_GetGivenName(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "username",
				Email:    "email",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetGivenName(); got != tt.want {
				t.Errorf("HeaderUser.GetGivenName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_GetFamilyName(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "username",
				Email:    "email",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetFamilyName(); got != tt.want {
				t.Errorf("HeaderUser.GetFamilyName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_GetEmail(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "username",
				Email:    "email",
			},
			want: "email",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.GetEmail(); got != tt.want {
				t.Errorf("HeaderUser.GetEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderUser_IsEmailVerified(t *testing.T) {
	type fields struct {
		Username string
		Groups   []string
		Email    string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "valid",
			fields: fields{
				Username: "username",
				Email:    "email",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &HeaderUser{
				Username: tt.fields.Username,
				Groups:   tt.fields.Groups,
				Email:    tt.fields.Email,
			}
			if got := u.IsEmailVerified(); got != tt.want {
				t.Errorf("HeaderUser.IsEmailVerified() = %v, want %v", got, tt.want)
			}
		})
	}
}
