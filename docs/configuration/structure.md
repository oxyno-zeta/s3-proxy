# Structure

The configuration must be set in multiple YAML files located in `conf/` folder from the current working directory.

You can create multiple files containing different part of the configuration. A global merge will be done across all data in all files.

Moreover, the configuration files will be watched for modifications.

You can see a full example in the [Example section](./example.md)

## Main structure

| Key            | Type                                                      | Required | Default | Description                                                                                                         |
| -------------- | --------------------------------------------------------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------------- |
| log            | [LogConfiguration](#logconfiguration)                     | No       | None    | Log configurations                                                                                                  |
| server         | [ServerConfiguration](#serverconfiguration)               | No       | None    | Server configurations                                                                                               |
| internalServer | [ServerConfiguration](#serverconfiguration)               | No       | None    | Internal Server configurations                                                                                      |
| template       | [TemplateConfiguration](#templateconfiguration)           | No       | None    | Template configurations                                                                                             |
| targets        | Map[String][targetconfiguration](#targetconfiguration)    | Yes      | None    | Targets configuration. Map key will be considered as the target name. (This will used in urls and list of targets.) |
| authProviders  | [AuthProvidersConfiguration](#authProvidersconfiguration) | No       | None    | Authentication providers configuration                                                                              |
| listTargets    | [ListTargetsConfiguration](#listtargetsconfiguration)     | No       | None    | List targets feature configuration                                                                                  |

## LogConfiguration

| Key      | Type   | Required | Default | Description                                         |
| -------- | ------ | -------- | ------- | --------------------------------------------------- |
| level    | String | No       | `info`  | Log level                                           |
| format   | String | No       | `json`  | Log format (available values are: `json` or `text`) |
| filePath | String | No       | `""`    | Log file path                                       |

## ServerConfiguration

| Key        | Type                                    | Required | Default | Description         |
| ---------- | --------------------------------------- | -------- | ------- | ------------------- |
| listenAddr | String                                  | No       | `""`    | Listen Address      |
| port       | Integer                                 | No       | `8080`  | Listening Port      |
| cors       | [ServerCorsConfig](#servercorsconfig)   | No       | None    | CORS configuration  |
| cache      | [ServerCacheConfig](#servercacheconfig) | No       | None    | Cache configuration |

## ServerCompressConfig

| Key     | Type     | Required | Default                                                                                                                                                                                       | Description                                |
| ------- | -------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| enabled | Boolean  | No       | `true`                                                                                                                                                                                        |  Is the compression enabled ?              |
| level   | Integer  | No       | `5`                                                                                                                                                                                           | The level of GZip compression              |
| types   | [String] | No       | `["text/html","text/css","text/plain","text/javascript","application/javascript","application/x-javascript","application/json","application/atom+xml","application/rss+xml","image/svg+xml"]` | The content type list compressed in output |

## ServerCacheConfig

| Key            | Type    | Required | Default | Description                             |
| -------------- | ------- | -------- | ------- | --------------------------------------- |
| noCacheEnabled | Boolean | false    | `false` | Force no cache headers on all responses |
| expires        | String  | false    | `""`    | `Expires` header value                  |
| cacheControl   | String  | false    | `""`    | `Cache-Control` header value            |
| pragma         | String  | false    | `""`    | `Pragma` header value                   |
| xAccelExpires  | String  | false    | `""`    | `X-Accel-Expires` header value          |

Here is an example of configuration to allow ETag support:

```yaml
server:
  cache:
    cacheControl: must-revalidate, max-age=0
```

## ServerCorsConfig

This feature is powered by [go-chi/cors](https://github.com/go-chi/cors). You can read more documentation about all field there.

| Key                | Type     | Required | Default                                                                        | Description                                                       |
| ------------------ | -------- | -------- | ------------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| enabled            | Boolean  | No       | `false`                                                                        | Is CORS support enabled ?                                         |
| allowAll           | Boolean  | No       | `false`                                                                        | Allow all CORS requests with all origins, all HTTP methods, etc ? |
| allowOrigins       | [String] | No       | Allow origins array. Example: https://fake.com. This support stars in origins. |
| allowMethods       | [String] | No       | Allow HTTP Methods                                                             |
| allowHeaders       | [String] | No       | Allow headers                                                                  |
| exposeHeaders      | [String] | No       | Expose headers                                                                 |
| maxAge             | Integer  | No       | Max age. 300 is the maximum value not ignored by any of major browsers.        |
| allowCredentials   | Boolean  | No       | Allow credentials                                                              |
| debug              | Boolean  | No       | Debug mode for [go-chi/cors](https://github.com/go-chi/cors)                   |
| optionsPassthrough | Boolean  | No       | OPTIONS method Passthrough                                                     |

## TemplateConfiguration

| Key                 | Type                                                    | Required | Default                                                                                                                                              | Description                                                                                               |
| ------------------- | ------------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| helpers             | [String]                                                | No       | `[templates/_helpers.tpl]`                                                                                                                           | Template Golang helpers                                                                                   |
| targetList          | [TemplateConfigurationItem](#templateconfigurationitem) | No       | `targetList: { path: "templates/target-list.tpl", headers: { "Content-Type": "{{ template \"main.headers.contentType\" . }}" } }`                    | Target list template path. More information [here](../feature-guide/templates.md#generic-case).           |
| folderList          | [TemplateConfigurationItem](#templateconfigurationitem) | No       | `folderList: { path: "templates/folder-list.tpl", headers: { "Content-Type": "{{ template \"main.headers.contentType\" . }}" } }`                    | Folder list template path. More information [here](../feature-guide/templates.md#generic-case).           |
| notFoundError       | [TemplateConfigurationItem](#templateconfigurationitem) | No       | `notFoundError: { path: "templates/not-found-error.tpl", headers: { "Content-Type": "{{ template \"main.headers.contentType\" . }}" } }`             | Not found template path. More information [here](../feature-guide/templates.md#generic-case).             |
| unauthorizedError   | [TemplateConfigurationItem](#templateconfigurationitem) | No       | `unauthorizedError: { path: "templates/unauthorized-error.tpl", headers: { "Content-Type": "{{ template \"main.headers.contentType\" . }}" } }`      | Unauthorized template path. More information [here](../feature-guide/templates.md#generic-case).          |
| forbiddenError      | [TemplateConfigurationItem](#templateconfigurationitem) | No       | `forbiddenError: { path: "templates/forbidden-error.tpl", headers: { "Content-Type": "{{ template \"main.headers.contentType\" . }}" } }`            | Forbidden template path. More information [here](../feature-guide/templates.md#generic-case).             |
| badRequestError     | [TemplateConfigurationItem](#templateconfigurationitem) | No       | `badRequestError: { path: "templates/bad-request-error.tpl", headers: { "Content-Type": "{{ template \"main.headers.contentType\" . }}" } }`         | Bad Request template path. More information [here](../feature-guide/templates.md#generic-case).           |
| internalServerError | [TemplateConfigurationItem](#templateconfigurationitem) | No       | `internalServerError: { path: "templates/internal-server-error.tpl", headers: { "Content-Type": "{{ template \"main.headers.contentType\" . }}" } }` | Internal server error template path. More information [here](../feature-guide/templates.md#generic-case). |

## TemplateConfigurationItem

| Key     | Type              | Required | Default | Description                                                                                                                                                                                                               |
| ------- | ----------------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| path    | String            | True     | `""`    | File path to template file                                                                                                                                                                                                |
| headers | Map[String]String | False    | None    | Headers containing templates. Key corresponds to header and value to the template. If templated value is empty, the header won't be added to answer. More information [here](../feature-guide/templates.md#generic-case). |

## TargetConfiguration

| Key            | Type                                          | Required | Default            | Description                                                                                                                                                                                                                              |
| -------------- | --------------------------------------------- | -------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| bucket         | [BucketConfiguration](#bucketconfiguration)   | Yes      | None               | Bucket configuration                                                                                                                                                                                                                     |
| resources      | [[Resource]](#resource)                       | No       | None               | Resources declaration for path whitelist or specific authentication on path list. WARNING: Think about all path that you want to protect. At the end of the list, you should add a resource filter for /\* otherwise, it will be public. |
| mount          | [MountConfiguration](#mountconfiguration)     | Yes      | None               | Mount point configuration                                                                                                                                                                                                                |
| actions        | [ActionsConfiguration](#actionsconfiguration) | No       | GET action enabled | Actions allowed on target (GET, PUT or DELETE)                                                                                                                                                                                           |
| keyRewriteList | [[KeyRewrite]](#keyrewrite)                   | No       | None               | Key rewrite list is here to allow rewriting keys before sending request to S3 (See more information [here](../feature-guide/key-rewrite.md))                                                                                             |
| templates      | [TargetTemplateConfig](#targettemplateconfig) | No       | None               | Custom target templates from files on local filesystem or in bucket                                                                                                                                                                      |

## KeyRewrite

See more information [here](../feature-guide/key-rewrite.md).

| Key    | Type   | Required | Default | Description                                             |
| ------ | ------ | -------- | ------- | ------------------------------------------------------- |
| source | String | Required | None    | Source regexp matcher with golang group naming support. |
| target | String | Required | None    | Target template for new key send to S3.                 |

## TargetTemplateConfig

| Key                 | Type                                                  | Required | Default | Description                                                                                                |
| ------------------- | ----------------------------------------------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------------- |
| helpers             | [[TargetHelperConfigItem](#targethelperconfigitem)]   | No       | None    | Helpers list custom template declarations.                                                                 |
| folderList          | [TargetTemplateConfigItem](#targettemplateconfigitem) | No       | None    | Folder list custom template declaration. More information [here](../feature-guide/templates.md).           |
| notFound            | [TargetTemplateConfigItem](#targettemplateconfigitem) | No       | None    | Not Found custom template declaration. More information [here](../feature-guide/templates.md).             |
| internalServerError | [TargetTemplateConfigItem](#targettemplateconfigitem) | No       | None    | Internal server error custom template declaration. More information [here](../feature-guide/templates.md). |
| forbidden           | [TargetTemplateConfigItem](#targettemplateconfigitem) | No       | None    | Forbidden custom template declaration. More information [here](../feature-guide/templates.md).             |
| unauthorized        | [TargetTemplateConfigItem](#targettemplateconfigitem) | No       | None    | Unauthorized custom template declaration. More information [here](../feature-guide/templates.md).          |
| badRequest          | [TargetTemplateConfigItem](#targettemplateconfigitem) | No       | None    | Bad Request custom template declaration. More information [here](../feature-guide/templates.md).           |

## TargetHelperConfigItem

| Key      | Type    | Required | Default | Description                                     |
| -------- | ------- | -------- | ------- | ----------------------------------------------- |
| inBucket | Boolean | No       | `false` | Is the file in bucket or on local file system ? |
| path     | String  | Yes      | None    | Path for template file                          |

## TargetTemplateConfigItem

| Key      | Type              | Required | Default                                                                                     | Description                                                                                                                                                                                                               |
| -------- | ----------------- | -------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| inBucket | Boolean           | No       | `false`                                                                                     | Is the file in bucket or on local file system ?                                                                                                                                                                           |
| path     | String            | Yes      | None                                                                                        | Path for template file                                                                                                                                                                                                    |
| headers  | Map[String]String | False    | This will be set to corresponding [TemplateConfiguration](#templateconfiguration) if empty. | Headers containing templates. Key corresponds to header and value to the template. If templated value is empty, the header won't be added to answer. More information [here](../feature-guide/templates.md#generic-case). |

## ActionsConfiguration

| Key    | Type                                                    | Required | Default | Description                                        |
| ------ | ------------------------------------------------------- | -------- | ------- | -------------------------------------------------- |
| GET    | [GetActionConfiguration](#getactionconfiguration)       | No       | None    | Action configuration for GET requests on target    |
| PUT    | [PutActionConfiguration](#putactionconfiguration)       | No       | None    | Action configuration for PUT requests on target    |
| DELETE | [DeleteActionConfiguration](#deleteactionconfiguration) | No       | None    | Action configuration for DELETE requests on target |

## GetActionConfiguration

| Key     | Type                                                          | Required | Default | Description                    |
| ------- | ------------------------------------------------------------- | -------- | ------- | ------------------------------ |
| enabled | Boolean                                                       | No       | `false` | Will allow GET requests        |
| config  | [GetActionConfigConfiguration](#getactionconfigconfiguration) | No       | None    | Configuration for GET requests |

## GetActionConfigConfiguration

| Key                                      | Type              | Required | Default | Description                                                                                                                                                                                                                                                                        |
| ---------------------------------------- | ----------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| redirectWithTrailingSlashForNotFoundFile | Boolean           | No       | `false` | This option allow to do a redirect with a trailing slash when a GET request on a file (not a folder) encountered a 404 not found.                                                                                                                                                  |
| indexDocument                            | String            | No       | `""`    | The index document name. If this document is found, get it instead of list folder. Example: `index.html`                                                                                                                                                                           |
| streamedFileHeaders                      | Map[String]String | No       | `nil`   |  Headers containing templates that will be added to streamed files in this target. Key corresponds to header and value to the template. If templated value is empty, the header won't be added to answer. More information [here](../feature-guide/templates.md#stream-file-case). |

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

| Key           | Type                                                            | Required | Default     | Description                                                                                                                                                                                                                                                                              |
| ------------- | --------------------------------------------------------------- | -------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name          | String                                                          | Yes      | None        | Bucket name in S3 provider                                                                                                                                                                                                                                                               |
| prefix        | String                                                          | No       | None        | Bucket prefix                                                                                                                                                                                                                                                                            |
| region        | String                                                          | No       | `us-east-1` | Bucket region                                                                                                                                                                                                                                                                            |
| s3Endpoint    | String                                                          | No       | None        | Custom S3 Endpoint for non AWS S3 bucket                                                                                                                                                                                                                                                 |
| credentials   | [BucketCredentialConfiguration](#bucketcredentialconfiguration) | No       | None        | Credentials to access S3 bucket                                                                                                                                                                                                                                                          |
| disableSSL    | Boolean                                                         | No       | `false`     | Disable SSL connection                                                                                                                                                                                                                                                                   |
| s3ListMaxKeys | Integer                                                         | No       | `1000`      | This flag will be used for the max pagination list management of files and "folders" in S3. In S3 list requests, the limit is fixed to 1000 items maximum. S3-Proxy will allow to increase this by making multiple requests to S3. Warning: This will increase the memory and CPU usage. |

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

## ResourceOIDC

| Key                    | Type                                                      | Required | Default | Description                                                                                                                                                                               |
| ---------------------- | --------------------------------------------------------- | -------- | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| authorizationAccesses  | [[OIDCAuthorizationAccesses]](#oidcauthorizationaccesses) | No       | None    | Authorization accesses matrix by group or email. If not set, authenticated users will be authorized (no group or email validation will be performed if authorizationOPAServer isn't set). |
| authorizationOPAServer | [OPAServerAuthorization](#opaserverauthorization)         | No       | None    | Authorization through an OPA (Open Policy Agent) server                                                                                                                                   |

## OPAServerAuthorization

| Key  | Type              | Required | Default | Description                                                                                                          |
| ---- | ----------------- | -------- | ------- | -------------------------------------------------------------------------------------------------------------------- |
| url  | String            | Yes      | None    | URL of the OPA server including the data path (see the dedicated section for [OPA](../feature-guide/opa.md))         |
| tags | Map[String]String | No       | `{}`    | Data that will be added as tags in the OPA input data (see the dedicated section for [OPA](../feature-guide/opa.md)) |

## OIDCAuthorizationAccesses

| Key    | Type    | Required               | Default | Description                                    |
| ------ | ------- | ---------------------- | ------- | ---------------------------------------------- |
| group  | String  | Required without email | None    | Group name                                     |
| email  | String  | Required without group | None    | Email                                          |
| regexp | Boolean | No                     | `false` | Consider group or email as regexp for matching |

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

| Key  | Type     | Required | Default | Description                                                                                                                            |
| ---- | -------- | -------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| host | String   | No       | `""`    | Host domain requested (eg: localhost:888 or google.fr). Put empty for all domains. Note: Glob patterns for host domains are supported. |
| path | [String] | Yes      | None    | A path list for mounting point                                                                                                         |

## ListTargetsConfiguration

| Key      | Type                                      | Required | Default | Description                                                                 |
| -------- | ----------------------------------------- | -------- | ------- | --------------------------------------------------------------------------- |
| enabled  | Boolean                                   | Yes      | None    | To enable the list targets feature                                          |
| mount    | [MountConfiguration](#mountconfiguration) | Yes      | None    | Mount point configuration                                                   |
| resource | [Resource](#resource)                     | No       | None    | Resources declaration for path whitelist or specific authentication on path |
