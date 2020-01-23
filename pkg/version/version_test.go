package version

import (
	"reflect"
	"testing"
)

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name string
		want *AppVersion
	}{
		{name: "test", want: &AppVersion{Version: "-unreleased", GitCommit: "", BuildDate: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVersion(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
