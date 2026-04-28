# User Isolation

## What is user isolation ?

User isolation is a per-target feature that transparently confines
every authenticated user to their own folder inside the configured
S3 prefix. Users never see their own identifier in a URL: the proxy
inserts `<identifier>/` after the bucket prefix on the way to S3
and strips it again on the way back.

The identifier is whatever `GenericUser.GetIdentifier()` returns
for the authenticated user:

- Basic auth: the username.
- OIDC: `preferred_username` if the IdP emits it, otherwise the
  email address.
- Header auth: the configured username header if present, otherwise
  the email header.

Using the identifier (rather than the username) means OIDC users
without a `preferred_username` claim still get a stable, non-empty
folder.

## General description

The feature is enabled per target, under the GET action
configuration. When enabled, every incoming request is rewritten
server-side so a non-admin user is structurally confined to their
own folder.

This is stronger than a post-hoc permission check. It is
structurally impossible for a non-admin user to construct a URL
that reaches another user's folder, because the proxy rewrites the
S3 key before it leaves the process.

The feature applies uniformly to `GET`, `HEAD`, `PUT` and `DELETE`.

Administrators listed in `userIsolationAdmins` bypass the rewrite
and see the full bucket prefix. They are the only identities that
can address another user's folder through the proxy.

## How this is working ?

Consider a target with bucket prefix `internal/` mounted at
`/files/` and user isolation enabled.

- User `bob` calls `GET /files/report.pdf`.
- The proxy computes the S3 key as `internal/bob/report.pdf`.
- S3 returns the object; the response is streamed back unchanged.

The same transparent rewrite applies to `HEAD`, `PUT` and `DELETE`.
On listings, entries are returned with their `path` rewritten so
the identifier never appears in the UI (e.g. `/files/report.pdf`,
not `/files/bob/report.pdf`).

If isolation is enabled but no user is authenticated, the request
is rejected with `403 Forbidden` to prevent unrestricted access.

For admins listed in `userIsolationAdmins` the rewrite is skipped:

- An admin listing `/files/` sees every user folder (`alice/`,
  `bob/`, `charlie/`, ...).
- `GET /files/bob/report.pdf` reaches `internal/bob/report.pdf`.
- PUT/DELETE for admins are not prefixed either. They operate on
  the bucket prefix directly.

Admins never have a dedicated folder created automatically. In
normal operation only real users own folders under the prefix.

Key rewrite rules (see [Key Rewrite](./key-rewrite.md)) run after
the username injection and receive the already-prefixed key.

## For which situations ?

This feature is useful when a single bucket is shared between
multiple authenticated users and each user must only see and
manipulate their own files, without needing separate buckets or
per-user credentials.

Typical cases:

- A multi-tenant upload area where each tenant receives their own
  login and must be confined to their own folder.
- A self-service report drop where users read and overwrite only
  files under their own namespace, while an operator account
  retains full visibility to support or audit.
- Any workflow where exposing the user identifier in URLs is
  unwanted, for example to keep shared URLs neutral or to prevent
  trivial enumeration of other users' folders.

## Examples

### Minimal configuration

```yaml
targets:
  shared:
    bucket:
      name: my-bucket
      prefix: internal/
      # ...credentials...
    mount:
      path:
        - /files/
    resources:
      - path: /files/*
        methods: [GET, PUT, DELETE, HEAD]
        provider: provider1
        basic:
          credentials:
            - user: alice
              password: { value: alicepw }
            - user: bob
              password: { value: bobpw }
            - user: admin
              password: { value: adminpw }
    actions:
      GET:
        enabled: true
        config:
          userIsolation: true
          userIsolationAdmins:
            - admin
      PUT:
        enabled: true
      DELETE:
        enabled: true
```

### Notes

- Isolation requires an authenticated user. The target must declare
  at least one resource with basic, OIDC or header authentication;
  configurations that enable `userIsolation` without an auth
  resource are rejected at startup.
- Keys are built as `<bucketPrefix><identifier>/<requestPath>`. If
  your identifiers can collide with real folder names inside the
  bucket prefix, use a dedicated prefix for the isolated target.
- For OIDC, `userIsolationAdmins` matches against the same
  identifier the proxy uses for the folder — `preferred_username`
  if present, otherwise the email. List the same value you expect
  the IdP to emit.
