// +build unit

package log

import (
	"testing"

	logrus "github.com/sirupsen/logrus"
)

func Test_loggerIns_Configure(t *testing.T) {
	type args struct {
		level    string
		format   string
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Cannot parse log level",
			args: args{
				level: "fake",
			},
			wantErr: true,
		},
		{
			name: "Parse log level ok",
			args: args{
				level: "info",
			},
			wantErr: false,
		},
		{
			name: "Format json ok",
			args: args{
				level:  "info",
				format: "json",
			},
			wantErr: false,
		},
		{
			name: "Create log file",
			args: args{
				level:    "info",
				format:   "json",
				filePath: "/tmp/fake-s3-proxy-log/dir/s3-proxy.log",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ll := &loggerIns{
				FieldLogger: logrus.New(),
			}
			if err := ll.Configure(tt.args.level, tt.args.format, tt.args.filePath); (err != nil) != tt.wantErr {
				t.Errorf("loggerIns.Configure() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
