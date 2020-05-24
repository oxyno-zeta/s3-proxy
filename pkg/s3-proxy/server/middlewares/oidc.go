package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	oidc "github.com/coreos/go-oidc"
	"github.com/go-chi/chi"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const redirectQueryKey = "rd"

var allVerifiers = make([]*oidc.IDTokenVerifier, 0)

// OIDCEndpoints will set OpenID Connect endpoints for authentication and callback
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
	mainRedirectURLObject, err := url.Parse(oidcCfg.RedirectURL)
	// Check if error exists
	if err != nil {
		return err
	}
	// Continue to build redirect url
	mainRedirectURLObject.Path = path.Join(mainRedirectURLObject.Path, oidcCfg.CallbackPath)
	mainRedirectURLStr := mainRedirectURLObject.String()

	// Create OIDC configuration
	config := oauth2.Config{
		ClientID: oidcCfg.ClientID,
		Endpoint: provider.Endpoint(),
		Scopes:   oidcCfg.Scopes,
	}
	if oidcCfg.ClientSecret != nil {
		config.ClientSecret = oidcCfg.ClientSecret.Value
	}

	// Store state
	state := oidcCfg.State

	// Store provider verifier in map
	allVerifiers = append(allVerifiers, verifier)

	mux.HandleFunc(oidcCfg.LoginPath, func(w http.ResponseWriter, r *http.Request) {
		// Get logger from request
		logEntry := GetLogEntry(r)
		// Parse query params from request
		qs := r.URL.Query()
		// Get redirect query from query params
		rdVal := qs.Get(redirectQueryKey)
		// OIDC Redirect URL
		oidcRedirectURLStr := mainRedirectURLStr
		// Check if redirect url exists
		if rdVal != "" {
			// Need to build new oidc redirect url
			oidcRedirectURL, err := url.Parse(oidcRedirectURLStr)
			// Check if error exists
			if err != nil {
				logEntry.Error(err)
				utils.HandleInternalServerError(w, err, oidcCfg.LoginPath, logEntry, tplConfig)
				return
			}
			qsValues := oidcRedirectURL.Query()
			// Add query param
			qsValues.Add(redirectQueryKey, rdVal)
			// Add query params to oidc redirect url
			oidcRedirectURL.RawQuery = qs.Encode()
			// Build new oidc redirect url string
			oidcRedirectURLStr = oidcRedirectURL.String()
		}
		// Add redirect URL string to oidc configuration
		config.RedirectURL = oidcRedirectURLStr

		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	mux.HandleFunc(mainRedirectURLObject.Path, func(w http.ResponseWriter, r *http.Request) {
		// Get logger from request
		logEntry := GetLogEntry(r)

		// ! In this particular case, no bucket request context because mounted in general and not per target

		// Get query parameters
		qs := r.URL.Query()
		// Get redirect url
		rdVal := qs.Get(redirectQueryKey)
		// Check if rdVal exists and that redirect url value is valid
		if rdVal != "" && !isValidRedirect(rdVal) {
			err := errors.New("redirect url is invalid")
			logEntry.Error(err)
			utils.HandleBadRequest(w, oidcCfg.CallbackPath, err, logEntry, tplConfig)
			return
		}

		// Manage default redirect case after validation
		if rdVal == "" {
			rdVal = "/"
		}

		// Check state
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

		var resp map[string]interface{}

		// Try to open JWT token in order to verify that we can open it
		err = idToken.Claims(&resp)
		if err != nil {
			logEntry.Error(err)
			utils.HandleInternalServerError(w, err, oidcCfg.CallbackPath, logEntry, tplConfig)
			return
		}
		// Now, we know that we can open jwt token to get claims

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
		http.Redirect(w, r, rdVal, http.StatusTemporaryRedirect)
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
			// Get logger from request
			logEntry := GetLogEntry(r)
			path := r.URL.RequestURI()
			// Get bucket request context from request
			brctx := GetBucketRequestContext(r)

			// Get JWT Token from header or cookie
			jwtContent, err := getJWTToken(logEntry, r, oidcAuthCfg.CookieName)
			// Check if error exists
			if err != nil {
				logEntry.Error(err)

				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleInternalServerError(w, err, path, logEntry, tplConfig)
				} else {
					brctx.HandleInternalServerError(err, path)
				}
				return
			}
			// Check if JWT content is empty or not
			if jwtContent == "" {
				logEntry.Error("No auth header or cookie detected, redirect to oidc login")

				// Initialize redirect URI
				rdURI := oidcAuthCfg.LoginPath
				// Check if redirect URI must be created
				// If request path isn't equal to login path, build redirect URI to keep incoming request
				if r.RequestURI != oidcAuthCfg.LoginPath {
					// Build incoming request
					incomingURI := utils.GetRequestURI(r)
					// URL Encode it
					urlEncodedIncomingURI := url.QueryEscape(incomingURI)
					// Build redirect URI
					rdURI = fmt.Sprintf("%s?%s=%s", oidcAuthCfg.LoginPath, redirectQueryKey, urlEncodedIncomingURI)
				}
				// Redirect
				http.Redirect(w, r, rdURI, http.StatusTemporaryRedirect)
				return
			}

			// Parse JWT token
			claims, err := parseAndValidateJWTToken(jwtContent)
			if err != nil {
				logEntry.Error(err)
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleInternalServerError(w, err, path, logEntry, tplConfig)
				} else {
					brctx.HandleInternalServerError(err, path)
				}
				return
			}

			// Initialize email
			email := ""

			// Get email claim
			emailClaim := claims["email"]

			if emailClaim != nil {
				// Get email
				email = emailClaim.(string)

				// Check email verified
				if oidcAuthCfg.EmailVerified {
					emailVerified := claims["email_verified"].(bool)
					if !emailVerified {
						logEntry.Errorf("Email not verified for %s", email)
						// Check if bucket request context doesn't exist to use local default files
						if brctx == nil {
							utils.HandleForbidden(w, path, logEntry, tplConfig)
						} else {
							brctx.HandleForbidden(path)
						}
						return
					}
				}
			}

			// Get groups
			groupsInterface := claims[oidcAuthCfg.GroupClaim]
			groups := make([]string, 0)
			// Check if groups interface exists
			if groupsInterface != nil {
				for _, item := range groupsInterface.([]interface{}) {
					groups = append(groups, item.(string))
				}
			}

			// Check if authorized
			if !isAuthorized(groups, email, authorizationAccesses) {
				logEntry.Errorf("Forbidden user %s", email)
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleForbidden(w, path, logEntry, tplConfig)
				} else {
					brctx.HandleForbidden(path)
				}
				return
			}

			logEntry.Infof("User authorized and authenticated: %s", email)

			// Next
			next.ServeHTTP(w, r)
		})
	}
}

func parseAndValidateJWTToken(jwtContent string) (map[string]interface{}, error) {
	ctx := context.Background()
	// Create result map
	var res map[string]interface{}

	// Loop over all verifiers
	for _, verifier := range allVerifiers {
		// Verify token
		idToken, err := verifier.Verify(ctx, jwtContent)
		// Check if error is present because of invalid provider verifier
		// The error is the one inside the oidc coreos library
		if err != nil && strings.Contains(err.Error(), "token issued by a different provider") {
			// Try with another verifier
			continue
		}
		// Error in this case is a bad error...
		if err != nil {
			return nil, err
		}

		// Get claims
		err = idToken.Claims(&res)
		if err != nil {
			return nil, err
		}

		return res, nil
	}

	// Error, can't be opened, maybe a forged token ?
	return nil, errors.New("jwt token cannot be open with oidc providers in configuration, maybe a forged token ?")
}

func getJWTToken(logEntry logrus.FieldLogger, r *http.Request, cookieName string) (string, error) {
	logEntry.Debug("Try to get Authorization header from request")
	// Get Authorization header
	authHd := r.Header.Get("Authorization")
	// Check if Authorization header is populated
	if authHd != "" {
		// Split header to get token => Format "Bearer TOKEN"
		sp := strings.Split(authHd, " ")
		if len(sp) != 2 || sp[0] != "Bearer" {
			return "", errors.New("authorization header doesn't follow bearer format")
		}
		// Get content
		content := sp[1]
		// Check if content exists
		if content != "" {
			return content, nil
		}
	}
	// Content is empty => Try to continue with cookie

	logEntry.Debug("Try get auth cookie from request")
	// Try to get auth cookie
	cookie, err := r.Cookie(cookieName)
	// Check if error exists
	if err != nil {
		logEntry.Debug("Can't load auth cookie")

		if err != http.ErrNoCookie {
			return "", err
		}
	}
	// Check if cookie value exists
	if cookie != nil {
		return cookie.Value, nil
	}

	return "", nil
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

// IsValidRedirect checks whether the redirect URL is whitelisted
func isValidRedirect(redirect string) bool {
	return strings.HasPrefix(redirect, "http://") || strings.HasPrefix(redirect, "https://")
}
