package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/gobwas/glob"
)

// Fork dead project https://github.com/go-chi/hostrouter/
// Add wildcard support, not found handler and internal server handler
// Remove not necessary parts

type HostRouter struct {
	routes                map[string]chi.Router
	notFoundHandler       http.HandlerFunc
	internalServerHandler func(err error) http.HandlerFunc
}

func NewHostRouter(notFoundHandler http.HandlerFunc, internalServerHandler func(err error) http.HandlerFunc) HostRouter {
	return HostRouter{
		routes:                map[string]chi.Router{},
		notFoundHandler:       notFoundHandler,
		internalServerHandler: internalServerHandler,
	}
}

func (hr HostRouter) Get(domain string) chi.Router {
	return hr.routes[domain]
}

func (hr HostRouter) Map(host string, h chi.Router) {
	hr.routes[strings.ToLower(host)] = h
}

func (hr HostRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get host
	host := requestHost(r)

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

func requestHost(r *http.Request) string {
	// not standard, but most popular
	host := r.Header.Get("X-Forwarded-Host")
	if host != "" {
		return host
	}

	// RFC 7239
	host = r.Header.Get("Forwarded")
	_, _, host = parseForwarded(host)

	if host != "" {
		return host
	}

	// if all else fails fall back to request host
	host = r.Host

	return host
}

func parseForwarded(forwarded string) (addr, proto, host string) {
	if forwarded == "" {
		return
	}

	for _, forwardedPair := range strings.Split(forwarded, ";") {
		if tv := strings.SplitN(forwardedPair, "=", 2); len(tv) == 2 {
			token, value := tv[0], tv[1]
			token = strings.TrimSpace(token)
			value = strings.TrimSpace(strings.Trim(value, `"`))

			switch strings.ToLower(token) {
			case "for":
				addr = value
			case "proto":
				proto = value
			case "host":
				host = value
			}
		}
	}

	return
}

func (hr HostRouter) getRouterWithWildcard(host string) (chi.Router, error) {
	for wh, rt := range hr.routes {
		g, err := glob.Compile(wh)
		// Check if error exists
		if err != nil {
			return nil, err
		}
		// Check if wildcard host match current host
		if g.Match(host) {
			return rt, nil
		}
	}

	// Default case
	return nil, nil
}
