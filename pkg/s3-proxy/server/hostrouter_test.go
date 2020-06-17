// +build unit

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
)

func TestHostRouter_ServeHTTP(t *testing.T) {
	starRouter := chi.NewRouter()
	starRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("star"))
	})
	starLocalhostRouter := chi.NewRouter()
	starLocalhostRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("starLocalhost"))
	})
	localhostRouter := chi.NewRouter()
	localhostRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("localhost"))
	})

	tests := []struct {
		name           string
		inputURL       string
		routes         map[string]chi.Router
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "should match the star glob",
			inputURL: "http://fake/",
			routes: map[string]chi.Router{
				"localhost":   localhostRouter,
				"*.localhost": starLocalhostRouter,
				"*":           starRouter,
			},
			expectedStatus: 200,
			expectedBody:   "star",
		},
		{
			name:     "should match the perfect host",
			inputURL: "http://localhost/",
			routes: map[string]chi.Router{
				"localhost":   localhostRouter,
				"*.localhost": starLocalhostRouter,
				"*":           starRouter,
			},
			expectedStatus: 200,
			expectedBody:   "localhost",
		},
		{
			name:     "should match the glob host",
			inputURL: "http://api.localhost/",
			routes: map[string]chi.Router{
				"localhost":   localhostRouter,
				"*.localhost": starLocalhostRouter,
				"*":           starRouter,
			},
			expectedStatus: 200,
			expectedBody:   "starLocalhost",
		},
		{
			name:     "should match the glob host (2)",
			inputURL: "http://ui.localhost/",
			routes: map[string]chi.Router{
				"localhost":   localhostRouter,
				"*.localhost": starLocalhostRouter,
				"*":           starRouter,
			},
			expectedStatus: 200,
			expectedBody:   "starLocalhost",
		},
		{
			name:     "should return a not found error",
			inputURL: "http://ui.localhost/",
			routes: map[string]chi.Router{
				"localhost": localhostRouter,
			},
			expectedStatus: 404,
			expectedBody:   "hostrouter not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := HostRouter{
				routes: tt.routes,
				notFoundHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(404)
					w.Write([]byte("hostrouter not found"))
				},
				internalServerHandler: func(err error) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(500)
						w.Write([]byte("hostrouter internal server error"))
					}
				},
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequest("GET", tt.inputURL, nil)
			if err != nil {
				t.Fatal(err)
			}

			hr.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}
