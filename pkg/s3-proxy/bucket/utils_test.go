// +build unit

package bucket

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
)

func Test_setHeadersFromObjectOutput(t *testing.T) {
	// Tests data
	now := time.Now()
	headerFullInput := http.Header{}
	headerFullInput.Add("Cache-Control", "cachecontrol")
	headerFullInput.Add("Expires", "expires")
	headerFullInput.Add("Content-Disposition", "contentdisposition")
	headerFullInput.Add("Content-Encoding", "contentencoding")
	headerFullInput.Add("Content-Language", "contentlanguage")
	headerFullInput.Add("Content-Length", "200")
	headerFullInput.Add("Content-Range", "bytes 200/200")
	headerFullInput.Add("Content-Type", "contenttype")
	headerFullInput.Add("ETag", "etag")
	headerFullInput.Add("Last-Modified", now.UTC().Format(http.TimeFormat))
	headerPartialInput := http.Header{}
	headerPartialInput.Add("Cache-Control", "cachecontrol")
	headerPartialInput.Add("Expires", "expires")
	headerPartialInput.Add("Content-Disposition", "contentdisposition")
	headerPartialInput.Add("Content-Encoding", "contentencoding")
	headerPartialInput.Add("Content-Language", "contentlanguage")
	headerPartialInput.Add("Content-Length", "200")
	headerPartialInput.Add("Content-Range", "bytes 200-1000/10000")
	headerPartialInput.Add("Content-Type", "contenttype")
	headerPartialInput.Add("ETag", "etag")
	headerPartialInput.Add("Last-Modified", now.UTC().Format(http.TimeFormat))
	// Structures
	type args struct {
		w   http.ResponseWriter
		obj *s3client.GetOutput
	}
	tests := []struct {
		name     string
		args     args
		expected respWriterTest
	}{
		{
			name: "Empty input",
			args: args{
				w: &respWriterTest{
					Headers: http.Header{},
					Status:  200,
				},
				obj: &s3client.GetOutput{},
			},
			expected: respWriterTest{
				Headers: http.Header{},
				Status:  200,
			},
		},
		{
			name: "Full input",
			args: args{
				w: &respWriterTest{
					Headers: http.Header{},
					Status:  200,
				},
				obj: &s3client.GetOutput{
					CacheControl:       "cachecontrol",
					Expires:            "expires",
					ContentDisposition: "contentdisposition",
					ContentEncoding:    "contentencoding",
					ContentLanguage:    "contentlanguage",
					ContentLength:      200,
					ContentRange:       "bytes 200/200",
					ContentType:        "contenttype",
					ETag:               "etag",
					LastModified:       now,
				},
			},
			expected: respWriterTest{
				Headers: headerFullInput,
				Status:  200,
			},
		},
		{
			name: "Partial input",
			args: args{
				w: &respWriterTest{
					Headers: http.Header{},
					Status:  200,
				},
				obj: &s3client.GetOutput{
					CacheControl:       "cachecontrol",
					Expires:            "expires",
					ContentDisposition: "contentdisposition",
					ContentEncoding:    "contentencoding",
					ContentLanguage:    "contentlanguage",
					ContentLength:      200,
					ContentRange:       "bytes 200-1000/10000",
					ContentType:        "contenttype",
					ETag:               "etag",
					LastModified:       now,
				},
			},
			expected: respWriterTest{
				Headers: headerPartialInput,
				Status:  206,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHeadersFromObjectOutput(tt.args.w, tt.args.obj)
			input := tt.args.w.(*respWriterTest)
			if !reflect.DeepEqual(input.Headers, tt.expected.Headers) && input.Status != tt.expected.Status {
				t.Errorf("setHeadersFromObjectOutput() source = %+v, want %+v", tt.args.w, tt.expected)
			}
		})
	}
}
