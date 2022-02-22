package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

const (
	schemeARN   string = "arn"
	schemeFile  string = "file"
	schemeHTTP  string = "http"
	schemeHTTPS string = "https"
	schemeS3    string = "s3"

	serviceS3             string = "s3"
	serviceSecretsManager string = "secretsmanager"
	serviceSSM            string = "ssm"
)

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
	funcMap["requestURI"] = GetRequestURI
	// Add request scheme function
	funcMap["requestScheme"] = GetRequestScheme
	// Add request host function
	funcMap["requestHost"] = GetRequestHost

	// Return result
	return template.FuncMap(funcMap)
}

// ClientIP will return client ip from request.
func ClientIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}

	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}

	return IPAddress
}

func GetRequestScheme(r *http.Request) string {
	// Get forwarded scheme
	fwdScheme := r.Header.Get("X-Forwarded-Proto")
	// Check if it is https
	if r.TLS != nil || fwdScheme == "https" {
		return "https"
	}

	// RFC 7239
	forwardedH := r.Header.Get("Forwarded")
	proto, _ := parseForwarded(forwardedH)
	// Check if protocol have been found
	if proto != "" {
		return proto
	}

	// Default
	return "http"
}

func GetRequestURI(r *http.Request) string {
	scheme := GetRequestScheme(r)

	return fmt.Sprintf("%s://%s%s", scheme, GetRequestHost(r), r.URL.RequestURI())
}

func GetRequestHost(r *http.Request) string {
	// not standard, but most popular
	host := r.Header.Get("X-Forwarded-Host")
	if host != "" {
		return host
	}

	// RFC 7239
	forwardedH := r.Header.Get("Forwarded")
	_, host = parseForwarded(forwardedH)

	if host != "" {
		return host
	}

	// if all else fails fall back to request host
	host = r.Host

	return host
}

func parseForwarded(forwarded string) (proto, host string) {
	if forwarded == "" {
		return
	}

	for _, forwardedPair := range strings.Split(forwarded, ";") {
		if tv := strings.SplitN(forwardedPair, "=", 2); len(tv) == 2 { // nolint: gomnd // No constant for that
			token, value := tv[0], tv[1]
			token = strings.TrimSpace(token)
			value = strings.TrimSpace(strings.Trim(value, `"`))

			switch strings.ToLower(token) {
			case "proto":
				proto = value
			case "host":
				host = value
			}
		}
	}

	return
}

// ParseCipherSuite parses a cipher suite name into the tls package cipher suite id.
//
// If the name is not recognized, 0 is returned.
func ParseCipherSuite(suiteName string) uint16 {
	for _, suite := range tls.CipherSuites() {
		if suite.Name == suiteName {
			return suite.ID
		}
	}

	return 0
}

// ParseTLSVersion parses the TLS version number from a string. This accepts raw version numbers
// "1.0", "1.1", "1.2", "1.3". If the string is prefixed with "TLS ", "TLSv", "TLS-", or "TLS_"
// (case-insensitive), that prefix is removed. The decimal separator ('.') can be replaced with
// either a '_' or a '-'.
//
// For example: "TLSv1.2", "TLS_1-2", "tls-1_2", "TLs 1-2", etc., are equivalent and return
// tls.VersionTLS12.
//
// If the version number cannot be parsed, 0 is returned.
func ParseTLSVersion(tlsVersionString string) uint16 {
	tlsVersionString = strings.ToLower(tlsVersionString)

	if strings.HasPrefix(tlsVersionString, "tlsv") {
		tlsVersionString = tlsVersionString[4:]
	} else if strings.HasPrefix(tlsVersionString, "tls") {
		tlsVersionString = tlsVersionString[3:]

		if len(tlsVersionString) == 0 {
			return 0
		}

		// Remove a dash, underscore, or space.
		if tlsVersionString[0] == '-' || tlsVersionString[0] == '_' || tlsVersionString[0] == ' ' {
			tlsVersionString = tlsVersionString[1:]
		}
	}

	tlsVersionString = strings.ReplaceAll(tlsVersionString, "_", ".")
	tlsVersionString = strings.ReplaceAll(tlsVersionString, "-", ".")

	switch tlsVersionString {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	}

	return 0
}

// GetDocumentFromURL retrieves a textual document from a URL, which may be an AWS ARN for an S3 object,
// Secrets Manager secret, or  Systems Manager parameter (arn:...); an HTTP or HTTPS URL; an S3 URL in
// the form s3://bucket/key; or a file in URL or regular path form.
func GetDocumentFromURL(rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	switch parsedURL.Scheme {
	case schemeARN:
		return getDocumentFromARN(rawURL)
	case schemeFile, "":
		return getDocumentFromFile(parsedURL.Path)
	case schemeHTTP, schemeHTTPS:
		return getDocumentFromHTTP(rawURL)
	case schemeS3:
		return getDocumentFromS3(parsedURL.Host, parsedURL.Path)
	}

	return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
}

// getDocumentFroARN retrieves a textual document from an AWS ARN for an S3 object, Secrets Manager secret, or
// Systems Manager parameter.
//
// Note that S3 objects are usually supplied in S3 URL form (s3://bucket/key) instead, which is handled by
// GetDocumentByURL directly.
func getDocumentFromARN(rawURL string) ([]byte, error) {
	docARN, err := arn.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	switch docARN.Service {
	case serviceS3:
		if docARN.Region != "" {
			return nil, fmt.Errorf("invalid S3 ARN: region cannot be set: %s", rawURL)
		}

		if docARN.AccountID != "" {
			return nil, fmt.Errorf("invalid S3 ARN: account ID cannot be set: %s", rawURL)
		}

		parts := strings.SplitN(docARN.Resource, "/", 2) //nolint:gomnd // Splitting once
		if len(parts) != 2 {                             //nolint:gomnd // Splitting once
			return nil, fmt.Errorf("invalid S3 resource in ARN: %s", rawURL)
		}

		return getDocumentFromS3(parts[0], parts[1])

	case serviceSecretsManager:
		if docARN.Region == "" {
			return nil, fmt.Errorf("invalid Secrets Manager ARN: region must be set: %s", rawURL)
		}

		if docARN.AccountID == "" {
			return nil, fmt.Errorf("invalid Secrets Manager ARN: account ID must be set: %s", rawURL)
		}

		if !strings.HasPrefix(docARN.Resource, "secret:") {
			return nil, fmt.Errorf("unsupported Secrets Manager resource in ARN: %s", rawURL)
		}

		return getDocumentFromSecretsManager(&docARN)

	case serviceSSM:
		if docARN.Region == "" {
			return nil, fmt.Errorf("invalid SSM ARN: region must be set: %s", rawURL)
		}

		if docARN.AccountID == "" {
			return nil, fmt.Errorf("invalid SSM ARN: account ID must be set: %s", rawURL)
		}

		if !strings.HasPrefix(docARN.Resource, "parameter/") {
			return nil, fmt.Errorf("unsupported SSM resource in ARN: %s", rawURL)
		}

		return getDocumentFromSSM(&docARN)
	}

	return nil, fmt.Errorf("unsupported AWS service %#v in ARN: %s", docARN.Service, rawURL)
}

// getDocumentFromFile retrieves a textual document from a file.
func getDocumentFromFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// getDocumentFromHTTP retrieves a textual document from an HTTP or HTTPS URL.
func getDocumentFromHTTP(rawURL string) ([]byte, error) {
	response, err := http.Get(rawURL) //nolint:gosec,noctx // We're getting HTTP data from an admin-specified URL.
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request to %s failed with status code %d", rawURL, response.StatusCode)
	}

	return io.ReadAll(response.Body)
}

// getDocumentFromS3 retrieves a textual document from the specified S3 bucket and key.
//
// If the object is server-side encrypted, S3 will automatically decrypt this for us before returning it. Client-side
// decryption is not supported.
func getDocumentFromS3(bucket, key string) ([]byte, error) {
	// We don't use the s3-proxy S3 client here to avoid polluting our metrics.
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	s3Client := s3.New(sess)

	goi := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	goo, err := s3Client.GetObject(&goi)

	if err != nil {
		return nil, err
	}

	defer goo.Body.Close()

	return io.ReadAll(goo.Body)
}

// getDocumentFromSecretsManager retrieves a textual document from the specified Secrets Manager secret.
func getDocumentFromSecretsManager(docARN *arn.ARN) ([]byte, error) {
	awsConfig := aws.Config{Region: aws.String(docARN.Region)}
	sess, err := session.NewSession(&awsConfig)

	if err != nil {
		return nil, err
	}

	// Make sure the account ID matches the ARN's account ID.
	stsClient := sts.New(sess)
	gcio, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})

	if err != nil {
		return nil, fmt.Errorf("unable to determine current account ID: %w", err)
	}

	if aws.StringValue(gcio.Account) != docARN.AccountID {
		return nil, fmt.Errorf("account ID in ARN: %s (current account is %s)", docARN.String(), aws.StringValue(gcio.Account))
	}

	secretsManagerClient := secretsmanager.New(sess)
	gsvi := secretsmanager.GetSecretValueInput{
		SecretId: aws.String(docARN.Resource[len("secret:"):]),
	}
	gsvo, err := secretsManagerClient.GetSecretValue(&gsvi)

	if err != nil {
		return nil, err
	}

	if gsvo.SecretBinary != nil {
		return gsvo.SecretBinary, nil
	}

	if gsvo.SecretString != nil {
		return []byte(*gsvo.SecretString), nil
	}

	return nil, fmt.Errorf("unexpected empty secret value")
}

// getDocumentFromSSM retrieves a textual document from the specified AWS Systems Manager parameter, decrypting it
// if necessary.
func getDocumentFromSSM(docARN *arn.ARN) ([]byte, error) {
	awsConfig := aws.Config{Region: aws.String(docARN.Region)}
	sess, err := session.NewSession(&awsConfig)

	if err != nil {
		return nil, err
	}

	// Make sure the account ID matches the ARN's account ID.
	stsClient := sts.New(sess)
	gcio, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})

	if err != nil {
		return nil, fmt.Errorf("unable to determine current account ID: %w", err)
	}

	if aws.StringValue(gcio.Account) != docARN.AccountID {
		return nil, fmt.Errorf("account ID in ARN: %s (current account is %s)", docARN.String(), aws.StringValue(gcio.Account))
	}

	ssmClient := ssm.New(sess)

	gpi := ssm.GetParameterInput{
		Name:           aws.String(docARN.Resource[len("parameter/"):]),
		WithDecryption: aws.Bool(true),
	}
	gpo, err := ssmClient.GetParameter(&gpi)

	if err != nil {
		return nil, err
	}

	if gpo.Parameter != nil {
		if gpo.Parameter.Value != nil {
			return []byte(*gpo.Parameter.Value), nil
		}
	}

	return nil, fmt.Errorf("unexpected empty parameter")
}
