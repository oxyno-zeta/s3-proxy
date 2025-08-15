package authorization

import (
	"github.com/thoas/go-funk"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

func isHeaderOIDCAuthorizedBasic(groups []string, email string, authorizationAccesses []*config.HeaderOIDCAuthorizationAccess) bool {
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
						return !item.Forbidden
					}
				}
			}

			// Check email case
			if item.Email != "" && item.EmailRegexp.MatchString(email) {
				return !item.Forbidden
			}
		} else {
			// Not a regex case
			// Check group case
			if item.Group != "" {
				result := funk.Contains(groups, item.Group)
				if result {
					return !item.Forbidden
				}
			}
			// Check email case
			if item.Email != "" && item.Email == email {
				return !item.Forbidden
			}
		}
	}

	// Not found case
	return false
}
