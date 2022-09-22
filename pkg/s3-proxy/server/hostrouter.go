package server

import (
	"net/http"
	"strings"

	"emperror.dev/errors"
	"github.com/go-chi/chi/v5"
	"github.com/gobwas/glob"
	utils "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
)

// Fork dead project https://github.com/go-chi/hostrouter/
// Add wildcard support, not found handler and internal server handler
// Remove not necessary parts
// Update to ensure that all wildcard domains will be tested in the injection order

type HostRouter struct {
	domainList            []string
	routes                map[string]chi.Router
	notFoundHandler       http.HandlerFunc
	internalServerHandler func(err error) http.HandlerFunc
}

func NewHostRouter(notFoundHandler http.HandlerFunc, internalServerHandler func(err error) http.HandlerFunc) HostRouter {
	return HostRouter{
		domainList:            []string{},
		routes:                map[string]chi.Router{},
		notFoundHandler:       notFoundHandler,
		internalServerHandler: internalServerHandler,
	}
}

func (hr *HostRouter) Get(domain string) chi.Router {
	return hr.routes[strings.ToLower(domain)]
}

func (hr *HostRouter) Map(host string, h chi.Router) {
	lowercaseHost := strings.ToLower(host)
	hr.domainList = append(hr.domainList, lowercaseHost)
	hr.routes[lowercaseHost] = h
}

func (hr HostRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get host
	host := utils.GetRequestHost(r)

	// Check if host is matching directly
	if router, ok := hr.routes[strings.ToLower(host)]; ok {
		router.ServeHTTP(w, r)

		return
	}

	// Check if host is matching wildcard
	rt, err := hr.getRouterWithWildcard(host)
	// Check error
	if err != nil {
		hr.internalServerHandler(err)(w, r)

		return
	}
	// Check if router exits
	if rt != nil {
		rt.ServeHTTP(w, r)

		return
	}

	hr.notFoundHandler(w, r)
}

func (hr *HostRouter) getRouterWithWildcard(host string) (chi.Router, error) {
	for _, wh := range hr.domainList {
		g, err := glob.Compile(wh)
		// Check if error exists
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// Check if wildcard host match current host
		if g.Match(host) {
			return hr.routes[wh], nil
		}
	}

	// Default case
	return nil, nil
}
