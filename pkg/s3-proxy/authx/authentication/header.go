package authentication

import (
	"net/http"
	"strings"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	"github.com/pkg/errors"
)

func (s *service) headerAuthMiddleware(res *config.Resource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get header configuration
			headerAuthCfg := s.cfg.AuthProviders.Header[res.Provider]
			// Get logger from request
			logEntry := log.GetLoggerFromContext(r.Context())
			// Get bucket request context from request
			brctx := bucket.GetBucketRequestContextFromContext(r.Context())
			// Get response handler
			resHan := responsehandler.GetResponseHandlerFromContext(r.Context())

			// Get Email header value
			emailHValue := r.Header.Get(headerAuthCfg.EmailHeader)
			// Get Username header value
			usernameHValue := r.Header.Get(headerAuthCfg.UsernameHeader)
			// Get groups header value
			groupsHValue := r.Header.Get(headerAuthCfg.GroupsHeader)

			// Check if email or username aren't set
			if emailHValue == "" || usernameHValue == "" {
				// Initialize error
				var err error
				// Switch
				switch {
				case emailHValue == "":
					err = errors.New("cannot find email value from header")
				case usernameHValue == "":
					err = errors.New("cannot find username value from header")
				}

				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralInternalServerError(r, w, s.cfgManager, err)
				} else {
					resHan.InternalServerError(brctx.LoadFileContent, err)
				}

				return
			}

			// Initialize groups
			var groups []string
			// Check if groups is set
			if groupsHValue != "" {
				groups = strings.Split(groupsHValue, ",")
			}

			// Create Header auth user
			huser := &models.HeaderUser{
				Username: usernameHValue,
				Email:    emailHValue,
				Groups:   groups,
			}

			// Add user to request context by creating a new context
			ctx := models.SetAuthenticatedUserInContext(r.Context(), huser)
			// Create new request with new context
			r = r.WithContext(ctx)

			// Update response handler to have the latest context values
			resHan.UpdateRequestAndResponse(r, w)

			logEntry.Infof("Header auth user %s authenticated", huser.GetIdentifier())
			s.metricsCl.IncAuthenticated("header-auth", res.Provider)

			next.ServeHTTP(w, r)
		})
	}
}
