//go:build unit

package server

import (
	"crypto/tls"
	"crypto/x509"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/stretchr/testify/assert"
)

var (
	// Test certificate, self-signed, for testhost.example.com
	testCertificate = `-----BEGIN CERTIFICATE-----
MIIDeDCCAmACCQDbKC6SZoxWRTANBgkqhkiG9w0BAQUFADB9MQswCQYDVQQGEwJV
UzETMBEGA1UECAwKV2FzaGluZ3RvbjEQMA4GA1UEBwwHU2VhdHRsZTEdMBsGA1UE
AwwUdGVzdGhvc3QuZXhhbXBsZS5jb20xKDAmBgkqhkiG9w0BCQEWGXRlc3RAdGVz
dGhvc3QuZXhhbXBsZS5jb20wIBcNMjIwMjE2MTYzNjU0WhgPMjEyMjAxMjMxNjM2
NTRaMH0xCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApXYXNoaW5ndG9uMRAwDgYDVQQH
DAdTZWF0dGxlMR0wGwYDVQQDDBR0ZXN0aG9zdC5leGFtcGxlLmNvbTEoMCYGCSqG
SIb3DQEJARYZdGVzdEB0ZXN0aG9zdC5leGFtcGxlLmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAL/yQZn2ZDxvtos+CDScWS7YKqlNgV0L2dAF/9SZ
WkhM6+vwrl0AP25+Xf6U50va8Ux2RUC7MCnhsmMq3dp8t1fUxs/WpViX30BE4tLJ
47OuvhSY05aDsUf902dQuTg0HaKxXYjuW8FvaaF9GaR3eu4eVU8ahm09D5YFtz5D
i/wsKkVqikzOsKvBi0dVHZ+fxBmf/1t4Mqualq4YqjWU2DGf7lfsdv6cCDKCmkgg
AWJ3yDA70fiUGq5nigBLE+5bPSTFE/PZOFK+WtQZV2//ykwkE/bk+UOTRkdZPZP0
TqgfkuQub2m3F8JhkzGPtfnQ5S9C+fsndCOd4OBfzcPCldkCAwEAATANBgkqhkiG
9w0BAQUFAAOCAQEAncN7syI1+HcuCEKxS7EArp9fA+bOQX6EIJhSuOeyNXKhHdlm
RFToPkoMRwsCnonmD44lNXjQ2LbTRE0ySCqIm6H9Ha9C7sLZAWnbOB2Iz65YbqyD
zJq0pnhb6TN9jiVO7kXIvcPWrrA1TwBo6Y7dx6Svy3WLlKbQWGwQx9q2Hr209s0L
GO9TXExY6u0fNFJDyh7KFeTablSIH+oDLAytZrjzBOyPqe8aZI2SXAcJjz3Hp9hv
V6sfsRW0PfYOsUxvMglI5LXHGflkM4tRC/WzNUhei6TJKxLhyk8FkSpkRvbsLVQn
JYwisSNsLosVijV7XU2AlvuoWQlNEkY8bPJx3Q==
-----END CERTIFICATE-----`

	testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAv/JBmfZkPG+2iz4INJxZLtgqqU2BXQvZ0AX/1JlaSEzr6/Cu
XQA/bn5d/pTnS9rxTHZFQLswKeGyYyrd2ny3V9TGz9alWJffQETi0snjs66+FJjT
loOxR/3TZ1C5ODQdorFdiO5bwW9poX0ZpHd67h5VTxqGbT0PlgW3PkOL/CwqRWqK
TM6wq8GLR1Udn5/EGZ//W3gyq5qWrhiqNZTYMZ/uV+x2/pwIMoKaSCABYnfIMDvR
+JQarmeKAEsT7ls9JMUT89k4Ur5a1BlXb//KTCQT9uT5Q5NGR1k9k/ROqB+S5C5v
abcXwmGTMY+1+dDlL0L5+yd0I53g4F/Nw8KV2QIDAQABAoIBAQCGkJbPEj55ZDQM
cCOehpG7Vo6p/I0Zpyo/PUV6TTxO/aZT1XrX9kmB9BN/W/K/ajHKUgwA8no0kmbW
QQIhn1eFusTahneKoYZA70o5TpJUsMfPdsi3d4G8n8UqZBxFu7ufCEszqS8ocCwU
q7hjZeQHtbpG56igQrN/kGhDvWURFsmAhi9763/wEgpDYWdLmw2hc7wPmuqg68r7
1Lk1CmcS7ZoQpx/QdhYtyG281f8lWOWQa/SL3VUQQl/J3U9GyCzSjHRy+ESSloYm
uzORowvexWB23324pAca6QYJPf5HqhzkLrfG3xTXI2xJPgoGiBMJqY84zxPaHJlm
mp8Laa4VAoGBAPBzskgH6t274P4slBML78M+E8zKM0amcEtWN+JgT7a1YKM3+3Wo
vwb/Y3RmHBN9Tget4shv2Gifm1zi4HmWgymt6ZTLnV9VfIrQXkC5PblDVCoAaxCL
ytWuLO5q+acq5iiVv5bB6mN0qm7GUl/dfClrHQ0bGb1V1l5BeRQnEdxnAoGBAMxb
oCHwwp3KDL7Xoxa08x7y0cEHAjyEnTFL/UIdZ+Bb/78HkxVAaYBq5fuw7bbcG8oT
Bjpn9FnOnNZXuy1ULNwl8OdkvYqOA5N8XwXcIA+yvIRTIwX/VTb8Rhie/FymStuT
UgA8HNoRjHy2eCP3VUmYI1t4KgmvOejB+HZZIJO/AoGAV7xPe/rvlvKb2QKZEQ4U
8S+wd9P7u7a1WLff8kjkLS2nUkb2COuGsF31gx5S9kWNeD3ZdvtggmRigxUBhTwH
JekgRru483U02U3IZmNxAy1vA1hduI7Zdvhzypbb+0Qq8PobCz48cQe7vGm+2t3t
FQvRcNvHm487he7r6A+Nc9cCgYEAtgwRlOqzlHj/7aqPYJUF19YcQUaLGXpRxi6Z
iCJF/To3k+edgVsGIR4ZjqPIwBNItjVIYRNmO/KxCMjSt8i6xcsO1jOKHjnwuZwb
0k6MSS/CfGbLVnZlZTxK/Xfz/F0vZnfQnuDuGt1zN04drHyS/6KGLN/ZIxN0FQNm
4Zb4TGUCgYEAl6eGVe+cZ5cIdwvNV49+X800BuZjSDSKNYBTaeIJWXeI9H+7b0qL
So0HeYWx9ixaRgxZ8yxGmB/CVOka/M5w06i0cwofTMWsiFYzPd6uPe2Mz6hcIPuE
csZ8PbpqNkbcznkfy8BDRhwanNsvzsXWyX/0LxU+CdZGQ9jDOZwItyY=
-----END RSA PRIVATE KEY-----`

	emptyCertificate = `-----BEGIN CERTIFICATE-----
-----END CERTIFICATE-----`

	emptyPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
-----END RSA PRIVATE KEY-----`
)

func TestGenerateTLSConfig(t *testing.T) {
	type resultType int
	const (
		expectNil resultType = iota
		expectStruct
		expectErr
	)

	tests := []struct {
		name         string
		config       *config.ServerSSLConfig
		expect       resultType
		errorString  string
		minVersion   *uint16
		maxVersion   *uint16
		cipherSuites []uint16
		certCount    *int
		certDNS      [][]string
	}{
		{
			name:   "nil config results in nil tls.Config",
			config: nil,
			expect: expectNil,
		},
		{
			name:   "enabled=false results in nil tls.Config",
			config: &config.ServerSSLConfig{Enabled: false},
			expect: expectNil,
		},
		{
			name:   "enabled=true with no certificates should result in an error",
			config: &config.ServerSSLConfig{Enabled: true},
			expect: expectErr,
		},
		{
			name: "default ciper suites and min version set",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(testCertificate),
						PrivateKey:  aws.String(testPrivateKey),
					},
				},
			},
			expect:       expectStruct,
			minVersion:   aws.Uint16(tls.VersionTLS12),
			cipherSuites: defaultCipherSuites,
		},
		{
			name: "versions, cipher suites, certificates respected and generated",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(testCertificate),
						PrivateKey:  aws.String(testPrivateKey),
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				MinTLSVersion:       aws.String("TLSv1.1"),
				MaxTLSVersion:       aws.String("TLSv1.2"),
				CipherSuites:        []string{"TLS_RSA_WITH_AES_128_GCM_SHA256"},
			},
			expect:       expectStruct,
			minVersion:   aws.Uint16(tls.VersionTLS11),
			maxVersion:   aws.Uint16(tls.VersionTLS12),
			cipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_GCM_SHA256},
			certDNS:      [][]string{{"localhost", "localhost.localdomain"}, {"testhost.example.com"}},
		},
		{
			name: "invalid min TLS version",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(testCertificate),
						PrivateKey:  aws.String(testPrivateKey),
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				MinTLSVersion:       aws.String("5.0"),
				MaxTLSVersion:       aws.String("TLSv1.2"),
				CipherSuites:        []string{"TLS_RSA_WITH_AES_128_GCM_SHA256"},
			},
			expect:      expectErr,
			errorString: "invalid TLS version: 5.0",
		},
		{
			name: "invalid max TLS version",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(testCertificate),
						PrivateKey:  aws.String(testPrivateKey),
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				MinTLSVersion:       aws.String("tls1.1"),
				MaxTLSVersion:       aws.String("6.0"),
				CipherSuites:        []string{"TLS_RSA_WITH_AES_128_GCM_SHA256"},
			},
			expect:      expectErr,
			errorString: "invalid TLS version: 6.0",
		},
		{
			name: "invalid cipher suite",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(testCertificate),
						PrivateKey:  aws.String(testPrivateKey),
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				MinTLSVersion:       aws.String("TLSv1.1"),
				MaxTLSVersion:       aws.String("TLSv1.2"),
				CipherSuites:        []string{"TLS_RSA_WITH_AES_128_GCM_SHA256", "TLS_NULL"},
			},
			expect:      expectErr,
			errorString: "invalid cipher suite: TLS_NULL",
		},
		{
			name: "neither certificate nor certificateUrl set",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						PrivateKey: aws.String(testPrivateKey),
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				MinTLSVersion:       aws.String("TLSv1.1"),
				MaxTLSVersion:       aws.String("TLSv1.2"),
				CipherSuites:        []string{"TLS_RSA_WITH_AES_128_GCM_SHA256"},
			},
			expect:      expectErr,
			errorString: "unable to load certificate: expected either certificate or certificateUrl to be set",
		},
		{
			name: "neither privateKey nor privateKeyUrl set",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(testCertificate),
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				MinTLSVersion:       aws.String("TLSv1.1"),
				MaxTLSVersion:       aws.String("TLSv1.2"),
				CipherSuites:        []string{"TLS_RSA_WITH_AES_128_GCM_SHA256"},
			},
			expect:      expectErr,
			errorString: "unable to load certificate: expected either privateKey or privateKeyUrl to be set",
		},
		{
			name: "invalid certificate URL should result in an error",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String(":r&qwer+asdf"),
						PrivateKey:     aws.String(testPrivateKey),
					},
				},
			},
			expect: expectErr,
		},
		{
			name: "unsupported certificate URL scheme should result in an error",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("ftp://ftp.example.com"),
						PrivateKey:     aws.String(testPrivateKey),
					},
				},
			},
			expect:      expectErr,
			errorString: "unable to load certificate: failed to get certificate from URL: ftp://ftp.example.com: unsupported URL scheme: ftp",
		},
		{
			name: "unsupported private key URL scheme should result in an error",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate:   aws.String(testCertificate),
						PrivateKeyURL: aws.String("ftp://ftp.example.com"),
					},
				},
			},
			expect:      expectErr,
			errorString: "unable to load certificate: failed to get private key from URL: ftp://ftp.example.com: unsupported URL scheme: ftp",
		},
		{
			name: "invalid HTTP timeout duration",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("http://example.com/certificate.pem"),
						CertificateURLConfig: &config.SSLURLConfig{
							HTTPTimeout: "foobar",
						},
						PrivateKey: aws.String(testPrivateKey),
					},
				},
			},
			expect:      expectErr,
			errorString: "unable to load certificate: invalid certificateUrlConfig: invalid httpTimeout: time: invalid duration \"foobar\"",
		},
		{
			name: "empty certificate",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(emptyCertificate),
						PrivateKey:  aws.String(testPrivateKey),
					},
				},
			},
			expect:      expectErr,
			errorString: "unable to load certificate: failed to create certificate:",
		},
		{
			name: "empty private key",
			config: &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: aws.String(testCertificate),
						PrivateKey:  aws.String(emptyPrivateKey),
					},
				},
			},
			expect:      expectErr,
			errorString: "unable to load certificate: failed to create certificate:",
		},
	}

	for _, currentTest := range tests {
		// Capture the current test for parallel processing. Otherwise currentTest will be modified during our test run.
		tt := currentTest

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			logger := log.NewLogger()
			tlsConfig, err := generateTLSConfig(tt.config, logger)

			switch tt.expect {
			case expectNil:
				if err != nil {
					t.Errorf("expected nil result but got error %v", err)
				} else if tlsConfig != nil {
					t.Errorf("expected nil result but got tls.Config")
				}
			case expectStruct:
				if err != nil {
					t.Errorf("expected successful result but got error %v", err)
					break
				}

				if tlsConfig == nil {
					t.Errorf("expected successful result but got nil")
					break
				}

				// If minVersion/maxVersion were set in the test config, check these results.
				if tt.minVersion != nil {
					if *tt.minVersion != tlsConfig.MinVersion {
						t.Errorf("expected MinVersion set to 0x%04x; got 0x%04x", *tt.minVersion, tlsConfig.MinVersion)
					}
				}

				if tt.maxVersion != nil {
					if *tt.maxVersion != tlsConfig.MaxVersion {
						t.Errorf("expected MaxVersion set to 0x%04x; got 0x%04x", *tt.maxVersion, tlsConfig.MaxVersion)
					}
				}

				// If we expected certain cipher suites, check that the result includes exactly this set.
				if len(tt.cipherSuites) > 0 {
					ciphersSeen := make(map[uint16]bool)

					for _, cipher := range tt.cipherSuites {
						ciphersSeen[cipher] = false
					}

					for _, cipher := range tlsConfig.CipherSuites {
						_, present := ciphersSeen[cipher]

						if !present {
							t.Errorf("unexpected cipher suite 0x%04x included", cipher)
						} else {
							ciphersSeen[cipher] = true
						}
					}

					for cipher, seen := range ciphersSeen {
						if !seen {
							t.Errorf("missing cipher suite 0x%04x", cipher)
						}
					}
				}

				// If we expected a certificate count, check it.
				if tt.certCount != nil {
					if *tt.certCount != len(tlsConfig.Certificates) {
						t.Errorf("expected %d certificates; got %d", *tt.certCount, len(tlsConfig.Certificates))
					}
				}

				// If we specified names in tt.certDNS, check those certs for the supplied names.
				// Ignore any certificates beyond those specified.
				limit := len(tlsConfig.Certificates)
				if limit > len(tt.certDNS) {
					limit = len(tt.certDNS)
				}

				assert.LessOrEqual(t, limit, len(tt.certDNS))

				for certIdx := 0; certIdx < limit; certIdx++ {
					namesSeen := make(map[string]bool)

					assert.Less(t, certIdx, len(tt.certDNS))
					for _, dnsName := range tt.certDNS[certIdx] {
						namesSeen[dnsName] = false
					}

					// We look *only* at the first certificate (the leaf); others are intermediates.
					certBytes := tlsConfig.Certificates[certIdx].Certificate[0]
					cert, err := x509.ParseCertificate(certBytes)

					if err != nil {
						t.Errorf("error parsing certificates[%d].certificate[0]: %v", certIdx, err)
						continue
					}

					// We need to gather the common name (CN) as well as the subject alternate names (SANs).
					// Some certs don't include the CN in the SANs.
					allDNSNames := make([]string, 0, 1+len(cert.DNSNames))
					allDNSNames = append(allDNSNames, cert.Subject.CommonName)
					allDNSNames = append(allDNSNames, cert.DNSNames...)

					for _, dnsName := range allDNSNames {
						_, present := namesSeen[dnsName]
						if !present {
							// DNS name found wasn't included on our list.
							t.Errorf("certificates[%d] has unexpected DNS name %#v", certIdx, dnsName)
						} else {
							// It's ok if the name is present more than once (e.g. CN and SAN)
							namesSeen[dnsName] = true
						}
					}

					for dnsName, seen := range namesSeen {
						if !seen {
							// DNS name on our list was not present in the certificates.
							t.Errorf("certificates[%d] did not include DNS name %#v", certIdx, dnsName)
						}
					}
				}

			case expectErr:
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if !strings.HasPrefix(err.Error(), tt.errorString) {
					t.Errorf("expected error message %#v; got %#v", tt.errorString, err.Error())
				}
			}
		})
	}
}
