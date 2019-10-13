# Configuration

The configuration must be set in a YAML file located in `conf/` folder from the current working directory. The file name is `config.yaml`.
The full path is `conf/config.yaml`.

## Main structure

| Key                   | Type                                            | Default | Description                                                                                |
| --------------------- | ----------------------------------------------- | ------- | ------------------------------------------------------------------------------------------ |
| log                   | [LogConfiguration](#logconfiguration)           | None    | Log configurations                                                                         |
| server                | [ServerConfiguration](#serverconfiguration)     | None    | Server configurations                                                                      |
| internalServer        | [ServerConfiguration](#serverconfiguration)     | None    | Internal Server configurations                                                             |
| template              | [TemplateConfiguration](#templateconfiguration) | None    | Template configurations                                                                    |
| mainBucketPathSupport | Boolean                                         | `false` | If only one bucket is in the list, use it as main url and don't mount it on /<BUCKET_NAME> |
| targets               | [[TargetConfiguration]](#targetconfiguration)   | None    | Targets configuration                                                                      |
| auth                  | AuthConfig                                      | None    | Authentication configuration                                                               |

## LogConfiguration

| Key    | Type   | Default | Description                                         |
| ------ | ------ | ------- | --------------------------------------------------- |
| level  | String | `info`  | Log level                                           |
| format | String | `json`  | Log format (available values are: `json` or `text`) |

## ServerConfiguration

| Key        | Type    | Default | Description    |
| ---------- | ------- | ------- | -------------- |
| listenAddr | String  | ""      | Listen Address |
| port       | Integer | `8080`  | Listening Port |

## TemplateConfiguration

| Key                 | Type   | Default                               | Description                         |
| ------------------- | ------ | ------------------------------------- | ----------------------------------- |
| targetList          | String | `templates/target-list.tpl`           | Target list template path           |
| folderList          | String | `templates/folder-list.tpl`           | Folder list template path           |
| notFound            | String | `templates/not-found.tpl`             | Not found template path             |
| unauthorized        | String | `templates/unauthorized.tpl`          | Unauthorized template path          |
| forbidden           | String | `templates/forbidden.tpl`             | Forbidden template path             |
| badRequest          | String | `templates/bad-request.tpl`           | Bad Request template path           |
| internalServerError | String | `templates/internal-server-error.tpl` | Internal server error template path |

## TargetConfiguration

| Key           | Type                                        | Default | Description                                                                        |
| ------------- | ------------------------------------------- | ------- | ---------------------------------------------------------------------------------- |
| name          | String                                      | None    | Target name. (This will used in urls and list of targets.)                         |
| bucket        | [BucketConfiguration](#bucketconfiguration) | None    | Bucket configuration                                                               |
| indexDocument | String                                      | ""      | The index document name. If this document is found, get it instead of list folder. |

## BucketConfiguration

| Key         | Type                                                            | Default     | Description                              |
| ----------- | --------------------------------------------------------------- | ----------- | ---------------------------------------- |
| name        | String                                                          | None        | Bucket name in S3 provider               |
| prefix      | String                                                          | None        | Bucket prefix                            |
| region      | String                                                          | `us-east-1` | Bucket region                            |
| s3Endpoint  | String                                                          | None        | Custom S3 Endpoint for non AWS S3 bucket |
| credentials | [BucketCredentialConfiguration](#bucketcredentialconfiguration) | None        | Credentials to access S3 bucket          |

## BucketCredentialConfiguration

| Key       | Type                                                | Default | Description          |
| --------- | --------------------------------------------------- | ------- | -------------------- |
| accessKey | [CredentialConfiguration](#credentialconfiguration) | None    | S3 Access Key ID     |
| secretKey | [CredentialConfiguration](#credentialconfiguration) | None    | S3 Secret Access Key |

## CredentialConfiguration

| Key   | Type   | Default | Description                                         |
| ----- | ------ | ------- | --------------------------------------------------- |
| path  | String | None    | File path contains credential in                    |
| env   | String | None    | Environment variable name to use to load credential |
| value | String | None    | Credential value directly (Not recommended)         |

## AuthConfiguration

| Key   | Type                                              | Default | Description              |
| ----- | ------------------------------------------------- | ------- | ------------------------ |
| basic | [BasicAuthConfiguration](#basicauthconfiguration) | None    | Basic auth configuration |
| oidc  | [OIDCAuthConfiguration](#oidcauthconfiguration)   | None    | OIDC Auth configuration  |

## OIDCAuthConfiguration

| Key                   | Type                                                      | Default                          | Description                                                                            |
| --------------------- | --------------------------------------------------------- | -------------------------------- | -------------------------------------------------------------------------------------- |
| clientID              | String                                                    | ""                               | Client ID                                                                              |
| clientSecret          | [CredentialConfiguration](#credentialconfiguration)       | ""                               | Client Secret                                                                          |
| issuerUrl             | String                                                    | ""                               | Issuer URL (example: https://fake.com/realm/fake-realm                                 |
| redirectUrl           | String                                                    | ""                               | Redirect URL (this is the service url)                                                 |
| scopes                | [String]                                                  | `["openid", "profile", "email"]` | Scopes                                                                                 |
| state                 | String                                                    | ""                               | Random string to have a secure connection with oidc provider                           |
| groupClaim            | String                                                    | `groups`                         | Groups claim path in token (`groups` must be a list of strings containing user groups) |
| emailVerified         | Boolean                                                   | `false`                          | Check that user email is verified in user token (field `email_verified`)               |
| cookieName            | String                                                    | `oidc`                           | Cookie generated name                                                                  |
| cookieSecure          | Boolean                                                   | `false`                          | Is the cookie secure ?                                                                 |
| authorizationAccesses | [[OIDCAuthorizationAccesses]](#oidcauthorizationaccesses) | None                             | Authorization accesses matrix by group or email                                        |

## OIDCAuthorizationAccesses

| Key   | Type   | Default | Description |
| ----- | ------ | ------- | ----------- |
| group | String | ""      | Group name  |
| email | String | ""      | Email       |

## BasicAuthConfiguration

| Key         | Type                                                        | Default                              | Description      |
| ----------- | ----------------------------------------------------------- | ------------------------------------ | ---------------- |
| realm       | String                                                      | None                                 | Basic Auth Realm |
| credentials | [[BasicAuthUserConfiguration]](#basicauthuserconfiguration) | List of authorized user and password |

## BasicAuthUserConfiguration

| Key      | Type                                                | Default | Description   |
| -------- | --------------------------------------------------- | ------- | ------------- |
| user     | String                                              | None    | User name     |
| password | [CredentialConfiguration](#credentialconfiguration) | None    | User password |
