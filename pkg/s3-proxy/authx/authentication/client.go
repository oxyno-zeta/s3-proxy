package authentication

import (
	"net/http"

	oidc "github.com/coreos/go-oidc"
	"github.com/go-chi/chi"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
)

type Client interface {
	// Middleware will redirect authentication to basic auth or OIDC depending on request path and resources declared
	Middleware(resources []*config.Resource) func(http.Handler) http.Handler
	// OIDCEndpoints will set OpenID Connect endpoints for authentication and callback
	OIDCEndpoints(providerKey string, oidcCfg *config.OIDCAuthConfig, mux chi.Router) error
}

func NewAuthenticationService(cfg *config.Config, metricsCl metrics.Client) Client {
	return &service{
		allVerifiers: map[string]*oidc.IDTokenVerifier{},
		cfg:          cfg,
		metricsCl:    metricsCl,
	}
}
