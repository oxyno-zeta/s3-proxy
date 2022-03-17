package templateutils

import (
	"bytes"
	"context"
	"io/ioutil"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
	"github.com/pkg/errors"
)

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
	by, err := ioutil.ReadFile(path)
	// Check if error exists
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(by), nil
}

func ExecuteTemplate(tplString string, data interface{}) (*bytes.Buffer, error) {
	// Load template from string
	tmpl, err := template.
		New("template-string-loaded").
		Funcs(sprig.TxtFuncMap()).
		Funcs(s3ProxyFuncMap()).
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

func s3ProxyFuncMap() template.FuncMap {
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

	// Return result
	return template.FuncMap(funcMap)
}
