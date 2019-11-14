# Configuration

The configuration must be set in a YAML file located in `conf/` folder from the current working directory. The file name is `config.yaml`.
The full path is `conf/config.yaml`.

You can see a full example in the [Example section](#example)

## Main structure

| Key                   | Type                                            | Required | Default | Description                                                                                |
| --------------------- | ----------------------------------------------- | -------- | ------- | ------------------------------------------------------------------------------------------ |
| log                   | [LogConfiguration](#logconfiguration)           | No       | None    | Log configurations                                                                         |
| server                | [ServerConfiguration](#serverconfiguration)     | No       | None    | Server configurations                                                                      |
| internalServer        | [ServerConfiguration](#serverconfiguration)     | No       | None    | Internal Server configurations                                                             |
| template              | [TemplateConfiguration](#templateconfiguration) | No       | None    | Template configurations                                                                    |
| mainBucketPathSupport | Boolean                                         | No       | `false` | If only one bucket is in the list, use it as main url and don't mount it on /<BUCKET_NAME> |
| resources             | [[Resource]](#resource)                         | No       | None    | Resources declaration for path whitelist or specific authentication on paths               |
| targets               | [[TargetConfiguration]](#targetconfiguration)   | Yes      | None    | Targets configuration                                                                      |
| auth                  | AuthConfig                                      | No       | None    | Authentication configuration                                                               |

## LogConfiguration

| Key    | Type   | Required | Default | Description                                         |
| ------ | ------ | -------- | ------- | --------------------------------------------------- |
| level  | String | No       | `info`  | Log level                                           |
| format | String | No       | `json`  | Log format (available values are: `json` or `text`) |

## ServerConfiguration

| Key        | Type    | Required | Default | Description    |
| ---------- | ------- | -------- | ------- | -------------- |
| listenAddr | String  | No       | ""      | Listen Address |
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

| Key           | Type                                        | Required | Default | Description                                                                                              |
| ------------- | ------------------------------------------- | -------- | ------- | -------------------------------------------------------------------------------------------------------- |
| name          | String                                      | Yes      | None    | Target name. (This will used in urls and list of targets.)                                               |
| bucket        | [BucketConfiguration](#bucketconfiguration) | Yes      | None    | Bucket configuration                                                                                     |
| indexDocument | String                                      | No       | ""      | The index document name. If this document is found, get it instead of list folder. Example: `index.html` |

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

## AuthConfiguration

Note:

OIDC authentication will be used in priority if the 2 authentication configurations are set. Basic auth will be ignored in this case.

| Key   | Type                                              | Required | Default | Description              |
| ----- | ------------------------------------------------- | -------- | ------- | ------------------------ |
| basic | [BasicAuthConfiguration](#basicauthconfiguration) | No       | None    | Basic auth configuration |
| oidc  | [OIDCAuthConfiguration](#oidcauthconfiguration)   | No       | None    | OIDC Auth configuration  |

## OIDCAuthConfiguration

| Key                   | Type                                                      | Required | Default                          | Description                                                                                                                                           |
| --------------------- | --------------------------------------------------------- | -------- | -------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| clientID              | String                                                    | Yes      | None                             | Client ID                                                                                                                                             |
| clientSecret          | [CredentialConfiguration](#credentialconfiguration)       | No       | None                             | Client Secret                                                                                                                                         |
| issuerUrl             | String                                                    | Yes      | None                             | Issuer URL (example: https://fake.com/realm/fake-realm                                                                                                |
| redirectUrl           | String                                                    | Yes      | None                             | Redirect URL (this is the service url)                                                                                                                |
| scopes                | [String]                                                  | No       | `["openid", "profile", "email"]` | Scopes                                                                                                                                                |
| state                 | String                                                    | Yes      | None                             | Random string to have a secure connection with oidc provider                                                                                          |
| groupClaim            | String                                                    | No       | `groups`                         | Groups claim path in token (`groups` must be a list of strings containing user groups)                                                                |
| emailVerified         | Boolean                                                   | No       | `false`                          | Check that user email is verified in user token (field `email_verified`)                                                                              |
| cookieName            | String                                                    | No       | `oidc`                           | Cookie generated name                                                                                                                                 |
| cookieSecure          | Boolean                                                   | No       | `false`                          | Is the cookie secure ?                                                                                                                                |
| authorizationAccesses | [[OIDCAuthorizationAccesses]](#oidcauthorizationaccesses) | No       | None                             | Authorization accesses matrix by group or email. If not set, authenticated users will be authorized (no group or email validation will be performed). |

## OIDCAuthorizationAccesses

| Key   | Type   | Required               | Default | Description |
| ----- | ------ | ---------------------- | ------- | ----------- |
| group | String | Required without email | None    | Group name  |
| email | String | Required without group | None    | Email       |

## BasicAuthConfiguration

| Key         | Type                                                        | Required | Default | Description                          |
| ----------- | ----------------------------------------------------------- | -------- | ------- | ------------------------------------ |
| realm       | String                                                      | Yes      | None    | Basic Auth Realm                     |
| credentials | [[BasicAuthUserConfiguration]](#basicauthuserconfiguration) | No       | None    | List of authorized user and password |

## BasicAuthUserConfiguration

| Key      | Type                                                | Required | Default | Description   |
| -------- | --------------------------------------------------- | -------- | ------- | ------------- |
| user     | String                                              | Yes      | None    | User name     |
| password | [CredentialConfiguration](#credentialconfiguration) | Yes      | None    | User password |

## Resource

| Key       | Type                                              | Required                            | Default                           | Description                                                                                                         |
| --------- | ------------------------------------------------- | ----------------------------------- | --------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| path      | String                                            | Yes                                 | None                              | Path or matching path (e.g.: `/*`)                                                                                  |
| whiteList | Boolean                                           | Required without oidc or basic      | None                              | Is this path in white list ? E.g.: No authentication                                                                |
| oidc      | [ResourceOIDC](#resourceoidc)                     | Required without whitelist or oidc  | None                              | OIDC configuration authorization override. This part will override the default groups/email authorization accesses. |
| basic     | [BasicAuthConfiguration](#basicauthconfiguration) | Required without whitelist or basic | Basic auth configuration override |

# ResourceOIDC

| Key                   | Type                                                      | Required | Default | Description                                                                                                                                           |
| --------------------- | --------------------------------------------------------- | -------- | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| authorizationAccesses | [[OIDCAuthorizationAccesses]](#oidcauthorizationaccesses) | No       | None    | Authorization accesses matrix by group or email. If not set, authenticated users will be authorized (no group or email validation will be performed). |

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

# If only bucket is in the list, use it as main url and don't mount it on /<BUCKET_NAME>
# mainBucketPathSupport: true

# Authentication
# Note: OIDC is always preferred by default against basic authentication
# auth:
#   oidc:
#     clientID: client-id
#     clientSecret:
#       path: client-secret-in-file # client secret file
#     state: my-secret-state-key # do not use this in production ! put something random here
#     issuerUrl: https://issuer-url/
#     redirectUrl: http://localhost:8080/ # /auth/oidc/callback will be added automatically
#     scopes: # OIDC Scopes (defaults: oidc, email, profile)
#       - oidc
#       - email
#       - profile
#     groupClaim: groups # path in token
#     emailVerified: true # check email verified field from token
#     authorizationAccesses: # Authorization accesses : groups or email
#       - group: devops_users
#   basic:
#     realm: My Basic Auth Realm
#     credentials:
#       - user: user1
#         password:
#           path: password1-in-file

# Resources declaration
# resources:
#   - path: /
#     whiteList: true
#   - path: /devops_internal_doc/*
#     whiteList: false # Force not white list to use default global authentication system
#   - path: /specific_doc
#     oidc:
#       authorizationAccesses: # Authorization accesses : groups or email
#         - group: specific_users

# Targets
targets:
  - name: first-bucket
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
    # indexDocument: index.html
```
