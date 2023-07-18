package generalutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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

// ClientIP will return client ip from request.
func ClientIP(r *http.Request) string {
	ipAddress := r.Header.Get("X-Real-Ip")
	if ipAddress == "" {
		ipAddress = r.Header.Get("X-Forwarded-For")
	}

	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	return ipAddress
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
		if tv := strings.SplitN(forwardedPair, "=", 2); len(tv) == 2 { //nolint: gomnd // No constant for that
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

// GetDocumentFromURLOption is a type alias for a function that can set various options to GetDocumentFromURL.
type GetDocumentFromURLOption func(awsCfg *aws.Config, httpClient *http.Client)

// WithAWSEndpoint is an option for GetDocumentFromURL to set the AWS service endpoint.
func WithAWSEndpoint(endpoint string) GetDocumentFromURLOption {
	return func(awsCfg *aws.Config, httpClient *http.Client) {
		if awsCfg != nil {
			customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					PartitionID:       "aws",
					URL:               endpoint,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})
			awsCfg.EndpointResolverWithOptions = customResolver
		}
	}
}

// WithAWSRegion is an option for GetDocumentFromURL to set the AWS region.
func WithAWSRegion(region string) GetDocumentFromURLOption {
	return func(awsCfg *aws.Config, httpClient *http.Client) {
		if awsCfg != nil {
			awsCfg.Region = region
		}
	}
}

// WithAWSStaticCredentials is an option for GetDocumentFromURL to set AWS credentials.
func WithAWSStaticCredentials(accessKey, secretKey, token string) GetDocumentFromURLOption {
	return func(awsCfg *aws.Config, httpClient *http.Client) {
		if awsCfg != nil {
			awsCfg.Credentials = credentials.NewStaticCredentialsProvider(accessKey, secretKey, token)
		}
	}
}

// WithHTTPTimeout is an option for GetDocumentFromURL to set the HTTP timeout.
func WithHTTPTimeout(timeout time.Duration) GetDocumentFromURLOption {
	return func(awsCfg *aws.Config, httpClient *http.Client) {
		if awsCfg != nil && awsCfg.HTTPClient == nil {
			awsCfg.HTTPClient = &http.Client{Timeout: timeout}
		}

		if httpClient != nil {
			httpClient.Timeout = timeout
		}
	}
}

// GetDocumentFromURL retrieves a textual document from a URL, which may be an AWS ARN for an S3 object,
// Secrets Manager secret, or  Systems Manager parameter (arn:...); an HTTP or HTTPS URL; an S3 URL in
// the form s3://bucket/key; or a file in URL or regular path form.
func GetDocumentFromURL(rawURL string, opts ...GetDocumentFromURLOption) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	switch parsedURL.Scheme {
	case "":
		return getDocumentFromFile(rawURL)
	case schemeARN:
		return getDocumentFromARN(rawURL, opts...)
	case schemeFile:
		if parsedURL.RawQuery != "" {
			return nil, errors.New("file URL cannot contain query")
		}

		if parsedURL.Fragment != "" {
			return nil, errors.New("file URL cannot contain fragment")
		}

		return getDocumentFromFile(parsedURL.Path)
	case schemeHTTP, schemeHTTPS:
		if parsedURL.Fragment != "" {
			return nil, errors.Errorf("%s URL cannot contain fragment", parsedURL.Scheme)
		}

		return getDocumentFromHTTP(rawURL, opts...)
	case schemeS3:
		if !strings.Contains(parsedURL.Path, "/") {
			return nil, errors.New("missing S3 key")
		}

		return getDocumentFromS3(parsedURL.Host, parsedURL.Path, opts...)
	}

	return nil, errors.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
}

// getDocumentFromARN retrieves a textual document from an AWS ARN for an S3 object, Secrets Manager secret, or
// Systems Manager parameter.
//
// Note that S3 objects are usually supplied in S3 URL form (s3://bucket/key) instead, which is handled by
// GetDocumentByURL directly.
func getDocumentFromARN(rawURL string, opts ...GetDocumentFromURLOption) ([]byte, error) {
	docARN, err := validateDocumentARN(rawURL)
	if err != nil {
		return nil, err
	}

	// Service and resource has already been validated here.
	switch docARN.Service {
	case serviceS3:
		parts := strings.SplitN(docARN.Resource, "/", 2) //nolint:gomnd // Splitting once
		if len(parts) != 2 {                             //nolint:gomnd // Splitting once
			// Should not get here; covered by validateDocumentARN.
			return nil, errors.New("missing S3 key")
		}

		return getDocumentFromS3(parts[0], parts[1], opts...)

	case serviceSecretsManager:
		return getDocumentFromSecretsManager(docARN, opts...)

	case serviceSSM:
		return getDocumentFromSSM(docARN, opts...)

	default:
		// Should not get here; covered by validateDocumentARN.
		return nil, errors.Errorf("unsupported AWS service %#v in ARN", docARN.Service)
	}
}

// getDocumentFromFile retrieves a textual document from a file.
func getDocumentFromFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// getDocumentFromHTTP retrieves a textual document from an HTTP or HTTPS URL.
func getDocumentFromHTTP(rawURL string, opts ...GetDocumentFromURLOption) ([]byte, error) {
	httpClient := &http.Client{}

	for _, opt := range opts {
		opt(nil, httpClient)
	}

	response, err := httpClient.Get(rawURL) //nolint:noctx // No context available.
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("request to %s failed with status code %d", rawURL, response.StatusCode)
	}

	return io.ReadAll(response.Body)
}

// getDocumentFromS3 retrieves a textual document from the specified S3 bucket and key. Optional AWS session
// configuration may be provided to override the endpoint and TLS settings.
//
// If the object is server-side encrypted, S3 will automatically decrypt this for us before returning it. Client-side
// decryption is not supported.
func getDocumentFromS3(bucket, key string, opts ...GetDocumentFromURLOption) ([]byte, error) {
	cfg, err := awscfg.LoadDefaultConfig(context.TODO())
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Add options
	for _, opt := range opts {
		opt(&cfg, nil)
	}

	// We don't use the s3-proxy S3 client here to avoid polluting our metrics.
	s3Client := s3.NewFromConfig(cfg)

	goi := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	goo, err := s3Client.GetObject(context.TODO(), &goi)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer goo.Body.Close()

	bb, err := io.ReadAll(goo.Body)
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return bb, nil
}

// getDocumentFromSecretsManager retrieves a textual document from the specified Secrets Manager secret.
//
// TODO: To support testing, this needs to take a context argument so an STS/Secrets Manager client can be injected for testing.
// This requires changes in the server.
func getDocumentFromSecretsManager(docARN *arn.ARN, opts ...GetDocumentFromURLOption) ([]byte, error) {
	cfg, err := awscfg.LoadDefaultConfig(context.TODO())
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cfg.Region = docARN.Region

	for _, opt := range opts {
		opt(&cfg, nil)
	}

	// Make sure the account ID matches the ARN's account ID.
	stsClient := sts.NewFromConfig(cfg)
	gcio, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})

	if err != nil {
		return nil, errors.Wrap(err, "unable to determine current account ID")
	}

	if gcio.Account != nil && *gcio.Account != docARN.AccountID {
		return nil, errors.Errorf("account ID in ARN: %s (current account is %s)", docARN.String(), *gcio.Account)
	}

	secretsManagerClient := secretsmanager.NewFromConfig(cfg)
	gsvi := secretsmanager.GetSecretValueInput{
		SecretId: aws.String(docARN.Resource[len("secret:"):]),
	}
	gsvo, err := secretsManagerClient.GetSecretValue(context.TODO(), &gsvi)

	if err != nil {
		return nil, err
	}

	if gsvo.SecretBinary != nil {
		return gsvo.SecretBinary, nil
	}

	if gsvo.SecretString != nil {
		return []byte(*gsvo.SecretString), nil
	}

	return nil, errors.Errorf("unexpected empty secret value")
}

// getDocumentFromSSM retrieves a textual document from the specified AWS Systems Manager parameter, decrypting it
// if necessary.
//
// TODO: To support testing, this needs to take a context argument so an STS/SSM client can be injected for testing.
// This requires changes in the server.
func getDocumentFromSSM(docARN *arn.ARN, opts ...GetDocumentFromURLOption) ([]byte, error) {
	cfg, err := awscfg.LoadDefaultConfig(context.TODO())
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cfg.Region = docARN.Region

	for _, opt := range opts {
		opt(&cfg, nil)
	}

	// Make sure the account ID matches the ARN's account ID.
	stsClient := sts.NewFromConfig(cfg)
	gcio, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})

	if err != nil {
		return nil, errors.Wrap(err, "unable to determine current account ID")
	}

	if gcio.Account != nil && *gcio.Account != docARN.AccountID {
		return nil, errors.Errorf("account ID in ARN does not match current acccount: %s (current account is %s)", docARN.String(), *gcio.Account)
	}

	ssmClient := ssm.NewFromConfig(cfg)

	gpi := ssm.GetParameterInput{
		Name:           aws.String(docARN.Resource[len("parameter/"):]),
		WithDecryption: aws.Bool(true),
	}
	gpo, err := ssmClient.GetParameter(context.TODO(), &gpi)
	// Check error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if gpo.Parameter != nil {
		if gpo.Parameter.Value != nil {
			return []byte(*gpo.Parameter.Value), nil
		}
	}

	return nil, errors.Errorf("unexpected empty parameter")
}

// ValidateDocumentURL verifies the document URL is supported.
//
// If the URL is malformed, contains an unsupported scheme, or uses unsupported features (e.g. query arguments or
// fragments for AWS URLs), an error is returned.
func ValidateDocumentURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return errors.WithStack(err)
	}

	switch parsedURL.Scheme {
	case schemeARN:
		_, err := validateDocumentARN(rawURL)

		return err

	case "":
		// File -- always ok.
		return nil

	case schemeFile:
		if parsedURL.RawQuery != "" {
			return errors.New("file URL cannot contain query")
		}

		if parsedURL.Fragment != "" {
			return errors.New("file URL cannot contain fragment")
		}

		return nil

	case schemeHTTP, schemeHTTPS:
		if parsedURL.Fragment != "" {
			return errors.Errorf("%s URL cannot contain fragment", parsedURL.Scheme)
		}

		return nil

	case schemeS3:
		if parsedURL.RawQuery != "" {
			return errors.New("s3 URL cannot contain query")
		}

		if parsedURL.Fragment != "" {
			return errors.New("s3 URL cannot contain fragment")
		}

		return nil
	}

	return errors.Errorf("unsupported URL scheme %s", parsedURL.Scheme)
}

func validateDocumentARN(rawURL string) (*arn.ARN, error) {
	docARN, err := arn.Parse(rawURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch docARN.Service {
	case serviceS3:
		if docARN.Region != "" {
			return nil, errors.New("invalid S3 ARN: region cannot be set")
		}

		if docARN.AccountID != "" {
			return nil, errors.New("invalid S3 ARN: account ID cannot be set")
		}

		parts := strings.SplitN(docARN.Resource, "/", 2) //nolint:gomnd // Splitting once
		if len(parts) != 2 {                             //nolint:gomnd // Splitting once
			return nil, errors.New("missing S3 key")
		}

		return &docARN, nil

	case serviceSecretsManager:
		if docARN.Region == "" {
			return nil, errors.New("invalid Secrets Manager ARN: region must be set")
		}

		if docARN.AccountID == "" {
			return nil, errors.New("invalid Secrets Manager ARN: account ID must be set")
		}

		if !strings.HasPrefix(docARN.Resource, "secret:") {
			return nil, errors.New("unsupported Secrets Manager resource in ARN: %s")
		}

		return &docARN, nil

	case serviceSSM:
		if docARN.Region == "" {
			return nil, errors.New("invalid SSM ARN: region must be set")
		}

		if docARN.AccountID == "" {
			return nil, errors.New("invalid SSM ARN: account ID must be set")
		}

		if !strings.HasPrefix(docARN.Resource, "parameter/") {
			return nil, errors.New("unsupported SSM resource in ARN")
		}

		return &docARN, nil
	}

	return nil, errors.Errorf("unsupported AWS service in ARN: %v", docARN.Service)
}
