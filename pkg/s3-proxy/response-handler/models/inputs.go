package models

import (
	"io"
	"time"
)

// Entry Entry with path for internal use (template).
type Entry struct {
	LastModified time.Time
	Type         string
	ETag         string
	Name         string
	Key          string
	Path         string
	Size         int64
}

// StreamInput represents a stream input file.
type StreamInput struct {
	LastModified       time.Time
	Body               io.ReadCloser
	Metadata           map[string]string
	CacheControl       string
	Expires            string
	ContentDisposition string
	ContentEncoding    string
	ContentLanguage    string
	ContentRange       string
	ContentType        string
	ETag               string
	ContentLength      int64
}

// PutInput represents a put input.
type PutInput struct {
	Metadata     map[string]string
	Key          string
	ContentType  string
	StorageClass string
	Filename     string
	ContentSize  int64
}

// DeleteInput represents a delete input.
type DeleteInput struct {
	Key string
}
