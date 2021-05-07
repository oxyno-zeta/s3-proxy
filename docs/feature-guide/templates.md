# Templates

The template feature will allow to customize the response body given by the application for many responses. It will allow to customize response headers to have also customized headers that goes with the customized response body.

All templates managed by S3-Proxy are Golang templates.

## Managed responses

S3-Proxy will manage HTML and JSON responses automatically by default. This switch is performed according to the `Accept` request header.

If no `Accept` header in found in the request, the HTML output will be used by default.

Default templates can be found [here](https://github.com/oxyno-zeta/s3-proxy/tree/master/templates).

## Functions

### General functions

In all these templates, all [Masterminds/sprig](https://github.com/Masterminds/sprig) functions are available.

### S3-Proxy specific functions

In all these templates, S3-Proxy specific functions are available:

- `humanSize` with `int` input in order to transform bytes to human size
- `requestURI` with `http.Request` input in order to get the full request URI from incoming request
- `requestScheme` with `http.Request` input in order to get the scheme from incoming request
- `requestHost` with `http.Request` input in order to get the hostname from incoming request

## Helpers

Different helpers are available by default:

- `main.userIdentifier` will return the user identifier from the incoming request only if user exists
- `main.headers.contentType` will return the content type header from the incoming request
- `main.body.errorJsonBody` will return the json content body for an error

## Template data structure and usage

### Target List

This template is used in order to list all targets buckets declared in the configuration file.

Available data:

| Name    | Type                                                                   | Description                                       |
| ------- | ---------------------------------------------------------------------- | ------------------------------------------------- |
| User    | [GenericUser](#genericuser)                                            | Authenticated user if present in incoming request |
| Request | [http.Request](https://golang.org/pkg/net/http/#Request)               | HTTP Request object from golang                   |
| Targets | Map[String][target](../configuration/structure.md#targetconfiguration) | The target map as coming from the configuration   |

### Folder List

This template is used in order to list files and folders in a bucket folder.

Available data:

| Name       | Type                                                     | Description                                       |
| ---------- | -------------------------------------------------------- | ------------------------------------------------- |
| User       | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request    | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |
| Entries    | [[Entry](#entry)]                                        | Folder entries                                    |
| BucketName | String                                                   | Bucket name                                       |
| Name       | String                                                   | Target name                                       |

### Not found error

This template is used for all `Not found` errors.

Available data:

| Name    | Type                                                     | Description                                       |
| ------- | -------------------------------------------------------- | ------------------------------------------------- |
| User    | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |
| Error   | Error                                                    | Error raised and caught                           |

### Unauthorized error

This template is used for all `Unauthorized` errors.

Available data:

| Name    | Type                                                     | Description                                       |
| ------- | -------------------------------------------------------- | ------------------------------------------------- |
| User    | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |
| Error   | Error                                                    | Error raised and caught                           |

### Forbidden error

This template is used for all `Forbidden` errors.

Available data:

| Name    | Type                                                     | Description                                       |
| ------- | -------------------------------------------------------- | ------------------------------------------------- |
| User    | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |
| Error   | Error                                                    | Error raised and caught                           |

### Internal Server Error

This template is used for all `Internal server error` errors.

Available data:

| Name    | Type                                                     | Description                                       |
| ------- | -------------------------------------------------------- | ------------------------------------------------- |
| User    | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |
| Error   | Error                                                    | Error raised and caught                           |

### Bad Request error

This template is used for all `Bad Request` errors.

Available data:

| Name    | Type                                                     | Description                                       |
| ------- | -------------------------------------------------------- | ------------------------------------------------- |
| User    | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |
| Error   | Error                                                    | Error raised and caught                           |

## Headers templates and structures

### Generic case

This case is the main one and used for all templates rendered explained before.

The following table will show the data structure available for the header template rendering:

| Name    | Type                                                     | Description                                       |
| ------- | -------------------------------------------------------- | ------------------------------------------------- |
| User    | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |

### Stream file case

This case is a special case, used only when a file is streamed from S3. This will allow to add headers to streamed files with GET requests.

The following table will show the data structure available for the header template rendering:

| Name       | Type                                                     | Description                                       |
| ---------- | -------------------------------------------------------- | ------------------------------------------------- |
| User       | [GenericUser](#genericuser)                              | Authenticated user if present in incoming request |
| Request    | [http.Request](https://golang.org/pkg/net/http/#Request) | HTTP Request object from golang                   |
| StreamFile | [StreamFile](#streamfile)                                | Stream file object                                |

## Common/Other structures

### GenericUser

Generic user is a golang interface that will match all kind of users managed by application.

These are the properties available:

| Name            | Type     | Description                                                                      |
| --------------- | -------- | -------------------------------------------------------------------------------- |
| GetType         | String   | Get type of user (OIDC or BASIC)                                                 |
| GetIdentifier   | String   | Get identifier (Username for basic auth user or Username or email for OIDC user) |
| GetUsername     | String   | Get username                                                                     |
| GetName         | String   | Get name (only available for OIDC user)                                          |
| GetGroups       | [String] | Get groups (only available for OIDC user)                                        |
| GetGivenName    | String   | Get given name (only available for OIDC user)                                    |
| GetFamilyName   | String   | Get family name (only available for OIDC user)                                   |
| GetEmail        | String   | Get email (only available for OIDC user)                                         |
| IsEmailVerified | Boolean  | Is Email Verified ? (only available for OIDC user)                               |

### Entry

| Name         | Type    | Description                   |
| ------------ | ------- | ----------------------------- |
| Type         | String  | Entry type (FOLDER or FILE)   |
| Name         | String  | Entry name                    |
| ETag         | String  | ETag from bucket (file only)  |
| LastModified | Time    | Last modified entry           |
| Size         | Integer | Entry file (file only)        |
| Key          | String  | Full key from S3 response     |
| Path         | String  | Access path to entry from web |

### StreamFile

| Name               | Type                                      | Description                       |
| ------------------ | ----------------------------------------- | --------------------------------- |
| CacheControl       | String                                    | Cache control value from S3       |
| Expires            | String                                    | Expires value from S3             |
| ContentDisposition | String                                    | Content disposition value from S3 |
| ContentEncoding    | String                                    | Content encoding value from S3    |
| ContentLanguage    | String                                    | Content language value from S3    |
| ContentLength      | Integer                                   | Content length value from S3      |
| ContentRange       | String                                    | Content range value from S3       |
| ContentType        | String                                    | Content type value from S3        |
| ETag               | String                                    | ETag value from S3                |
| LastModified       | [Time](https://golang.org/pkg/time/#Time) | Last modified value from S3       |
| Metadata           | Map[String]String                         | Metadata value from S3            |
