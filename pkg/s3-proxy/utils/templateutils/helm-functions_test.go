//go:build unit

package templateutils

import (
	"testing"
)

func Test_toJSON(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "ok",
			args: args{
				v: map[string]interface{}{
					"number": 1,
					"string": "str",
				},
			},
			want: `{"number":1,"string":"str"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toJSON(tt.args.v); got != tt.want {
				t.Errorf("toJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toYAML(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "ok",
			args: args{
				v: map[string]interface{}{
					"number": 1,
					"string": "str",
				},
			},
			want: `number: 1
string: str`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toYAML(tt.args.v); got != tt.want {
				t.Errorf("toYAML() = %v, want %v", got, tt.want)
			}
		})
	}
}
