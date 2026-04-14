//go:build unit

package middlewares

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPathTraversalAttempt(t *testing.T) {
	tests := []struct {
		name        string
		escapedPath string
		want        bool
		wantErr     bool
	}{
		{
			name:        ".. traversal is detected",
			escapedPath: "/public/../secret/traversal.txt",
			want:        true,
		},
		{
			name:        ".. at root is detected",
			escapedPath: "/../secret/file.txt",
			want:        true,
		},
		{
			name:        "single . segment is detected",
			escapedPath: "/public/./file.txt",
			want:        true,
		},
		{
			name:        "%2E%2E traversal is detected",
			escapedPath: "/public/%2E%2E/secret/file.txt",
			want:        true,
		},
		{
			name:        "lowercase %2e%2e traversal is detected",
			escapedPath: "/public/%2e%2e/secret/file.txt",
			want:        true,
		},
		{
			name:        "mixed-case %2E%2e traversal is detected",
			escapedPath: "/public/%2E%2e/secret/file.txt",
			want:        true,
		},
		{
			name:        "%2E%2E at root is detected",
			escapedPath: "/%2E%2E/secret/file.txt",
			want:        true,
		},
		{
			name:        "single %2E segment is detected",
			escapedPath: "/public/%2E/file.txt",
			want:        true,
		},
		{
			name:        "mixed .%2E traversal is detected",
			escapedPath: "/public/.%2E/secret/file.txt",
			want:        true,
		},
		{
			name:        "mixed %2E. traversal is detected",
			escapedPath: "/public/%2E./secret/file.txt",
			want:        true,
		},
		{
			name:        "%2F-based double traversal is detected",
			escapedPath: "/open/value%2F..%2F..%2Frestricted/file.txt",
			want:        true,
		},
		{
			name:        "%2F-based single traversal is detected",
			escapedPath: "/public/foo%2F..%2Fsecret/file.txt",
			want:        true,
		},
		{
			name:        "%2F-based traversal mixed with legitimate %2F is detected",
			escapedPath: "/upload/foo%2F..%2Fbar%2Fbaz/file.txt",
			want:        true,
		},
		{
			name:        "invalid percent-encoding returns error",
			escapedPath: "/public/%zz/file.txt",
			wantErr:     true,
		},
		{
			name:        "consecutive slashes are allowed (accidental double-slash)",
			escapedPath: "/public//file.txt",
			want:        false,
		},
		{
			name:        ".foo filename is allowed",
			escapedPath: "/dir/.foo/",
			want:        false,
		},
		{
			name:        "..foo filename is allowed",
			escapedPath: "/dir/..foo/",
			want:        false,
		},
		{
			name:        "dir.with..dots is allowed",
			escapedPath: "/dir.with..dots/",
			want:        false,
		},
		{
			name:        "%2Efoo encoded filename is allowed",
			escapedPath: "/dir/%2Efoo",
			want:        false,
		},
		{
			name:        "%2E%2Efoo encoded filename is allowed",
			escapedPath: "/dir/%2E%2Efoo",
			want:        false,
		},
		{
			name:        "%2F encoded slash in resource name is allowed",
			escapedPath: "/upload/foo%2Fbar/file.txt",
			want:        false,
		},
		{
			name:        "%20 encoded space is allowed",
			escapedPath: "/public/my%20file.txt",
			want:        false,
		},
		{
			name:        "normal path is allowed",
			escapedPath: "/public/normal.txt",
			want:        false,
		},
		{
			name:        "trailing slash is allowed",
			escapedPath: "/public/dir/",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isPathTraversalAttempt(tt.escapedPath)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
