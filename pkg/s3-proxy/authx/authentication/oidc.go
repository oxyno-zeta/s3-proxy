package authentication

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"

	oidc "github.com/coreos/go-oidc/v3/oidc"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	utils "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
)

const (
	redirectQueryKey       = "rd"
	stateRedirectSeparator = ":"
)

// OIDCEndpoints will set OpenID Connect endpoints for authentication and callback.
func (s *service) OIDCEndpoints(providerKey string, oidcCfg *config.OIDCAuthConfig, mux chi.Router) error {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, oidcCfg.IssuerURL)
	if err != nil {
		return errors.WithStack(err)
	}

	oidcConfig := &oidc.Config{
		ClientID: oidcCfg.ClientID,
	}
	verifier := provider.Verifier(oidcConfig)

	// Create main redirect url
	mainRedirectURLStr := ""
	// Create main redirect callback path
	mainRedirectURLCallbackPath := oidcCfg.CallbackPath
	// Check if redirect url is configured
	if oidcCfg.RedirectURL != "" {
		// Build redirect url
		mainRedirectURLObject, err := url.Parse(oidcCfg.RedirectURL)
		// Check if error exists
		if err != nil {
			return errors.WithStack(err)
		}
		// Continue to build redirect url
		mainRedirectURLObject.Path = path.Join(mainRedirectURLObject.Path, oidcCfg.CallbackPath)
		mainRedirectURLStr = mainRedirectURLObject.String()
		// Store path
		mainRedirectURLCallbackPath = mainRedirectURLObject.Path
	}

	// Keep the browser binding provider-specific and host-only. All replicas
	// validate it from the request; no shared transaction storage is required.
	stateCookieName := fmt.Sprintf("s3_proxy_oidc_state_%x", sha256.Sum256([]byte(providerKey)))
	if oidcCfg.CookieSecure {
		stateCookieName = "__Host-" + stateCookieName
	}

	// Store provider verifier in map
	s.allVerifiers[providerKey] = verifier

	mux.HandleFunc(oidcCfg.LoginPath, func(w http.ResponseWriter, r *http.Request) {
		// Parse query params from request
		qs := r.URL.Query()
		// Get redirect query from query params
		rdVal := qs.Get(redirectQueryKey)
		// Limit how long the callback can complete this login attempt.
		expires := time.Now().Add(oidcCfg.StateLifetime)
		// Give each login a cryptographically random nonce and carry its expiry
		// and redirect in state so any replica can process the callback.
		newState := rand.Text() + stateRedirectSeparator + strconv.FormatInt(expires.Unix(), 10) + stateRedirectSeparator + rdVal
		// Store a fixed-size hash in the browser cookie to bind all state fields
		// to this browser. Changing any field makes the callback check fail.
		stateHash := sha256.Sum256([]byte(newState))
		http.SetCookie(w, &http.Cookie{ //nolint:gosec
			Name:     stateCookieName,
			Value:    hex.EncodeToString(stateHash[:]),
			Path:     "/",
			Expires:  expires,
			MaxAge:   int(oidcCfg.StateLifetime.Seconds()),
			HttpOnly: true,
			Secure:   oidcCfg.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})

		// Create OIDC configuration
		config := generateOIDCConfig(
			oidcCfg,
			provider,
			mainRedirectURLStr,
			r,
		)

		// The redirect target is carried in the OAuth state and is validated
		// against the request host when the callback is handled.
		http.Redirect(w, r, config.AuthCodeURL(newState), http.StatusFound) //nolint:gosec // G710: rd is validated before it is used after the callback
	})

	mux.HandleFunc(mainRedirectURLCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		// Get context
		ctx := r.Context()
		// Get logger from request
		logEntry := log.GetLoggerFromContext(ctx)

		// ! In this particular case, no bucket request context because mounted in general and not per target

		// Get state from request
		reqQueryState := r.URL.Query().Get("state")
		// Check if state exists
		if reqQueryState == "" {
			// Create error
			err := errors.New("state not found in request")
			// Answer
			responsehandler.GeneralBadRequestError(r, w, s.cfgManager, err)

			return
		}

		// Require the binding cookie set in the browser that initiated login.
		stateCookie, err := r.Cookie(stateCookieName)
		// Check no cookie
		if errors.Is(err, http.ErrNoCookie) {
			responsehandler.GeneralBadRequestError(r, w, s.cfgManager, errors.New("state cookie not found in request"))

			return
		}
		// Check error
		if err != nil {
			responsehandler.GeneralBadRequestError(r, w, s.cfgManager, errors.Wrap(err, "failed to read state cookie"))

			return
		}

		// Hash the complete returned state to check that its nonce, expiry and
		// redirect match the browser's cookie without a replica-local lookup.
		stateHash := sha256.Sum256([]byte(reqQueryState))
		// Compare in constant time to avoid leaking matching hash prefixes.
		if subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(hex.EncodeToString(stateHash[:]))) != 1 {
			// Create error
			err = errors.New("state did not match")
			// Answer
			responsehandler.GeneralBadRequestError(r, w, s.cfgManager, err)

			return
		}

		// Consume the browser binding before exchanging the single-use code.
		http.SetCookie(w, &http.Cookie{ //nolint:gosec
			Name:     stateCookieName,
			Path:     "/",
			Expires:  time.Unix(1, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   oidcCfg.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})

		// Enforce expiry on the server as well as in the browser. The cookie
		// hash above binds the timestamp and redirect to the original login.
		split := strings.SplitN(reqQueryState, stateRedirectSeparator, 3) //nolint:mnd
		if len(split) != 3 {                                              //nolint:mnd
			responsehandler.GeneralBadRequestError(r, w, s.cfgManager, errors.New("invalid state format"))

			return
		}

		expires, err := strconv.ParseInt(split[1], 10, 64)
		// Check error or if state is expired
		if err != nil || !time.Now().Before(time.Unix(expires, 0)) {
			responsehandler.GeneralBadRequestError(r, w, s.cfgManager, errors.New("state expired or invalid"))

			return
		}

		// Get redirect value
		rdVal := split[2]

		// Check if rdVal exists and that redirect url value is valid
		if rdVal != "" {
			isValid, err2 := isValidRedirect(rdVal, utils.GetRequestURI(r))
			// Check error
			if err2 != nil {
				// Answer
				responsehandler.GeneralInternalServerError(r, w, s.cfgManager, err2)

				return
			}
			// Check if it is invalid
			if !isValid {
				// Create error
				err2 = errors.New("redirect url is invalid")
				// Answer
				responsehandler.GeneralBadRequestError(r, w, s.cfgManager, err2)

				return
			}
		}

		// Create OIDC configuration
		config := generateOIDCConfig(
			oidcCfg,
			provider,
			mainRedirectURLStr,
			r,
		)

		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			// Create error
			err = errors.New("failed to exchange token: " + err.Error())
			// Answer
			responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, err)

			return
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			// Create error
			err = errors.New("no id_token field in token")
			// Answer
			responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, err)

			return
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			// Create error
			err = errors.New("failed to verify ID Token: " + err.Error())
			// Answer
			responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, err)

			return
		}

		var resp map[string]any

		// Try to open JWT token in order to verify that we can open it
		err = idToken.Claims(&resp)
		if err != nil {
			// Answer
			responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, errors.WithStack(err))

			return
		}

		// Initialize cookie domain
		cookieDomain := ""
		// Get request host of current request
		reqHost := utils.GetRequestHost(r)
		// Loop over domains
		for _, it := range oidcCfg.CookieDomains {
			// Check if domain asked is matching the one in the request host
			if strings.HasSuffix(reqHost, it) {
				// Save cookie domain
				cookieDomain = it
				// Break the loop
				break
			}
		}

		// Build cookie
		cookie := &http.Cookie{ //nolint:gosec // G124: Secure is intentionally configurable; HttpOnly and SameSite are always enabled
			Expires:  idToken.Expiry,
			Name:     oidcCfg.CookieName,
			Value:    rawIDToken,
			HttpOnly: true,
			Secure:   oidcCfg.CookieSecure,
			Path:     "/",
			Domain:   cookieDomain,
		}
		http.SetCookie(w, cookie)

		// Keep the validated redirect separate from the request-tainted state value.
		redirectURL := rdVal
		// Manage default redirect case
		if rdVal == "" {
			redirectURL = "/"
		}

		logEntry.Info("Successful authentication detected")
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect) //nolint:gosec // G710: redirectURL was validated against the request host above
	})

	return nil
}

func (s *service) oidcAuthMiddleware(res *config.Resource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get oidc configuration
			oidcAuthCfg := s.cfg.AuthProviders.OIDC[res.Provider]
			// Get context
			ctx := r.Context()
			// Get logger from request
			logEntry := log.GetLoggerFromContext(ctx)
			// Get bucket request context from request
			brctx := bucket.GetBucketRequestContextFromContext(ctx)
			// Get response handler
			resHan := responsehandler.GetResponseHandlerFromContext(ctx)

			// Get JWT Token from header or cookie
			jwtContent, err := getJWTToken(logEntry, r, oidcAuthCfg.CookieName)
			// Check if error exists
			if err != nil {
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralInternalServerError(r, w, s.cfgManager, err)
				} else {
					resHan.InternalServerError(brctx.LoadFileContent, err)
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
			claims, err := parseAndValidateJWTToken(ctx, jwtContent, s.allVerifiers[res.Provider])
			if err != nil {
				logEntry.Error(err)
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralInternalServerError(r, w, s.cfgManager, err)
				} else {
					resHan.InternalServerError(brctx.LoadFileContent, err)
				}

				return
			}

			// Create OIDC user
			ouser := &models.OIDCUser{}

			// Initialize email
			email := ""

			// Get email claim
			emailClaim := claims["email"]

			if emailClaim != nil {
				// Get email
				email, _ = emailClaim.(string)

				// Check email verified
				if oidcAuthCfg.EmailVerified && claims["email_verified"] != nil {
					// Get email verified field
					emailVerified, _ := claims["email_verified"].(bool)
					// Update email verified in user
					ouser.EmailVerified = emailVerified
				}
			}

			// Update user
			ouser.Email = email

			// Get groups
			groupsInterface := claims[oidcAuthCfg.GroupClaim]
			groups := make([]string, 0)
			// Check if groups interface exists
			if groupsInterface != nil {
				//nolint: forcetypeassert // Ignore this
				for _, item := range groupsInterface.([]any) {
					//nolint: forcetypeassert // Ignore this
					groups = append(groups, item.(string))
				}
			}

			// Update user
			ouser.Groups = groups

			// Finishing building user
			if claims["family_name"] != nil {
				ouser.FamilyName, _ = claims["family_name"].(string)
			}

			if claims["given_name"] != nil {
				ouser.GivenName, _ = claims["given_name"].(string)
			}

			if claims["name"] != nil {
				ouser.Name, _ = claims["name"].(string)
			}

			if claims["preferred_username"] != nil {
				ouser.PreferredUsername, _ = claims["preferred_username"].(string)
			}

			// Manage OIDC UID
			uid, err := buildOIDCUID(oidcAuthCfg, claims, ouser.PreferredUsername, ouser.Email)
			if err != nil {
				// Manage error
				if brctx == nil {
					responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, err)
				} else {
					resHan.UnauthorizedError(brctx.LoadFileContent, err)
				}

				return
			}
			// Save UID
			ouser.UID = uid

			// Add user to request context by creating a new context
			ctx = models.SetAuthenticatedUserInContext(ctx, ouser)
			// Create new request with new context
			r = r.WithContext(ctx)

			// Update response handler to have the latest context values
			resHan.UpdateRequestAndResponse(r, w)

			// Check if email is verified or not
			if oidcAuthCfg.EmailVerified && !ouser.EmailVerified {
				// Create error
				err := fmt.Errorf("email not verified for user %s", ouser.GetIdentifier())
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralForbiddenError(r, w, s.cfgManager, err)
				} else {
					resHan.ForbiddenError(brctx.LoadFileContent, err)
				}

				return
			}

			logEntry.Infof("OIDC User authenticated: %s", ouser.GetIdentifier())
			s.metricsCl.IncAuthenticated("oidc", res.Provider)

			// Next
			next.ServeHTTP(w, r)
		})
	}
}

func parseAndValidateJWTToken(
	ctx context.Context,
	jwtContent string,
	verifier *oidc.IDTokenVerifier,
) (map[string]any, error) {
	// Create result map
	var res map[string]any

	// Verify token
	idToken, err := verifier.Verify(ctx, jwtContent)
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

func buildOIDCUID(
	oidcAuthCfg *config.OIDCAuthConfig,
	claims map[string]any,
	preferredUsername string,
	email string,
) (string, error) {
	var uid string

	switch {
	case oidcAuthCfg.UIDClaim != "":
		uid, _ = claims[oidcAuthCfg.UIDClaim].(string)
	case preferredUsername != "":
		uid = preferredUsername
	default:
		uid = email
	}

	// Reject unusable identities before they can enter authenticated context.
	if uid == "" {
		return "", errors.New("oidc user identifier is missing or empty")
	}
	// Reject padding instead of removing it: normalization could turn a distinct
	// identity into another user's S3 prefix or a configured isolation admin.
	if uid != strings.TrimSpace(uid) {
		return "", errors.New("oidc user identifier must not have leading or trailing whitespace")
	}

	return uid, nil
}

func getJWTToken(logEntry log.Logger, r *http.Request, cookieName string) (string, error) {
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

		if !errors.Is(err, http.ErrNoCookie) {
			return "", err
		}
	}
	// Check if cookie value exists
	if cookie != nil {
		return cookie.Value, nil
	}

	return "", nil
}

func generateOIDCConfig(
	oidcCfg *config.OIDCAuthConfig,
	provider *oidc.Provider,
	mainRedirectURL string,
	req *http.Request,
) *oauth2.Config {
	// Create oidc configuration
	cfg := oauth2.Config{
		ClientID: oidcCfg.ClientID,
		Endpoint: provider.Endpoint(),
		Scopes:   oidcCfg.Scopes,
	}
	// Check if client secret is set
	if oidcCfg.ClientSecret != nil {
		cfg.ClientSecret = oidcCfg.ClientSecret.Value
	}

	// Check if main redirect url is defined
	// Otherwise build a dynamic one
	if mainRedirectURL != "" {
		cfg.RedirectURL = mainRedirectURL
	} else {
		cfg.RedirectURL = fmt.Sprintf(
			"%s://%s%s",
			utils.GetRequestScheme(req),
			utils.GetRequestHost(req),
			oidcCfg.CallbackPath,
		)
	}

	// Return
	return &cfg
}

// IsValidRedirect checks whether the redirect URL is whitelisted.
func isValidRedirect(redirectURLStr, reqURLStr string) (bool, error) {
	// Check if it isn't forged with complete urls
	if !strings.HasPrefix(redirectURLStr, "http://") && !strings.HasPrefix(redirectURLStr, "https://") {
		return false, nil
	}

	// Parse request URL
	currentURL, err := url.Parse(reqURLStr)
	// Check error
	if err != nil {
		return false, errors.WithStack(err)
	}
	// Parse redirect URL
	redURL, err := url.Parse(redirectURLStr)
	// Check error
	if err != nil {
		return false, errors.WithStack(err)
	}

	// Check if hosts aren't the same
	if redURL.Host != currentURL.Host {
		return false, nil
	}

	// Default
	return true, nil
}
