package server

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
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
)

func TestGenerateTLSConfig(t *testing.T) {
	logger := log.NewLogger()
	var tlsConfig *tls.Config
	var err error

	// Nil config should result in a nil result.
	tlsConfig, err = generateTLSConfig(nil, logger)

	if err != nil {
		t.Errorf("generateTLSConfig(nil) should not return an error")
	} else if tlsConfig != nil {
		t.Errorf("generateTLSConfig(nil) should return a nil result")
	}

	// Enabled: false should result in a nil result.
	tlsConfig, err = generateTLSConfig(&config.ServerSSLConfig{Enabled: aws.Bool(false)}, logger)

	if err != nil {
		t.Errorf("generateTLSConfig({Enabled: false}) should not return an error")
	} else if tlsConfig != nil {
		t.Errorf("generateTLSConfig({Enabled: false}) should return a nil result")
	}

	// Enabled: nil with no certificates should result in a nil result.
	tlsConfig, err = generateTLSConfig(&config.ServerSSLConfig{}, logger)

	if err != nil {
		t.Errorf("generateTLSConfig({}) should not return an error")
	} else if tlsConfig != nil {
		t.Errorf("generateTLSConfig({}) should return a nil result")
	}

	// Enabled: true with no certificates should result in an error.
	_, err = generateTLSConfig(&config.ServerSSLConfig{Enabled: aws.Bool(true)}, logger)

	if err == nil {
		t.Errorf("generateTLSConfig({Enabled: true}) with no certificates should return an error")
	}

	// Check that the default cipher suites and min version are set.
	tlsConfig, err = generateTLSConfig(&config.ServerSSLConfig{
		Certificates: []config.ServerSSLCertificate{
			{
				Certificate: aws.String(testCertificate),
				PrivateKey:  aws.String(testPrivateKey),
			},
		},
	}, logger)

	if err != nil {
		t.Errorf("generateTLSConfig with certs should not return an error")
	} else if tlsConfig == nil {
		t.Errorf("generateTLSConfig with certs should return a valid tls.Config")
	} else {
		if tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("generateTLSConfig with certs should set MinVersion to TLS 1.2")
		}

		if len(tlsConfig.CipherSuites) == 0 {
			t.Errorf("generateTLSConfig should set CipherSuites to the default value")
		} else {
			notSeen := make(map[uint16]bool)

			for _, cipher := range defaultCipherSuites {
				notSeen[cipher] = true
			}

			for _, cipher := range tlsConfig.CipherSuites {
				_, present := notSeen[cipher]

				if !present {
					t.Errorf("generateTLSConfig default cipher suites included unexpected cipher 0x%04x", cipher)
				} else {
					delete(notSeen, cipher)
				}
			}

			for cipher := range notSeen {
				t.Errorf("generateTLSConfig default cipher suites missing expected cipher suite 0x%04x", cipher)
			}
		}
	}

	// Check that versions are set, cipher suites are respected, and self-signed certificates are generated.
	tlsConfig, err = generateTLSConfig(&config.ServerSSLConfig{
		Enabled: aws.Bool(true),
		Certificates: []config.ServerSSLCertificate{
			{
				Certificate: aws.String(testCertificate),
				PrivateKey:  aws.String(testPrivateKey),
			},
		},
		SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
		MinTLSVersion:       aws.String("TLSv1.1"),
		MaxTLSVersion:       aws.String("TLSv1.2"),
		CipherSuites:        []string{"TLS_RSA_WITH_AES_128_GCM_SHA256"},
	}, logger)

	if err != nil {
		t.Errorf("generateTLSConfig with certs, min field, cipher should not return an error")
	} else if tlsConfig == nil {
		t.Errorf("generateTLSConfig with certs, min field, cipher should return a tls.Config")
	} else {
		if tlsConfig.MinVersion != tls.VersionTLS11 {
			t.Errorf("expected min version to be TLS 1.1 (%x), got %x", tls.VersionTLS11, tlsConfig.MinVersion)
		}

		if tlsConfig.MaxVersion != tls.VersionTLS12 {
			t.Errorf("expected max version to be TLS 1.2 (%x), got %x", tls.VersionTLS12, tlsConfig.MinVersion)
		}

		if len(tlsConfig.CipherSuites) != 1 {
			t.Errorf("expected 1 cipher suite, got %d", len(tlsConfig.CipherSuites))
		} else if tlsConfig.CipherSuites[0] != tls.TLS_RSA_WITH_AES_128_GCM_SHA256 {
			t.Errorf("expected cipher suite to be 0x%04x, got 0x%04x", tls.TLS_RSA_WITH_AES_128_GCM_SHA256, tlsConfig.CipherSuites[0])
		}

		if len(tlsConfig.Certificates) != 2 {
			t.Errorf("expected 2 certificates, got %d", len(tlsConfig.Certificates))
		} else {
			if len(tlsConfig.Certificates[0].Certificate) < 1 {
				t.Errorf("expected self-signed certificate to be non-empty: %#v", tlsConfig.Certificates[1])
			} else {
				// Self-signed is the first certificate
				selfSignedBytes := tlsConfig.Certificates[0].Certificate[0]
				selfSigned, err := x509.ParseCertificate(selfSignedBytes)

				if err != nil {
					t.Errorf("error parsing self-signed certificate: %v", err)
				} else if len(selfSigned.DNSNames) != 2 {
					t.Errorf("expected 2 DNS names in self-signed certificate, got %d", len(selfSigned.DNSNames))
				} else {
					localhostSeen := false
					localhostLocalDomainSeen := false

					for _, dnsName := range selfSigned.DNSNames {
						if dnsName == "localhost" {
							localhostSeen = true
						} else if dnsName == "localhost.localdomain" {
							localhostLocalDomainSeen = true
						} else {
							t.Errorf("expected self-signed certificate to have DNS name 'localhost' or 'localhost.localdomain', got '%s'", dnsName)
						}
					}

					if !localhostSeen {
						t.Errorf("expected self-signed certificate to have DNS name 'localhost'")
					}

					if !localhostLocalDomainSeen {
						t.Errorf("expected self-signed certificate to have DNS name 'localhost.localdomain'")
					}
				}
			}

			if len(tlsConfig.Certificates[1].Certificate) < 1 {
				t.Errorf("expected provided certificate to be non-empty: %#v", tlsConfig.Certificates[1])
			}
		}
	}
}
