package bucket

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
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
	Err          error
	ListResult   []*s3client.ListElementOutput
	HeadResult   *s3client.HeadOutput
	GetResult    *s3client.GetOutput
	ListCalled   bool
	HeadCalled   bool
	GetCalled    bool
	PutCalled    bool
	DeleteCalled bool
}

func (s *s3clientTest) ListFilesAndDirectories(key string) ([]*s3client.ListElementOutput, error) {
	s.ListCalled = true
	return s.ListResult, s.Err
}

func (s *s3clientTest) HeadObject(key string) (*s3client.HeadOutput, error) {
	s.HeadCalled = true
	return s.HeadResult, s.Err
}

func (s *s3clientTest) GetObject(key string) (*s3client.GetOutput, error) {
	s.GetCalled = true
	return s.GetResult, s.Err
}

func (s *s3clientTest) PutObject(input *s3client.PutInput) error {
	s.PutCalled = true
	return s.Err
}

func (s *s3clientTest) DeleteObject(key string) error {
	s.DeleteCalled = true
	return s.Err
}
