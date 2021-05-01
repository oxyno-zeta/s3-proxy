// +build unit

package models

import (
	"reflect"
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

func TestBasicAuthUser_GetUsername(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "basic",
			},
			want: "basic",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetUsername(); got != tt.want {
				t.Errorf("BasicAuthUser.GetUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuthUser_GetName(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "basic",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetName(); got != tt.want {
				t.Errorf("BasicAuthUser.GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuthUser_GetGroups(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "basic",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetGroups(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BasicAuthUser.GetGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuthUser_GetGivenName(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "basic",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetGivenName(); got != tt.want {
				t.Errorf("BasicAuthUser.GetGivenName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuthUser_GetFamilyName(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "basic",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetFamilyName(); got != tt.want {
				t.Errorf("BasicAuthUser.GetFamilyName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuthUser_GetEmail(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid",
			fields: fields{
				Username: "basic",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.GetEmail(); got != tt.want {
				t.Errorf("BasicAuthUser.GetEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuthUser_IsEmailVerified(t *testing.T) {
	type fields struct {
		Username string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "valid",
			fields: fields{
				Username: "basic",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &BasicAuthUser{
				Username: tt.fields.Username,
			}
			if got := u.IsEmailVerified(); got != tt.want {
				t.Errorf("BasicAuthUser.IsEmailVerified() = %v, want %v", got, tt.want)
			}
		})
	}
}
