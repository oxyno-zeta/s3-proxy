package authorization

import (
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/thoas/go-funk"
)

func isOIDCAuthorizedBasic(groups []string, email string, authorizationAccesses []*config.OIDCAuthorizationAccess) bool {
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
