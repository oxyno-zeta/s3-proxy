package templateutils

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

// Copied from https://github.com/helm/helm/blob/a499b4b179307c267bdf3ec49b880e3dbd2a5591/pkg/engine/funcs.go

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toYAML(v any) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}

	return strings.TrimSuffix(string(data), "\n")
}

// toJSON takes an interface, marshals it to json, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}

	return string(data)
}
