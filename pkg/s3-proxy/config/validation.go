package config

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"emperror.dev/errors"
	utils "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"

	"github.com/thoas/go-funk"
)

func validateBusinessConfig(out *Config) error {
	// Validate resources if they exists in all targets, validate target mount path and validate actions
	for key, target := range out.Targets {
		// Check if resources are declared
		if target.Resources != nil {
			for j := 0; j < len(target.Resources); j++ {
				res := target.Resources[j]
				// Validate resource
				err := validateResource(fmt.Sprintf("resource %d from target %s", j, key), res, out.AuthProviders, target.Mount.Path)
				// Return error if exists
				if err != nil {
					return err
				}
			}
		}
		// Check mount path items
		pathList := target.Mount.Path
		for j := 0; j < len(pathList); j++ {
			p := pathList[j]
			// Check path value
			err := validatePath(fmt.Sprintf("path %d in target %s", j, key), p)
			if err != nil {
				return err
			}
		}
		// Check actions
		if target.Actions.GET == nil && target.Actions.PUT == nil && target.Actions.DELETE == nil {
			return errors.Errorf("at least one action must be declared in target %s", key)
		}
		// This part will check that at least one action is enabled
		oneMustBeEnabled := false

		if target.Actions.GET != nil {
			oneMustBeEnabled = target.Actions.GET.Enabled || oneMustBeEnabled
		}

		if target.Actions.PUT != nil {
			oneMustBeEnabled = target.Actions.PUT.Enabled || oneMustBeEnabled
		}

		if target.Actions.DELETE != nil {
			oneMustBeEnabled = target.Actions.DELETE.Enabled || oneMustBeEnabled
		}

		if !oneMustBeEnabled {
			return errors.Errorf("at least one action must be enabled in target %s", key)
		}
	}

	// Validate list targets object
	if out.ListTargets != nil && out.ListTargets.Enabled {
		// Check list targets resource
		if out.ListTargets.Resource != nil {
			res := out.ListTargets.Resource
			// Validate resource
			err := validateResource("resource from list targets", res, out.AuthProviders, out.ListTargets.Mount.Path)
			// Return error if exists
			if err != nil {
				return err
			}
		}
		// Check mount path items
		pathList := out.ListTargets.Mount.Path
		for j := 0; j < len(pathList); j++ {
			p := pathList[j]
			// Check path value
			err := validatePath(fmt.Sprintf("path %d in list targets", j), p)
			if err != nil {
				return err
			}
		}
	}

	// Validate authentication providers
	if out.AuthProviders != nil && out.AuthProviders.OIDC != nil {
		for prov, authProviderCfg := range out.AuthProviders.OIDC {
			// Check that state doesn't contain ":"
			if strings.Contains(authProviderCfg.State, ":") {
				return errors.Errorf("provider %s state can't contain ':' character", prov)
			}

			// Build redirect url
			u, err := url.Parse(authProviderCfg.RedirectURL)
			// Check if error exists
			if err != nil {
				return errors.WithStack(err)
			}
			// Continue to build redirect url
			u.Path = path.Join(u.Path, authProviderCfg.CallbackPath)

			// Now, full callback path is generated

			// Check if new path is "/"
			if u.Path == "/" {
				return errors.Errorf("provider %s can't have a callback path equal to / (to avoid redirect loop)", prov)
			}

			// Check login path
			if authProviderCfg.LoginPath == "/" {
				return errors.Errorf("provider %s can't have a login path equal to / (to avoid redirect loop)", prov)
			}

			// Check that they are different
			if authProviderCfg.LoginPath == u.Path {
				return errors.Errorf("provider %s can't have same login and callback path (to avoid redirect loop)", prov)
			}
		}
	}

	if out.Server != nil && out.Server.SSL != nil {
		err := validateSSLConfig(out.Server.SSL, "server")
		if err != nil {
			return err
		}
	}

	if out.InternalServer != nil && out.InternalServer.SSL != nil {
		err := validateSSLConfig(out.InternalServer.SSL, "internalServer")
		if err != nil {
			return err
		}
	}

	return nil
}

func validateResource(beginErrorMessage string, res *Resource, authProviders *AuthProviderConfig, mountPathList []string) error {
	// Check resource http methods
	// Filter http methods that are not supported
	filtered := funk.FilterString(res.Methods, func(s string) bool {
		return s != http.MethodGet && s != http.MethodPut && s != http.MethodDelete
	})
	// Check if size is > 0
	if len(filtered) > 0 {
		return errors.New(beginErrorMessage + " must have a HTTP method in GET, PUT or DELETE")
	}
	// Check resource not valid
	if res.WhiteList == nil && res.Basic == nil && res.OIDC == nil && res.Header == nil {
		return errors.New(beginErrorMessage + " must have whitelist, basic, header or oidc configuration")
	}
	// Check if provider exists
	if res.WhiteList != nil && !*res.WhiteList && res.Provider == "" {
		return errors.New(beginErrorMessage + " must have a provider")
	}
	// Check auth logins are provided in case of no whitelist
	if res.WhiteList != nil && !*res.WhiteList && res.Basic == nil && res.OIDC == nil && res.Header == nil {
		return errors.New(beginErrorMessage + " must have authentication configuration declared (oidc, header or basic)")
	}
	// Check that provider is declared is auth providers and correctly linked
	if res.Provider != "" {
		// Check if auth providers exists
		if authProviders == nil {
			return errors.New(beginErrorMessage + " has declared a provider but authentication providers aren't declared")
		}
		// Check that auth provider exists for target provider
		exists := (authProviders.Basic != nil && authProviders.Basic[res.Provider] != nil) ||
			(authProviders.OIDC != nil && authProviders.OIDC[res.Provider] != nil) ||
			(authProviders.Header != nil && authProviders.Header[res.Provider] != nil)
		if !exists {
			return errors.New(beginErrorMessage + " must have a valid provider declared in authentication providers")
		}
		// Check that selected provider is in link with authentication selected
		// Check basic
		if res.Basic != nil && authProviders.Basic[res.Provider] == nil {
			return errors.New(
				beginErrorMessage + " must use a valid authentication configuration with selected authentication provider: basic auth not allowed")
		}
		// Check oidc
		if res.OIDC != nil && authProviders.OIDC[res.Provider] == nil {
			return errors.New(beginErrorMessage + " must use a valid authentication configuration with selected authentication provider: oidc not allowed")
		}
		// Check that oidc authorization is valid
		if res.OIDC != nil && res.OIDC.AuthorizationOPAServer != nil && len(res.OIDC.AuthorizationAccesses) != 0 {
			return errors.New(beginErrorMessage + " cannot contain oidc authorization accesses and OPA server together at the same time")
		}
		// Check header
		if res.Header != nil && authProviders.Header[res.Provider] == nil {
			return errors.New(beginErrorMessage + " must use a valid authentication configuration with selected authentication provider: header not allowed")
		}
		// Check that header authorization is valid
		if res.Header != nil && res.Header.AuthorizationOPAServer != nil && len(res.Header.AuthorizationAccesses) != 0 {
			return errors.New(beginErrorMessage + " cannot contain header authorization accesses and OPA server together at the same time")
		}
	}
	// Check if resource path contains mount path item
	pathMatch := false
	// Loop over mount path list
	for i := 0; i < len(mountPathList); i++ {
		mountPath := mountPathList[i]
		// Check
		if strings.HasPrefix(res.Path, mountPath) {
			pathMatch = true
			// Stop loop now
			break
		}
	}
	// Check if matching was found
	if !pathMatch {
		return errors.New(beginErrorMessage + " must start with path declared in mount path section")
	}

	// Return no error
	return nil
}

func validatePath(beginErrorMessage string, fpath string) error {
	// Check that path begins with /
	if !strings.HasPrefix(fpath, "/") {
		return errors.New(beginErrorMessage + " must starts with /")
	}
	// Check that path ends with /
	if !strings.HasSuffix(fpath, "/") {
		return errors.New(beginErrorMessage + " must ends with /")
	}
	// Return no error
	return nil
}

func validateSSLConfig(serverSSL *ServerSSLConfig, section string) error {
	if serverSSL.Enabled {
		if len(serverSSL.Certificates) == 0 && len(serverSSL.SelfSignedHostnames) == 0 {
			return errors.Errorf("at least one of %s.ssl.certificates or %s.ssl.selfSignedHostnames must have values", section, section)
		}
	}

	if serverSSL.MinTLSVersion != nil && utils.ParseTLSVersion(*serverSSL.MinTLSVersion) == 0 {
		return errors.Errorf(
			"%s.ssl.minTLSVersion %#v must be a valid TLS version: expected \"TLSv1.0\", \"TLSv1.1\", \"TLSv1.2\", or \"TLSv1.3\"",
			section, *serverSSL.MinTLSVersion)
	}

	if serverSSL.MaxTLSVersion != nil && utils.ParseTLSVersion(*serverSSL.MaxTLSVersion) == 0 {
		return errors.Errorf(
			"%s.ssl.maxTLSVersion %#v must be a valid TLS version: expected \"TLSv1.0\", \"TLSv1.1\", \"TLSv1.2\", or \"TLSv1.3\"",
			section, *serverSSL.MaxTLSVersion)
	}

	for _, cipherSuiteName := range serverSSL.CipherSuites {
		if utils.ParseCipherSuite(cipherSuiteName) == 0 {
			var cipherSuiteNames []string

			for _, cipherSuite := range tls.CipherSuites() {
				cipherSuiteNames = append(cipherSuiteNames, fmt.Sprintf(`"%s"`, cipherSuite.Name))
			}

			return errors.Errorf(
				"invalid cipher suite %#v in %s.ssl.cipherSuites; expected one of %s", cipherSuiteName, section,
				strings.Join(cipherSuiteNames, ", "))
		}
	}

	for i, cert := range serverSSL.Certificates {
		err := validateSSLCertificateComponentConfig(
			cert.Certificate, cert.CertificateURL, cert.CertificateURLConfig,
			fmt.Sprintf("%s.ssl.certificates[%d].certificate", section, i))

		if err != nil {
			return err
		}

		err = validateSSLCertificateComponentConfig(
			cert.PrivateKey, cert.PrivateKeyURL, cert.PrivateKeyURLConfig,
			fmt.Sprintf("%s.ssl.certificates[%d].privateKey", section, i))
		if err != nil {
			return err
		}
	}

	return nil
}

func validateSSLCertificateComponentConfig(
	component *string, componentURL *string, componentURLConfig *SSLURLConfig, componentName string) error {
	if component == nil {
		if componentURL == nil {
			return errors.Errorf("either %s or %sUrl must be set", componentName, componentName)
		}

		err := utils.ValidateDocumentURL(*componentURL)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("%sUrl is a malformed/unsupported URL: %s", componentName, *componentURL))
		}

		if componentURLConfig != nil {
			err = validateSSLURLConfig(componentURLConfig, fmt.Sprintf("%sUrlConfig", componentName))
			if err != nil {
				return err
			}
		}
	} else if componentURL != nil {
		return errors.Errorf("%s and %sUrl cannot both be set", componentName, componentName)
	}

	return nil
}

func validateSSLURLConfig(urlConfig *SSLURLConfig, component string) error {
	if urlConfig.HTTPTimeout != "" {
		duration, err := time.ParseDuration(urlConfig.HTTPTimeout)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("%s.httpTimeout is invalid", component))
		}

		if duration < time.Duration(0) {
			return errors.Errorf("%s.httpTimeout cannot be negative", component)
		}
	}

	if urlConfig.AWSEndpoint != "" {
		if urlConfig.AWSRegion == "" {
			return errors.Errorf("%s.awsRegion must be set when %s.awsEndpoint is set", component, component)
		}
	}

	return nil
}
