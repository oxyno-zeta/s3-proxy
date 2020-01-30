package middlewares

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path"

	oidc "github.com/coreos/go-oidc"
	"github.com/go-chi/chi"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/utils"
	"github.com/thoas/go-funk"

	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func OIDCEndpoints(oidcCfg *config.OIDCAuthConfig, tplConfig *config.TemplateConfig, mux chi.Router) error {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, oidcCfg.IssuerURL)
	if err != nil {
		return err
	}

	oidcConfig := &oidc.Config{
		ClientID: oidcCfg.ClientID,
	}
	verifier := provider.Verifier(oidcConfig)

	// Build redirect url
	u, err := url.Parse(oidcCfg.RedirectURL)
	// Check if error exists
	if err != nil {
		return err
	}
	// Continue to build redirect url
	u.Path = path.Join(u.Path, oidcCfg.CallbackPath)
	redirectURL := u.String()

	// Create OIDC configuration
	config := oauth2.Config{
		ClientID:    oidcCfg.ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: redirectURL,
		Scopes:      oidcCfg.Scopes,
	}
	if oidcCfg.ClientSecret != nil {
		config.ClientSecret = oidcCfg.ClientSecret.Value
	}

	// Store state
	state := oidcCfg.State

	mux.HandleFunc(oidcCfg.LoginPath, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	mux.HandleFunc(oidcCfg.CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		logEntry := GetLogEntry(r)
		if r.URL.Query().Get("state") != state {
			err := errors.New("state did not match")
			logEntry.Error(err)
			utils.HandleBadRequest(w, oidcCfg.CallbackPath, err, logEntry, tplConfig)
			return
		}

		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			err = errors.New("failed to exchange token: " + err.Error())
			logEntry.Error(err)
			utils.HandleInternalServerError(w, err, oidcCfg.CallbackPath, logEntry, tplConfig)
			return
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			err = errors.New("no id_token field in token")
			logEntry.Error(err)
			utils.HandleInternalServerError(w, err, oidcCfg.CallbackPath, logEntry, tplConfig)
			return
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			err = errors.New("failed to verify ID Token: " + err.Error())
			logEntry.Error(err)
			utils.HandleInternalServerError(w, err, oidcCfg.CallbackPath, logEntry, tplConfig)
			return
		}

		resp := struct {
			OAuth2Token   *oauth2.Token
			IDTokenClaims *json.RawMessage // ID Token payload is just JSON.
		}{oauth2Token, new(json.RawMessage)}

		// Try to open JWT token
		err = idToken.Claims(&resp.IDTokenClaims)
		if err != nil {
			logEntry.Error(err)
			utils.HandleInternalServerError(w, err, oidcCfg.CallbackPath, logEntry, tplConfig)
			return
		}

		// Build cookie
		cookie := &http.Cookie{
			Expires:  oauth2Token.Expiry,
			Name:     oidcCfg.CookieName,
			Value:    rawIDToken,
			HttpOnly: true,
			Secure:   oidcCfg.CookieSecure,
			Path:     "/",
		}
		http.SetCookie(w, cookie)

		logEntry.Info("Successful authentication detected")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	return nil
}

// nolint:whitespace
func oidcAuthorizationMiddleware(
	oidcAuthCfg *config.OIDCAuthConfig,
	tplConfig *config.TemplateConfig,
	authorizationAccesses []*config.OIDCAuthorizationAccess,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logEntry := GetLogEntry(r)
			path := r.URL.RequestURI()
			// Try to get auth cookie
			logEntry.Debug("Try get auth cookie from request")
			cookie, err := r.Cookie(oidcAuthCfg.CookieName)
			if err != nil {
				logEntry.Debug("Can't load auth cookie")
				if err != http.ErrNoCookie {
					logEntry.Error(err)
					utils.HandleInternalServerError(w, err, path, logEntry, tplConfig)
					return
				}
				if cookie == nil {
					logEntry.Error("No auth cookie detected, redirect to oidc login")
					http.Redirect(w, r, oidcAuthCfg.LoginPath, http.StatusTemporaryRedirect)
					return
				}
			}
			// Parse JWT token
			parser := new(jwt.Parser)
			token, _, err := parser.ParseUnverified(cookie.Value, jwt.MapClaims{})
			if err != nil {
				logEntry.Error(err)
				utils.HandleInternalServerError(w, err, path, logEntry, tplConfig)
				return
			}
			err = token.Claims.Valid()
			if err != nil {
				logEntry.Error(err)
				utils.HandleInternalServerError(w, err, path, logEntry, tplConfig)
				return
			}

			claims := token.Claims.(jwt.MapClaims)
			// Get email
			email := claims["email"].(string)

			// Check email verified
			if oidcAuthCfg.EmailVerified {
				emailVerified := claims["email_verified"].(bool)
				if !emailVerified {
					logEntry.Errorf("Email not verified for %s", email)
					utils.HandleForbidden(w, path, logEntry, tplConfig)
					return
				}
			}

			// Get groups
			groupsInterface := claims[oidcAuthCfg.GroupClaim]
			groups := make([]string, 0)
			for _, item := range groupsInterface.([]interface{}) {
				groups = append(groups, item.(string))
			}
			// Check if authorized
			if !isAuthorized(groups, email, authorizationAccesses) {
				logEntry.Errorf("Forbidden user %s", email)
				utils.HandleForbidden(w, path, logEntry, tplConfig)
				return
			}

			logEntry.Infof("User authorized and authenticated: %s", email)

			// Next
			next.ServeHTTP(w, r)
		})
	}
}

func isAuthorized(groups []string, email string, authorizationAccesses []*config.OIDCAuthorizationAccess) bool {
	// Check if there is a list of groups or email
	if len(authorizationAccesses) == 0 {
		// No group or email => consider this as authentication only required => ok
		return true
	}

	// Loop over groups and email
	for _, item := range authorizationAccesses {
		if item.Regexp {
			// Regex case
			// Check group case
			if item.Group != "" {
				for _, grp := range groups {
					// Try matching for group regexp
					if item.GroupRegexp.MatchString(grp) {
						return true
					}
				}
			}

			// Check email case
			if item.Email != "" && item.EmailRegexp.MatchString(email) {
				return true
			}
		} else {
			// Not a regex case

			// Check group case
			if item.Group != "" {
				result := funk.Contains(groups, item.Group)
				if result {
					return true
				}
			}
			// Check email case
			if item.Email != "" && item.Email == email {
				return true
			}
		}
	}

	// Not found case
	return false
}
