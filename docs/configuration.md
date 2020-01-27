# Configuration

The configuration must be set in a YAML file located in `conf/` folder from the current working directory. The file name is `config.yaml`.
The full path is `conf/config.yaml`.

You can see a full example in the [Example section](#example)

## Main structure

| Key            | Type                                                      | Required | Default | Description                            |
| -------------- | --------------------------------------------------------- | -------- | ------- | -------------------------------------- |
| log            | [LogConfiguration](#logconfiguration)                     | No       | None    | Log configurations                     |
| server         | [ServerConfiguration](#serverconfiguration)               | No       | None    | Server configurations                  |
| internalServer | [ServerConfiguration](#serverconfiguration)               | No       | None    | Internal Server configurations         |
| template       | [TemplateConfiguration](#templateconfiguration)           | No       | None    | Template configurations                |
| targets        | [[TargetConfiguration]](#targetconfiguration)             | Yes      | None    | Targets configuration                  |
| authProviders  | [AuthProvidersConfiguration](#authProvidersconfiguration) | No       | None    | Authentication providers configuration |
| listTargets    | [ListTargetsConfiguration](#listtargetsconfiguration)     | No       | None    | List targets feature configuration     |

## LogConfiguration

| Key    | Type   | Required | Default | Description                                         |
| ------ | ------ | -------- | ------- | --------------------------------------------------- |
| level  | String | No       | `info`  | Log level                                           |
| format | String | No       | `json`  | Log format (available values are: `json` or `text`) |

## ServerConfiguration

| Key        | Type    | Required | Default | Description    |
| ---------- | ------- | -------- | ------- | -------------- |
| listenAddr | String  | No       | `""`    | Listen Address |
| port       | Integer | No       | `8080`  | Listening Port |

## TemplateConfiguration

| Key                 | Type   | Required | Default                               | Description                         |
| ------------------- | ------ | -------- | ------------------------------------- | ----------------------------------- |
| targetList          | String | No       | `templates/target-list.tpl`           | Target list template path           |
| folderList          | String | No       | `templates/folder-list.tpl`           | Folder list template path           |
| notFound            | String | No       | `templates/not-found.tpl`             | Not found template path             |
| unauthorized        | String | No       | `templates/unauthorized.tpl`          | Unauthorized template path          |
| forbidden           | String | No       | `templates/forbidden.tpl`             | Forbidden template path             |
| badRequest          | String | No       | `templates/bad-request.tpl`           | Bad Request template path           |
| internalServerError | String | No       | `templates/internal-server-error.tpl` | Internal server error template path |

## TargetConfiguration

| Key           | Type                                          | Required | Default            | Description                                                                                              |
| ------------- | --------------------------------------------- | -------- | ------------------ | -------------------------------------------------------------------------------------------------------- |
| name          | String                                        | Yes      | None               | Target name. (This will used in urls and list of targets.)                                               |
| bucket        | [BucketConfiguration](#bucketconfiguration)   | Yes      | None               | Bucket configuration                                                                                     |
| indexDocument | String                                        | No       | `""`               | The index document name. If this document is found, get it instead of list folder. Example: `index.html` |
| resources     | [[Resource]](#resource)                       | No       | None               | Resources declaration for path whitelist or specific authentication on path list                         |
| mount         | [MountConfiguration](#mountconfiguration)     | Yes      | None               | Mount point configuration                                                                                |
| actions       | [ActionsConfiguration](#actionsconfiguration) | No       | GET action enabled | Actions allowed on target (GET, PUT or DELETE)                                                           |

## ActionsConfiguration

| Key    | Type                                                    | Required | Default | Description                                        |
| ------ | ------------------------------------------------------- | -------- | ------- | -------------------------------------------------- |
| GET    | [GetActionConfiguration](#getactionconfiguration)       | No       | None    | Action configuration for GET requests on target    |
| PUT    | [PutActionConfiguration](#putactionconfiguration)       | No       | None    | Action configuration for PUT requests on target    |
| DELETE | [DeleteActionConfiguration](#deleteactionconfiguration) | No       | None    | Action configuration for DELETE requests on target |

## GetActionConfiguration

| Key     | Type    | Required | Default | Description             |
| ------- | ------- | -------- | ------- | ----------------------- |
| enabled | Boolean | No       | `false` | Will allow GET requests |

## PutActionConfiguration

| Key     | Type                                                          | Required | Default | Description                    |
| ------- | ------------------------------------------------------------- | -------- | ------- | ------------------------------ |
| enabled | Boolean                                                       | No       | `false` | Will allow PUT requests        |
| config  | [PutActionConfigConfiguration](#putactionconfigconfiguration) | No       | None    | Configuration for PUT requests |

## PutActionConfigConfiguration

| Key           | Type              | Required | Default | Description                                                                                                                                                                                                                        |
| ------------- | ----------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| metadata      | Map[String]String | No       | None    | Metadata key/values that will be put on S3 objects                                                                                                                                                                                 |
| storageClass  | String            | No       | `""`    | Storage class that will be used for uploaded objects. See storage class here: [https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html](https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html) |
| allowOverride | Boolean           | No       | `false` | Will allow override objects if enabled                                                                                                                                                                                             |

## DeleteActionConfiguration

| Key     | Type    | Required | Default | Description                |
| ------- | ------- | -------- | ------- | -------------------------- |
| enabled | Boolean | No       | `false` | Will allow DELETE requests |

## BucketConfiguration

| Key         | Type                                                            | Required | Default     | Description                              |
| ----------- | --------------------------------------------------------------- | -------- | ----------- | ---------------------------------------- |
| name        | String                                                          | Yes      | None        | Bucket name in S3 provider               |
| prefix      | String                                                          | No       | None        | Bucket prefix                            |
| region      | String                                                          | No       | `us-east-1` | Bucket region                            |
| s3Endpoint  | String                                                          | No       | None        | Custom S3 Endpoint for non AWS S3 bucket |
| credentials | [BucketCredentialConfiguration](#bucketcredentialconfiguration) | No       | None        | Credentials to access S3 bucket          |

## BucketCredentialConfiguration

| Key       | Type                                                | Required | Default | Description          |
| --------- | --------------------------------------------------- | -------- | ------- | -------------------- |
| accessKey | [CredentialConfiguration](#credentialconfiguration) | No       | None    | S3 Access Key ID     |
| secretKey | [CredentialConfiguration](#credentialconfiguration) | No       | None    | S3 Secret Access Key |

## CredentialConfiguration

| Key   | Type   | Required                           | Default | Description                                         |
| ----- | ------ | ---------------------------------- | ------- | --------------------------------------------------- |
| path  | String | Only if env and value are not set  | None    | File path contains credential in                    |
| env   | String | Only if path and value are not set | None    | Environment variable name to use to load credential |
| value | String | Only if path and env are not set   | None    | Credential value directly (Not recommended)         |

## AuthProvidersConfiguration

| Key   | Type                                                         | Required | Default | Description                                       |
| ----- | ------------------------------------------------------------ | -------- | ------- | ------------------------------------------------- |
| basic | [map[string]BasicAuthConfiguration](#basicauthconfiguration) | No       | None    | Basic Auth configuration and key as provider name |
| oidc  | [map[string]OIDCAuthConfiguration](#oidcauthconfiguration)   | No       | None    | OIDC Auth configuration and key as provider name  |

## OIDCAuthConfiguration

| Key           | Type                                                | Required | Default                          | Description                                                                                                    |
| ------------- | --------------------------------------------------- | -------- | -------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| clientID      | String                                              | Yes      | None                             | Client ID                                                                                                      |
| clientSecret  | [CredentialConfiguration](#credentialconfiguration) | No       | None                             | Client Secret                                                                                                  |
| issuerUrl     | String                                              | Yes      | None                             | Issuer URL (example: https://fake.com/realm/fake-realm                                                         |
| redirectUrl   | String                                              | Yes      | None                             | Redirect URL (this is the service url)                                                                         |
| scopes        | [String]                                            | No       | `["openid", "profile", "email"]` | Scopes                                                                                                         |
| state         | String                                              | Yes      | None                             | Random string to have a secure connection with oidc provider                                                   |
| groupClaim    | String                                              | No       | `groups`                         | Groups claim path in token (`groups` must be a list of strings containing user groups)                         |
| emailVerified | Boolean                                             | No       | `false`                          | Check that user email is verified in user token (field `email_verified`)                                       |
| cookieName    | String                                              | No       | `oidc`                           | Cookie generated name                                                                                          |
| cookieSecure  | Boolean                                             | No       | `false`                          | Is the cookie generated secure ?                                                                               |
| loginPath     | String                                              | No       | `""`                             | Override login path for authentication. If not defined, `/auth/PROVIDER_NAME` will be used                     |
| callbackPath  | String                                              | No       | `""`                             | Override callback path for authentication callback. If not defined,`/auth/PROVIDER_NAME/callback` will be used |

## BasicAuthConfiguration

| Key   | Type   | Required | Default | Description      |
| ----- | ------ | -------- | ------- | ---------------- |
| realm | String | Yes      | None    | Basic Auth Realm |

## Resource

| Key       | Type                            | Required                            | Default | Description                                                  |
| --------- | ------------------------------- | ----------------------------------- | ------- | ------------------------------------------------------------ |
| path      | String                          | Yes                                 | None    | Path or matching path (e.g.: `/*`)                           |
| methods   | [String]                        | No                                  | `[GET]` | HTTP methods allowed (Allowed values `GET`, `PUT`, `DELETE`) |
| whiteList | Boolean                         | Required without oidc or basic      | None    | Is this path in white list ? E.g.: No authentication         |
| oidc      | [ResourceOIDC](#resourceoidc)   | Required without whitelist or oidc  | None    | OIDC configuration authorization                             |
| basic     | [ResourceBasic](#resourcebasic) | Required without whitelist or basic | None    | Basic auth configuration                                     |

# ResourceOIDC

| Key                   | Type                                                      | Required | Default | Description                                                                                                                                           |
| --------------------- | --------------------------------------------------------- | -------- | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| authorizationAccesses | [[OIDCAuthorizationAccesses]](#oidcauthorizationaccesses) | No       | None    | Authorization accesses matrix by group or email. If not set, authenticated users will be authorized (no group or email validation will be performed). |

## OIDCAuthorizationAccesses

| Key   | Type   | Required               | Default | Description |
| ----- | ------ | ---------------------- | ------- | ----------- |
| group | String | Required without email | None    | Group name  |
| email | String | Required without group | None    | Email       |

## ResourceBasic

| Key         | Type                                                        | Required | Default | Description                          |
| ----------- | ----------------------------------------------------------- | -------- | ------- | ------------------------------------ |
| credentials | [[BasicAuthUserConfiguration]](#basicauthuserconfiguration) | Yes      | None    | List of authorized user and password |

## BasicAuthUserConfiguration

| Key      | Type                                                | Required | Default | Description   |
| -------- | --------------------------------------------------- | -------- | ------- | ------------- |
| user     | String                                              | Yes      | None    | User name     |
| password | [CredentialConfiguration](#credentialconfiguration) | Yes      | None    | User password |

## MountConfiguration

| Key  | Type     | Required | Default | Description                                            |
| ---- | -------- | -------- | ------- | ------------------------------------------------------ |
| host | String   | No       | `""`    | Host domain requested (eg: localhost:888 or google.fr) |
| path | [String] | Yes      | None    | A path list for mounting point                         |

## ListTargetsConfiguration

| Key      | Type                                      | Required | Default | Description                                                                 |
| -------- | ----------------------------------------- | -------- | ------- | --------------------------------------------------------------------------- |
| enabled  | Boolean                                   | Yes      | None    | To enable the list targets feature                                          |
| mount    | [MountConfiguration](#mountconfiguration) | Yes      | None    | Mount point configuration                                                   |
| resource | [Resource](#resource)                     | No       | None    | Resources declaration for path whitelist or specific authentication on path |

## Example

```yaml
# Log configuration
log:
  # Log level
  level: info
  # Log format
  format: text

# Server configurations
# server:
#   listenAddr: ""
#   port: 8080

# Template configurations
# template:
#   badRequest: templates/bad-request.tpl
#   folderList: templates/folder-list.tpl
#   forbidden: templates/forbidden.tpl
#   internalServerError: templates/internal-server-error.tpl
#   notFound: templates/not-found.tpl
#   targetList: templates/target-list.tpl
#   unauthorized: templates/unauthorized.tpl

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
#       scopes: # OIDC Scopes (defaults: oidc, email, profile)
#         - oidc
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
#     # Whitelist
#     whitelist: false
#     # A authentication provider declared in section before, here is the key name
#     provider: provider1
#     # OIDC section for access filter
#     oidc:
#       # NOTE: This list can be empty ([]) for authentication only and no group filter
#       authorizationAccesses: # Authorization accesses : groups or email
#         - group: devops_users
#     # Basic authentication section
#     basic:
#       credentials:
#         - user: user1
#           password:
#             path: password1-in-file

# Targets
targets:
  - name: first-bucket
    # ## Mount point
    # mount:
    #   path:
    #     - /
    #   # A specific host can be added for filtering. Otherwise, all hosts will be accepted
    #   # host: localhost:8080
    # ## Resources declaration
    # resources:
    #   # A Path must be declared for a resource filtering (a wildcard can be added to match every sub path)
    #   - path: /
    #     # Whitelist
    #     whiteList: true
    #     # A Path must be declared for a resource filtering (a wildcard can be added to match every sub path)
    #   - path: /specific_doc/*
    #     # A authentication provider declared in section before, here is the key name
    #     provider: provider1
    #     # OIDC section for access filter
    #     oidc:
    #       # NOTE: This list can be empty ([]) for authentication only and no group filter
    #       authorizationAccesses: # Authorization accesses : groups or email
    #         - group: specific_users
    #     # A Path must be declared for a resource filtering (a wildcard can be added to match every sub path)
    #   - path: /directory1/*
    #     # A authentication provider declared in section before, here is the key name
    #     provider: provider1
    #     # Basic authentication section
    #     basic:
    #       credentials:
    #         - user: user1
    #           password:
    #             path: password1-in-file
    # ## Index document to display if exists in folder
    # indexDocument: index.html
    # ## Actions
    # actions:
    #   # Action for GET requests on target
    #   GET:
    #     # Will allow GET requests
    #     enabled: true
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
    ## Bucket configuration
    bucket:
      name: super-bucket
      prefix:
      region: eu-west-1
      s3Endpoint:
      # credentials:
      #   accessKey:
      #     env: AWS_ACCESS_KEY_ID
      #   secretKey:
      #     path: secret_key_file
```
