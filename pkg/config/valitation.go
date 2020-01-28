package config

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/thoas/go-funk"
)

func validateBusinessConfig(out *Config) error {
	// Validate resources if they exists in all targets, validate target mount path and validate actions
	for i := 0; i < len(out.Targets); i++ {
		target := out.Targets[i]
		// Check if resources are declared
		if target.Resources != nil {
			for j := 0; j < len(target.Resources); j++ {
				res := target.Resources[j]
				// Validate resource
				err := validateResource(fmt.Sprintf("resource %d from target %d", j, i), res, out.AuthProviders, target.Mount.Path)
				// Return error if exists
				if err != nil {
					return err
				}
			}
		}
		// Check mount path items
		pathList := target.Mount.Path
		for j := 0; j < len(pathList); j++ {
			path := pathList[j]
			// Check path value
			err := validatePath(fmt.Sprintf("path %d in target %d", j, i), path)
			if err != nil {
				return err
			}
		}
		// Check actions
		if target.Actions.GET == nil && target.Actions.PUT == nil && target.Actions.DELETE == nil {
			return fmt.Errorf("at least one action must be declared in target %d", i)
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
			return fmt.Errorf("at least one action must be enabled in target %d", i)
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
			path := pathList[j]
			// Check path value
			err := validatePath(fmt.Sprintf("path %d in list targets", j), path)
			if err != nil {
				return err
			}
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
	if res.WhiteList == nil && res.Basic == nil && res.OIDC == nil {
		return errors.New(beginErrorMessage + " must have whitelist, basic configuration or oidc configuration")
	}
	// Check if provider exists
	if res.WhiteList != nil && !*res.WhiteList && res.Provider == "" {
		return errors.New(beginErrorMessage + " must have a provider")
	}
	// Check auth logins are provided in case of no whitelist
	if res.WhiteList != nil && !*res.WhiteList && res.Basic == nil && res.OIDC == nil {
		return errors.New(beginErrorMessage + " must have authentication configuration declared (oidc or basic)")
	}
	// Check that provider is declared is auth providers and correctly linked
	if res.Provider != "" {
		// Check that auth provider exists
		exists := authProviders.Basic[res.Provider] != nil || authProviders.OIDC[res.Provider] != nil
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

func validatePath(beginErrorMessage string, path string) error {
	// Check that path begins with /
	if !strings.HasPrefix(path, "/") {
		return errors.New(beginErrorMessage + " must starts with /")
	}
	// Check that path ends with /
	if !strings.HasSuffix(path, "/") {
		return errors.New(beginErrorMessage + " must ends with /")
	}
	// Return no error
	return nil
}
