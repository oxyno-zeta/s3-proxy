package utils

import (
	"net/http"
)

type respWriterTest struct {
	Headers http.Header
	Status  int
	Resp    []byte
}

func (r *respWriterTest) Header() http.Header          { return r.Headers }
func (r *respWriterTest) Write(in []byte) (int, error) { r.Resp = in; return len(in), nil }
func (r *respWriterTest) WriteHeader(s int)            { r.Status = s }
