package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func loadCredential(credCfg *CredentialConfig) error {
	if credCfg.Path != "" {
		// Secret file
		databytes, err := ioutil.ReadFile(credCfg.Path)
		if err != nil {
			return err
		}
		// Store value
		credCfg.Value = string(databytes)
	} else if credCfg.Env != "" {
		// Environment variable
		envValue := os.Getenv(credCfg.Env)
		if envValue == "" {
			return fmt.Errorf(TemplateErrLoadingEnvCredentialEmpty, credCfg.Env)
		}
		// Store value
		credCfg.Value = envValue
	}
	// Value case is already managed by koanf
	return nil
}

func validateResource(beginErrorMessage string, res *Resource, authProviders *AuthProviderConfig, mountPathList []string) error {
	// Check resource not valid
	if res.WhiteList == nil && res.Basic == nil && res.OIDC == nil {
		return errors.New(beginErrorMessage + " have whitelist, basic configuration or oidc configuration")
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
