// +build unit

package bucket

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
)

type respWriterTest struct {
	Headers http.Header
	Status  int
	Resp    []byte
}

func (r *respWriterTest) Header() http.Header          { return r.Headers }
func (r *respWriterTest) Write(in []byte) (int, error) { r.Resp = in; return len(in), nil }
func (r *respWriterTest) WriteHeader(s int)            { r.Status = s }

type s3clientTest struct {
	ListErr      error
	HeadErr      error
	GetErr       error
	PutErr       error
	DeleteErr    error
	ListResult   []*s3client.ListElementOutput
	HeadResult   *s3client.HeadOutput
	GetResult    *s3client.GetOutput
	ListCalled   bool
	HeadCalled   bool
	GetCalled    bool
	PutCalled    bool
	DeleteCalled bool
	ListInput    string
	HeadInput    string
	GetInput     string
	PutInput     *s3client.PutInput
	DeleteInput  string
}

func (s *s3clientTest) ListFilesAndDirectories(key string) ([]*s3client.ListElementOutput, error) {
	s.ListInput = key
	s.ListCalled = true
	return s.ListResult, s.ListErr
}

func (s *s3clientTest) HeadObject(key string) (*s3client.HeadOutput, error) {
	s.HeadInput = key
	s.HeadCalled = true
	return s.HeadResult, s.HeadErr
}

func (s *s3clientTest) GetObject(key string) (*s3client.GetOutput, error) {
	s.GetInput = key
	s.GetCalled = true
	return s.GetResult, s.GetErr
}

func (s *s3clientTest) PutObject(input *s3client.PutInput) error {
	s.PutInput = input
	s.PutCalled = true
	return s.PutErr
}

func (s *s3clientTest) DeleteObject(key string) error {
	s.DeleteInput = key
	s.DeleteCalled = true
	return s.DeleteErr
}
