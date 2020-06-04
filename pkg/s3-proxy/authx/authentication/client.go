package authentication

import (
	"net/http"

	oidc "github.com/coreos/go-oidc"
	"github.com/go-chi/chi"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

type Client interface {
	// Middleware will redirect authentication to basic auth or OIDC depending on request path and resources declared
	Middleware(resources []*config.Resource) func(http.Handler) http.Handler
	// OIDCEndpoints will set OpenID Connect endpoints for authentication and callback
	OIDCEndpoints(oidcCfg *config.OIDCAuthConfig, mux chi.Router) error
}

func NewAuthenticationService(cfg *config.Config) Client {
	return &service{
		allVerifiers: make([]*oidc.IDTokenVerifier, 0),
		cfg:          cfg,
	}
}
