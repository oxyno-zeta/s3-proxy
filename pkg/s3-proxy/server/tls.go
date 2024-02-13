package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"emperror.dev/errors"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	utils "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
)

// The intersection of the recommended cipher suites from https://ciphersuite.info/cs/?security=recommended
// and the suites implemented in Go.
var defaultCipherSuites = []uint16{
	tls.TLS_AES_128_GCM_SHA256,
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
}

// The number of bits to allow in a generated certificate serial number.
const certSerialBits = 128

// The number of bits to use in a generated RSA private key.
const rsaKeySize = 2048

// How long generated self-signed certificates should be valid for (10 years).
const certValidityDuration = 10 * 365 * 24 * time.Hour

// generateTLSConfig creates a crypto/tls.Config configuration for a net/http.Server from an s3-proxy
// ServerSSLConfig.
func generateTLSConfig(sslConfig *config.ServerSSLConfig, logger log.Logger) (*tls.Config, error) {
	result := tls.Config{
		MinVersion:   tls.VersionTLS12,
		CipherSuites: defaultCipherSuites,
	}

	if sslConfig == nil || !sslConfig.Enabled {
		return nil, nil //nolint:nilnil // We do not want a TLS config in these cases
	}

	if len(sslConfig.SelfSignedHostnames) == 0 && len(sslConfig.Certificates) == 0 {
		return nil, errors.New("at least one certificate must be specified")
	}

	// Generate self-signed certificates for each hostname requested
	if len(sslConfig.SelfSignedHostnames) > 0 {
		selfSignedCert, err := generateSelfSignedCertificate(sslConfig.SelfSignedHostnames)
		if err != nil {
			logger.Errorf("failed to generate self-signed certificate: %v", err)

			return nil, err
		}

		result.Certificates = append(result.Certificates, selfSignedCert)
	}

	// Set min and max TLS versions if they were specified in the config.
	if sslConfig.MinTLSVersion != nil {
		result.MinVersion = utils.ParseTLSVersion(*sslConfig.MinTLSVersion)
		if result.MinVersion == 0 {
			logger.Errorf("invalid TLS version: %v", *sslConfig.MinTLSVersion)

			return nil, errors.Errorf("invalid TLS version: %v", *sslConfig.MinTLSVersion)
		}
	}

	if sslConfig.MaxTLSVersion != nil {
		result.MaxVersion = utils.ParseTLSVersion(*sslConfig.MaxTLSVersion)
		if result.MaxVersion == 0 {
			logger.Errorf("invalid TLS version: %v", *sslConfig.MaxTLSVersion)

			return nil, errors.Errorf("invalid TLS version: %v", *sslConfig.MaxTLSVersion)
		}
	}

	// Set the cipher suites if they were specified in the config.
	if len(sslConfig.CipherSuites) > 0 {
		result.CipherSuites = nil

		for _, cipherSuiteName := range sslConfig.CipherSuites {
			suiteID := utils.ParseCipherSuite(cipherSuiteName)
			if suiteID == 0 {
				logger.Errorf("invalid cipher suite: %v", cipherSuiteName)

				return nil, errors.Errorf("invalid cipher suite: %v", cipherSuiteName)
			}

			result.CipherSuites = append(result.CipherSuites, suiteID)
		}
	}

	// Add each supplied certificate to the TLS config.
	for _, certConfig := range sslConfig.Certificates {
		cert, err := getCertificateFromConfig(certConfig, logger)

		if err != nil {
			logger.Errorf("unable to load certificate: %v", err)

			return nil, errors.Wrap(err, "unable to load certificate")
		}

		result.Certificates = append(result.Certificates, *cert)
	}

	return &result, nil
}

// getCertificateFromConfig creates a crypto/tls.Certificate from a certificate configuration, performing any
// network accesses (S3, SSM, Secrets Manager) necessary.
func getCertificateFromConfig(certConfig *config.ServerSSLCertificate, logger log.Logger) (*tls.Certificate, error) {
	var certificate, privateKey []byte

	certificateURLOptions, err := getURLOptions(certConfig.CertificateURLConfig)
	if err != nil {
		return nil, errors.Wrap(err, "invalid certificateUrlConfig")
	}

	privateKeyURLOptions, err := getURLOptions(certConfig.PrivateKeyURLConfig)
	if err != nil {
		return nil, errors.Wrap(err, "invalid privateKeyUrlConfig")
	}

	switch {
	// Certificate supplied directly; just copy it.
	case certConfig.Certificate != nil:
		certificate = []byte(*certConfig.Certificate)

	// Certificate supplied as a URL.
	case certConfig.CertificateURL != nil:
		certificate, err = utils.GetDocumentFromURL(*certConfig.CertificateURL, certificateURLOptions...)

		if err != nil {
			logger.Errorf("Failed to get certificate from URL %s: %v", *certConfig.CertificateURL, err)

			return nil, errors.Wrap(err, "failed to get certificate from URL: "+*certConfig.CertificateURL)
		}

	default:
		logger.Error("Expected either certificate or certificateUrl to be set")

		return nil, errors.New("expected either certificate or certificateUrl to be set")
	}

	switch {
	// Private key supplied directly; just copy it.
	case certConfig.PrivateKey != nil:
		privateKey = []byte(*certConfig.PrivateKey)

	// Private key supplied as a URL.
	case certConfig.PrivateKeyURL != nil:
		privateKey, err = utils.GetDocumentFromURL(*certConfig.PrivateKeyURL, privateKeyURLOptions...)

		if err != nil {
			logger.Errorf("Failed to get private key from URL %s: %v", *certConfig.PrivateKeyURL, err)

			return nil, errors.Wrap(err, "failed to get private key from URL: "+*certConfig.PrivateKeyURL)
		}

	default:
		logger.Error("Expected either privateKey or privateKeyUrl to be set")

		return nil, errors.New("expected either privateKey or privateKeyUrl to be set")
	}

	cert, err := tls.X509KeyPair(certificate, privateKey)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create certificate")
	}

	if len(cert.Certificate) == 0 {
		return nil, errors.New("no certificates loaded")
	}

	for _, cert := range cert.Certificate {
		if len(cert) == 0 {
			return nil, errors.New("empty certificate loaded")
		}
	}

	return &cert, nil
}

// generateSelfSignedCertificate returns a single crypto/tls.Certificate containing a self-signed certficate for
// the specified hostnames.
func generateSelfSignedCertificate(hostnames []string) (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)

	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to generate private key")
	}

	now := time.Now().UTC()

	// Make the start time an hour earlier to account for clock skew.
	startTime := now.Add(-1 * time.Hour)

	// End time is 10 years.
	endTime := startTime.Add(certValidityDuration)

	// Generate a universally unique serial number.
	one := big.NewInt(1)
	maxSerialNumber := &big.Int{}
	maxSerialNumber.Lsh(one, certSerialBits)
	serialNumber, err := rand.Int(rand.Reader, maxSerialNumber)

	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to generate serial number")
	}

	template := x509.Certificate{
		DNSNames:           hostnames,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotAfter:           endTime,
		NotBefore:          startTime,
		SerialNumber:       serialNumber,
		SignatureAlgorithm: x509.SHA256WithRSA,
		Subject:            pkix.Name{CommonName: hostnames[0]},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, privateKey.Public(), privateKey)

	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to create self-signed certificate")
	}

	if len(certDER) == 0 {
		return tls.Certificate{}, errors.New("failed to create self-signed certificate: empty certificate")
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}, nil
}

func getURLOptions(urlConfig *config.SSLURLConfig) ([]utils.GetDocumentFromURLOption, error) {
	if urlConfig == nil {
		return nil, nil
	}

	var result []utils.GetDocumentFromURLOption

	if urlConfig.HTTPTimeout != "" {
		timeout, err := time.ParseDuration(urlConfig.HTTPTimeout)
		if err != nil {
			return nil, errors.Wrap(err, "invalid httpTimeout")
		}

		result = append(result, utils.WithHTTPTimeout(timeout))
	}

	if urlConfig.AWSRegion != "" {
		result = append(result, utils.WithAWSRegion(urlConfig.AWSRegion))
	}

	if urlConfig.AWSEndpoint != "" {
		result = append(result, utils.WithAWSEndpoint(urlConfig.AWSEndpoint))
	}

	if urlConfig.AWSDisableSSL {
		result = append(result, utils.WithAWSDisableSSL(urlConfig.AWSDisableSSL))
	}

	if s3Creds := urlConfig.AWSCredentials; s3Creds != nil {
		if s3Creds.AccessKey != nil && s3Creds.SecretKey != nil && s3Creds.AccessKey.Value != "" && s3Creds.SecretKey.Value != "" {
			result = append(result, utils.WithAWSStaticCredentials(s3Creds.AccessKey.Value, s3Creds.SecretKey.Value, ""))
		}
	}

	return result, nil
}
