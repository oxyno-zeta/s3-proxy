package templateutils

import (
	"bytes"
	"context"
	"os"
	"strings"
	"text/template"

	"emperror.dev/errors"
	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
)

const recursionMaxNums = 1000

func LoadAllHelpersContent(
	ctx context.Context,
	loadS3FileContent func(ctx context.Context, path string) (string, error),
	items []*config.TargetHelperConfigItem,
	pathList []string,
) (string, error) {
	// Initialize template content
	tplContent := ""

	// Check if there is a list of config items
	if len(items) != 0 {
		// Loop over items
		for _, item := range items {
			// Load template content
			tpl, err := loadHelperContent(
				ctx,
				loadS3FileContent,
				item,
			)
			// Check error
			if err != nil {
				return "", err
			}
			// Concat
			tplContent = tplContent + "\n" + tpl
		}
	} else {
		// Load from local files
		// Loop over local path
		for _, item := range pathList {
			// Load template content
			tpl, err := LoadLocalFileContent(item)
			// Check error
			if err != nil {
				return "", err
			}
			// Concat
			tplContent = tplContent + "\n" + tpl
		}
	}

	// Return
	return tplContent, nil
}

func loadHelperContent(
	ctx context.Context,
	loadS3FileContent func(ctx context.Context, path string) (string, error),
	item *config.TargetHelperConfigItem,
) (string, error) {
	// Check if it is in bucket and if load from S3 function exists
	if item.InBucket && loadS3FileContent != nil {
		// Try to get file from bucket
		return loadS3FileContent(ctx, item.Path)
	}

	// Not in bucket, need to load from FS
	return LoadLocalFileContent(item.Path)
}

func LoadTemplateContent(
	ctx context.Context,
	loadS3FileContent func(ctx context.Context, path string) (string, error),
	item *config.TargetTemplateConfigItem,
) (string, error) {
	// Check if it is in bucket and if load from S3 function exists
	if item.InBucket && loadS3FileContent != nil {
		// Try to get file from bucket
		return loadS3FileContent(ctx, item.Path)
	}

	// Not in bucket, need to load from FS
	return LoadLocalFileContent(item.Path)
}

func LoadLocalFileContent(path string) (string, error) {
	// Read file from file path
	by, err := os.ReadFile(path)
	// Check if error exists
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(by), nil
}

func ExecuteTemplate(tplString string, data interface{}) (*bytes.Buffer, error) {
	// Create template
	tmpl := template.New("template-string-loaded")

	// Load template from string
	tmpl, err := tmpl.
		Funcs(sprig.TxtFuncMap()).
		Funcs(s3ProxyFuncMap(tmpl)).
		Parse(tplString)
	// Check if error exists
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Generate template in buffer
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, data)
	// Check if error exists
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return buf, nil
}

func s3ProxyFuncMap(t *template.Template) template.FuncMap {
	// Initialize includedNames
	// That will help detect nested loop references
	includedNames := make(map[string]int)

	// Result
	funcMap := map[string]interface{}{}

	// Add human size function
	funcMap["humanSize"] = func(fmt int64) string {
		return humanize.Bytes(uint64(fmt))
	}
	// Add request URI function
	funcMap["requestURI"] = generalutils.GetRequestURI
	// Add request scheme function
	funcMap["requestScheme"] = generalutils.GetRequestScheme
	// Add request host function
	funcMap["requestHost"] = generalutils.GetRequestHost
	// Add the 'include' function here so we can close over t.
	// Copied from Helm: https://github.com/helm/helm/blob/3d1bc72827e4edef273fb3d8d8ded2a25fa6f39d/pkg/engine/engine.go#L112
	funcMap["include"] = func(name string, data interface{}) (string, error) {
		// Initialize buffer
		var buf strings.Builder

		if v, ok := includedNames[name]; ok {
			if v > recursionMaxNums {
				return "", errors.Wrapf(errors.New("unable to execute template"), "rendering template has a nested reference name: %s", name)
			}

			includedNames[name]++
		} else {
			includedNames[name] = 1
		}

		err := t.ExecuteTemplate(&buf, name, data)
		includedNames[name]--

		return buf.String(), err
	}
	// Copied from Helm
	funcMap["toYaml"] = toYAML
	funcMap["toJson"] = toJSON
	// Inspired from helm
	funcMap["tpl"] = func(tpl string, data interface{}) (string, error) {
		// Execute template
		buf, err := ExecuteTemplate(tpl, data)
		// Check error
		if err != nil {
			return "", err
		}

		// Default
		return buf.String(), nil
	}

	// Return result
	return template.FuncMap(funcMap)
}
