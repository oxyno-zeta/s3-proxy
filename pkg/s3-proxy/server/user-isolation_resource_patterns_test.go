//go:build integration

package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// Resource-layout integration tests for the common shared-storage shape:
// one admin and several clients, every client confined to their own folder
// via userIsolation. Two patterns are covered:
//
//   - "perUserPaths"  — one resource per client (path "/mount/<user>/*"),
//     plus an admin-only catch-all. This mirrors the layout that breaks
//     under transparent-injection userIsolation: the URL never contains
//     the username, so client uploads from the mount root never match
//     the per-user resource and fall through to the admin-only rule,
//     which returns 401.
//
//   - "flatPaths"     — one resource for "/mount/*" GET/PUT/HEAD covering
//     every authenticated user, and a separate "/mount/*" DELETE rule
//     restricted to admins. The proxy itself enforces per-user folder
//     confinement at the bucket-key level, so the resource list does
//     not need to know any usernames.
//
// These tests are infrastructure-agnostic: the shape applies to any
// shared-storage proxy deployment, not a specific environment.

// credsFor builds a basic-auth credential block for the given user names,
// using the convention "pw-<user>" for passwords throughout this file.
func credsFor(users []string) *config.ResourceBasic {
	out := make([]*config.BasicAuthUserConfig, 0, len(users))
	for _, u := range users {
		out = append(out, &config.BasicAuthUserConfig{
			User:     u,
			Password: &config.CredentialConfig{Value: "pw-" + u},
		})
	}

	return &config.ResourceBasic{Credentials: out}
}

// perUserPathsConfig builds a target where each client gets its own auth
// path "/mount/<client>/*" and admins also have a catch-all "/mount/*".
func perUserPathsConfig(
	s3URL, accessKey, secretAccessKey, region, bucket string,
	svrCfg *config.ServerConfig,
	clients, admins []string,
) *config.Config {
	allUsers := append([]string{}, clients...)
	allUsers = append(allUsers, admins...)

	resources := []*config.Resource{
		{
			Path:     "/mount/",
			Methods:  []string{"GET"},
			Provider: "provider1",
			Basic:    credsFor(allUsers),
		},
	}
	for _, u := range clients {
		resources = append(resources,
			&config.Resource{
				Path:     "/mount/" + u + "/*",
				Methods:  []string{"GET", "PUT"},
				Provider: "provider1",
				Basic:    credsFor(append([]string{u}, admins...)),
			},
			&config.Resource{
				Path:     "/mount/" + u + "/*",
				Methods:  []string{"DELETE"},
				Provider: "provider1",
				Basic:    credsFor(admins),
			},
		)
	}

	resources = append(resources, &config.Resource{
		Path:     "/mount/*",
		Methods:  []string{"GET", "PUT", "DELETE"},
		Provider: "provider1",
		Basic:    credsFor(admins),
	})

	return baseSharedStorageConfig(s3URL, accessKey, secretAccessKey, region, bucket, svrCfg, resources, admins)
}

// flatPathsConfig is the recommended layout under transparent-injection
// userIsolation: a single "/mount/*" rule for every authenticated user,
// and a separate admin-only DELETE rule. The proxy injects the user's
// identifier between the bucket prefix and the request path, so per-user
// paths in the resource list are unnecessary.
func flatPathsConfig(
	s3URL, accessKey, secretAccessKey, region, bucket string,
	svrCfg *config.ServerConfig,
	clients, admins []string,
) *config.Config {
	allUsers := append([]string{}, clients...)
	allUsers = append(allUsers, admins...)

	resources := []*config.Resource{
		{
			Path:     "/mount/",
			Methods:  []string{"GET"},
			Provider: "provider1",
			Basic:    credsFor(allUsers),
		},
		{
			Path:     "/mount/*",
			Methods:  []string{"GET", "PUT", "HEAD"},
			Provider: "provider1",
			Basic:    credsFor(allUsers),
		},
		{
			Path:     "/mount/*",
			Methods:  []string{"DELETE"},
			Provider: "provider1",
			Basic:    credsFor(admins),
		},
	}

	return baseSharedStorageConfig(s3URL, accessKey, secretAccessKey, region, bucket, svrCfg, resources, admins)
}

// baseSharedStorageConfig assembles a minimal Config with userIsolation
// enabled and basic-auth provider1, so the two pattern builders only have
// to differ in their resource list.
func baseSharedStorageConfig(
	s3URL, accessKey, secretAccessKey, region, bucket string,
	svrCfg *config.ServerConfig,
	resources []*config.Resource,
	admins []string,
) *config.Config {
	return &config.Config{
		Server:      svrCfg,
		ListTargets: &config.ListTargetsConfig{},
		Tracing:     &config.TracingConfig{},
		Templates:   testsDefaultGeneralTemplateConfig,
		AuthProviders: &config.AuthProviderConfig{
			Basic: map[string]*config.BasicAuthConfig{
				"provider1": {Realm: "realm1"},
			},
		},
		Targets: map[string]*config.TargetConfig{
			"target1": {
				Name: "target1",
				Bucket: &config.BucketConfig{
					Name: bucket, Prefix: "data/", Region: region, S3Endpoint: s3URL,
					Credentials: &config.BucketCredentialConfig{
						AccessKey: &config.CredentialConfig{Value: accessKey},
						SecretKey: &config.CredentialConfig{Value: secretAccessKey},
					},
					DisableSSL: true,
				},
				Mount:     &config.MountConfig{Path: []string{"/mount/"}},
				Resources: resources,
				Actions: &config.ActionsConfig{
					GET: &config.GetActionConfig{
						Enabled: true,
						Config: &config.GetActionConfigConfig{
							UserIsolation:       true,
							UserIsolationAdmins: admins,
						},
					},
					PUT:    &config.PutActionConfig{Enabled: true, Config: &config.PutActionConfigConfig{}},
					DELETE: &config.DeleteActionConfig{Enabled: true},
					HEAD:   &config.HeadActionConfig{Enabled: true},
				},
			},
		},
	}
}

// TestUserIsolation_PerUserResourcePaths_RootUploadDenied documents the
// failure mode of the per-user-paths layout under transparent-injection
// userIsolation: a non-admin uploading to the mount root URL hits the
// admin-only catch-all and is rejected at basic auth (401).
func TestUserIsolation_PerUserResourcePaths_RootUploadDenied(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"

	_, s3server, err := newIsolationFakeS3(accessKey, secretAccessKey, region, bucket)
	require.NoError(t, err)

	defer s3server.Close()

	svrCfg := defaultIsolationServerConfig()
	cfg := perUserPathsConfig(s3server.URL, accessKey, secretAccessKey, region, bucket, svrCfg,
		[]string{"alice"}, []string{"admin"})
	do := buildIsolationRouter(t, cfg)

	// Client uploads to mount root — what the React UI does when the user
	// has not navigated into a subfolder. URL is /mount/, no username.
	// The only resource that matches the URL is /mount/* (admin-only),
	// so basic auth fails alice's credentials and the proxy returns 401.
	w := do("PUT", "http://localhost/mount/", "alice", "pw-alice",
		multipartBody(t, "upload.txt", "alice-wrote-this"), multipartContentType())
	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"non-admin upload to /mount/ must 401 under per-user resource paths: "+
			"the only matching resource is the admin-only catch-all")

	// Same upload under /mount/<user>/ matches the per-user resource and
	// succeeds — this is the URL pattern the per-user layout requires.
	w2 := do("PUT", "http://localhost/mount/alice/", "alice", "pw-alice",
		multipartBody(t, "upload.txt", "alice-wrote-this"), multipartContentType())
	assert.Equal(t, http.StatusNoContent, w2.Code,
		"PUT to /mount/<user>/ matches the per-user resource and should succeed")
}

// TestUserIsolation_FlatResourcePaths_RootUploadAllowed validates the
// recommended layout: a single /mount/* rule covers every auth user.
// Transparent injection puts each non-admin's writes under their own
// folder regardless of URL shape, and a separate admin-only DELETE
// rule keeps destructive ops gated.
func TestUserIsolation_FlatResourcePaths_RootUploadAllowed(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"

	s3Client, s3server, err := newIsolationFakeS3(accessKey, secretAccessKey, region, bucket)
	require.NoError(t, err)

	defer s3server.Close()

	svrCfg := defaultIsolationServerConfig()
	cfg := flatPathsConfig(s3server.URL, accessKey, secretAccessKey, region, bucket, svrCfg,
		[]string{"alice"}, []string{"admin"})
	do := buildIsolationRouter(t, cfg)

	// Client uploads to /mount/ (no folder in URL). The flat /mount/*
	// rule passes auth and the proxy injects alice/ before the bucket key.
	w := do("PUT", "http://localhost/mount/", "alice", "pw-alice",
		multipartBody(t, "upload.txt", "alice-wrote-this"), multipartContentType())
	assert.Equal(t, http.StatusNoContent, w.Code,
		"PUT to /mount/ must succeed for any auth user under the flat layout")

	// Object lands at data/alice/upload.txt — the proxy did the prefixing,
	// alice never had to put "alice/" in the URL.
	out, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: new(bucket),
		Key:    new("data/alice/upload.txt"),
	})
	require.NoError(t, err, "uploaded object must land at data/alice/upload.txt")

	if err == nil {
		defer out.Body.Close()
	}

	// Client DELETE on the same path is rejected — the DELETE rule lists
	// admins only, so basic auth returns 401.
	w2 := do("DELETE", "http://localhost/mount/upload.txt", "alice", "pw-alice", "", "")
	assert.Equal(t, http.StatusUnauthorized, w2.Code,
		"client DELETE must 401 because the flat DELETE rule is admin-only")

	// Admin DELETE on alice's file via /mount/alice/ succeeds because admin
	// matches the rule and bypasses transparent injection (admin sees the
	// full bucket prefix).
	w3 := do("DELETE", "http://localhost/mount/alice/upload.txt", "admin", "pw-admin", "", "")
	assert.Equal(t, http.StatusNoContent, w3.Code,
		"admin DELETE on a client's file via /mount/<user>/ should succeed")
}

// TestUserIsolation_FlatResourcePaths_MultipleAdminsBypassIsolation exercises
// the full GET / PUT / DELETE matrix with two admins and two clients to make
// sure every name on userIsolationAdmins gets the admin treatment (not just
// the first), that clients stay confined when admins are present, and that
// admins can both write and delete inside any client folder.
func TestUserIsolation_FlatResourcePaths_MultipleAdminsBypassIsolation(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"

	s3Client, s3server, err := newIsolationFakeS3(accessKey, secretAccessKey, region, bucket)
	require.NoError(t, err)

	defer s3server.Close()

	// Seed a file in each client folder so admins have something to read /
	// delete and clients have a baseline to be confined against.
	for _, k := range []string{"data/alice/secret.txt", "data/bob/secret.txt"} {
		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Body:   strings.NewReader("seed-" + k),
			Bucket: new(bucket),
			Key:    new(k),
		})
		require.NoError(t, err)
	}

	svrCfg := defaultIsolationServerConfig()
	clients := []string{"alice", "bob"}
	admins := []string{"primary-admin", "secondary-admin"}
	cfg := flatPathsConfig(s3server.URL, accessKey, secretAccessKey, region, bucket, svrCfg, clients, admins)
	do := buildIsolationRouter(t, cfg)

	// --- Both admins bypass isolation on GET ---
	for _, a := range admins {
		w := do("GET", "http://localhost/mount/alice/secret.txt", a, "pw-"+a, "", "")
		assert.Equalf(t, http.StatusOK, w.Code, "admin %q GET on alice's file must succeed", a)
		assert.Containsf(t, w.Body.String(), "seed-data/alice/secret.txt",
			"admin %q must read alice's actual file (no folder injection for admins)", a)
	}

	// --- Both admins can PUT into another user's folder ---
	for _, a := range admins {
		w := do("PUT", "http://localhost/mount/bob/", a, "pw-"+a,
			multipartBody(t, a+"-note.txt", a+"-wrote-this"), multipartContentType())
		assert.Equalf(t, http.StatusNoContent, w.Code, "admin %q PUT into bob/ must succeed", a)

		out, getErr := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: new(bucket),
			Key:    new("data/bob/" + a + "-note.txt"),
		})
		require.NoErrorf(t, getErr, "admin %q upload must land at data/bob/%s-note.txt", a, a)

		out.Body.Close()
	}

	// --- Both admins can DELETE in another user's folder ---
	// First admin deletes the file the second admin uploaded; second admin
	// deletes the seeded alice/secret.txt. This proves DELETE bypass works
	// for both admins, not just the first name on the list.
	wDel1 := do("DELETE", "http://localhost/mount/bob/secondary-admin-note.txt", "primary-admin", "pw-primary-admin", "", "")
	assert.Equal(t, http.StatusNoContent, wDel1.Code,
		"primary-admin must be able to DELETE secondary-admin's file in bob/")

	_, getErr := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: new(bucket),
		Key:    new("data/bob/secondary-admin-note.txt"),
	})
	require.Error(t, getErr, "secondary-admin's file must be gone after primary-admin's DELETE")

	wDel2 := do("DELETE", "http://localhost/mount/alice/secret.txt", "secondary-admin", "pw-secondary-admin", "", "")
	assert.Equal(t, http.StatusNoContent, wDel2.Code,
		"secondary-admin must be able to DELETE alice's seeded file")

	_, getErr = s3Client.GetObject(&s3.GetObjectInput{
		Bucket: new(bucket),
		Key:    new("data/alice/secret.txt"),
	})
	require.Error(t, getErr, "alice's seeded file must be gone after secondary-admin's DELETE")

	// --- Clients stay confined even with multiple admins around ---
	// alice attempting to read bob's file: under transparent injection the
	// proxy rewrites the key to data/alice/bob/secret.txt (which does not
	// exist) — bob's real file is never reachable.
	w := do("GET", "http://localhost/mount/bob/secret.txt", "alice", "pw-alice", "", "")
	assert.Equal(t, http.StatusNotFound, w.Code,
		"alice must hit her own (empty) bob/secret.txt path, not bob's actual file")

	// alice DELETE is rejected: the DELETE rule lists admins only, alice's
	// credentials are not on the allowed list and basic auth returns 401.
	wAliceDel := do("DELETE", "http://localhost/mount/whatever.txt", "alice", "pw-alice", "", "")
	assert.Equal(t, http.StatusUnauthorized, wAliceDel.Code,
		"non-admin DELETE must 401 even when multiple admins are configured")

	// alice PUT lands in her own folder regardless of URL — same applies for
	// bob, confirming clients keep their per-user isolation when more than
	// one admin exists alongside them.
	for _, c := range clients {
		w := do("PUT", "http://localhost/mount/", c, "pw-"+c,
			multipartBody(t, c+"-own.txt", c+"-content"), multipartContentType())
		assert.Equalf(t, http.StatusNoContent, w.Code, "client %q PUT to /mount/ must succeed", c)

		out, getErr := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: new(bucket),
			Key:    new("data/" + c + "/" + c + "-own.txt"),
		})
		require.NoErrorf(t, getErr, "client %q upload must land at data/%s/%s-own.txt", c, c, c)

		out.Body.Close()
	}
}
