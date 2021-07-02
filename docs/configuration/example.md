# Example

<!-- prettier-ignore-start -->
!!! Note
  The following example is a full file example. But this can be split in multiple files, it will be merged by S3-Proxy automatically.
<!-- prettier-ignore-end -->

Here is a full example of a configuration file:

```yaml
# Log configuration
log:
  # Log level
  level: info
  # Log format
  format: text
  # Log file path
  # filePath:

# Server configurations
# server:
#   listenAddr: ""
#   port: 8080
#   # Compress options
#   compress:
#     enabled: true
#     # Compression level
#     # level: 5
#     # Types
#     # types:
#     #   - text/html
#     #   - text/css
#     #   - text/plain
#     #   - text/javascript
#     #   - application/javascript
#     #   - application/x-javascript
#     #   - application/json
#     #   - application/atom+xml
#     #   - application/rss+xml
#     #   - image/svg+xml
#   # CORS configuration
#   cors:
#     # Enabled
#     enabled: false
#     # Allow all traffic
#     allowAll: true
#     # Allow Origins
#     # Example: https://fake.com
#     allowOrigins: []
#     # Allow HTTP Methods
#     allowMethods: []
#     # Allow Headers
#     allowHeaders: []
#     # Expose Headers
#     exposeHeaders: []
#     # Max age
#     # 300 is the maximum value not ignored by any of major browsers
#     # Source: https://github.com/go-chi/cors
#     maxAge: 0
#     # Allow credentials
#     allowCredentials: false
#     # Run debug
#     debug: false
#     # OPTIONS method Passthrough
#     optionsPassthrough: false
#   # Cache configuration
#   cache:
#     # Force no cache headers on all responses
#     noCacheEnabled: true
#     # Expires header value
#     expires:
#     # Cache-control header value
#     cacheControl:
#     # Pragma header value
#     pragma:
#     # X-Accel-Expires header value
#     xAccelExpires:

# Template configurations
# templates:
#   helpers:
#     - templates/_helpers.tpl
#   targetList:
#     path: templates/target-list.tpl
#     headers:
#       Content-Type: '{{ template "main.headers.contentType" . }}'
#   folderList:
#     path: templates/folder-list.tpl
#     headers:
#       Content-Type: '{{ template "main.headers.contentType" . }}'
#   badRequestError:
#     path: templates/bad-request-error.tpl
#     headers:
#       Content-Type: '{{ template "main.headers.contentType" . }}'
#   forbiddenError:
#     path: templates/forbidden-error.tpl
#     headers:
#       Content-Type: '{{ template "main.headers.contentType" . }}'
#   internalServerError:
#     path: templates/internal-server-error.tpl
#     headers:
#       Content-Type: '{{ template "main.headers.contentType" . }}'
#   notFoundError:
#     path: templates/not-found-error.tpl
#     headers:
#       Content-Type: '{{ template "main.headers.contentType" . }}'
#   unauthorizedError:
#     path: templates/unauthorized-error.tpl
#     headers:
#       Content-Type: '{{ template "main.headers.contentType" . }}'

# Authentication Providers
# authProviders:
#   oidc:
#     provider1:
#       clientID: client-id
#       clientSecret:
#         path: client-secret-in-file # client secret file
#       state: my-secret-state-key # do not use this in production ! put something random here
#       issuerUrl: https://issuer-url/
#       redirectUrl: http://localhost:8080/ # /auth/oidc/callback will be added automatically
#       scopes: # OIDC Scopes (defaults: openid, email, profile)
#         - openid
#         - email
#         - profile
#       groupClaim: groups # path in token
#       # cookieSecure: true # Is the cookie generated secure ?
#       # cookieName: oidc # Cookie generated name
#       emailVerified: true # check email verified field from token
#       # loginPath: /auth/provider1 # Override login path dynamically generated from provider key
#       # callbackPath: /auth/provider1/callback # Override callback path dynamically generated from provider key
#   basic:
#     provider2:
#       realm: My Basic Auth Realm

# List targets feature
# This will generate a webpage with list of targets with links using targetList template
# listTargets:
#   # To enable the list targets feature
#   enabled: false
#   ## Mount point
#   mount:
#     path:
#       - /
#     # A specific host can be added for filtering. Otherwise, all hosts will be accepted
#     # host: localhost:8080
#   ## Resource configuration
#   resource:
#     # A Path must be declared for a resource filtering
#     path: /
#     # HTTP Methods authorized (Must be in GET, PUT or DELETE)
#     methods:
#       - GET
#       - PUT
#       - DELETE
#     # Whitelist
#     whitelist: false
#     # A authentication provider declared in section before, here is the key name
#     provider: provider1
#     # OIDC section for access filter
#     oidc:
#       # NOTE: This list can be empty ([]) for authentication only and no group filter
#       authorizationAccesses: # Authorization accesses : groups or email or regexp
#         - group: devops_users
#     # Basic authentication section
#     basic:
#       credentials:
#         - user: user1
#           password:
#             path: password1-in-file

# Targets map
targets:
  first-bucket:
    ## Mount point
    mount:
      path:
        - /
      # A specific host can be added for filtering. Otherwise, all hosts will be accepted
      # host: localhost:8080
    # ## Resources declaration
    # ## WARNING: Think about all path that you want to protect. At the end of the list, you should add a resource filter for /* otherwise, it will be public.
    # resources:
    #   # A Path must be declared for a resource filtering (a wildcard can be added to match every sub path)
    #   - path: /
    #     # Whitelist
    #     whiteList: true
    #     # A Path must be declared for a resource filtering (a wildcard can be added to match every sub path)
    #   - path: /specific_doc/*
    #     # HTTP Methods authorized (Must be in GET, PUT or DELETE)
    #     methods:
    #       - GET
    #       - PUT
    #       - DELETE
    #     # A authentication provider declared in section before, here is the key name
    #     provider: provider1
    #     # OIDC section for access filter
    #     oidc:
    #       # NOTE: This list can be empty ([]) for authentication only and no group filter
    #       authorizationAccesses: # Authorization accesses : groups or email or regexp
    #         - group: specific_users
    #     # A Path must be declared for a resource filtering (a wildcard can be added to match every sub path)
    #   - path: /directory1/*
    #     # HTTP Methods authorized (Must be in GET, PUT or DELETE)
    #     methods:
    #       - GET
    #       - PUT
    #       - DELETE
    #     # A authentication provider declared in section before, here is the key name
    #     provider: provider1
    #     # Basic authentication section
    #     basic:
    #       credentials:
    #         - user: user1
    #           password:
    #             path: password1-in-file
    #     # A Path must be declared for a resource filtering (a wildcard can be added to match every sub path)
    #   - path: /opa-protected/*
    #     # OIDC section for access filter
    #     oidc:
    #       # Authorization through OPA server configuration
    #       authorizationOPAServer:
    #         # OPA server url with data path
    #         url: http://localhost:8181/v1/data/example/authz/allowed
    # ## Actions
    # actions:
    #   # Action for GET requests on target
    #   GET:
    #     # Will allow GET requests
    #     enabled: true
    #     # Configuration for GET requests
    #     config:
    #       # Redirect with trailing slash when a file isn't found
    #       redirectWithTrailingSlashForNotFoundFile: true
    #       # Index document to display if exists in folder
    #       indexDocument: index.html
    #       # Allow to add headers to streamed files (can be templated)
    #       streamedFileHeaders: {}
    #   # Action for PUT requests on target
    #   PUT:
    #     # Will allow PUT requests
    #     enabled: true
    #     # Configuration for PUT requests
    #     config:
    #       # Metadata key/values that will be put on S3 objects
    #       metadata:
    #         key: value
    #       # Storage class that will be used for uploaded objects
    #       # See storage class here: https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html
    #       storageClass: STANDARD # GLACIER, ...
    #       # Will allow override objects if enabled
    #       allowOverride: false
    #   # Action for DELETE requests on target
    #   DELETE:
    #     # Will allow DELETE requests
    #     enabled: true
    # # Key rewrite list
    # # This will allow to rewrite keys before doing any requests to S3
    # # For more information about how this works, see in the documentation.
    # keyRewriteList:
    #   - # Source represents a Regexp (golang format with group naming support)
    #     source: ^/(?P<one>\w+)/(?P<two>\w+)/(?P<three>\w+)?$
    #     # Target represents the template of the new key that will be used
    #     target: /$two/$one/$three/$one/
    ## Target custom templates
    # templates:
    #   # Helpers
    #   helpers:
    #   - inBucket: false
    #     path: ""
    #   # Folder list template
    #   folderList:
    #     inBucket: false
    #     path: ""
    #     headers: {}
    #   # Not found error template
    #   notFoundError:
    #     inBucket: false
    #     path: ""
    #     headers: {}
    #   # Internal server error template
    #   internalServerError:
    #     inBucket: false
    #     path: ""
    #     headers: {}
    #   # Forbidden error template
    #   forbiddenError:
    #     inBucket: false
    #     path: ""
    #     headers: {}
    #   # Unauthorized error template
    #   unauthorizedError:
    #     inBucket: false
    #     path: ""
    #     headers: {}
    #   # Bad Request error template
    #   badRequestError:
    #     inBucket: false
    #     path: ""
    #     headers: {}
    ## Bucket configuration
    bucket:
      name: super-bucket
      prefix:
      region: eu-west-1
      s3Endpoint:
      disableSSL: false
      # s3ListMaxKeys: 1000
      # credentials:
      #   accessKey:
      #     env: AWS_ACCESS_KEY_ID
      #   secretKey:
      #     path: secret_key_file
```
