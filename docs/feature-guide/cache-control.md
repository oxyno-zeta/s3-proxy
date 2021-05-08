# Cache headers control

The server allow to control the cache headers at the server level or directly on a template file response or on streamed files. The following options are presented below.

## No Cache

<!-- prettier-ignore-start -->
!!! Warning
    This will override all headers applied on a template or a streamed file.
<!-- prettier-ignore-end -->

This configuration by default enabled for all responses including templates and streamed files. This is enabled by default for security reasons and to force no cache on everything.

Here is an example to enforce this default value:

```yaml linenums="1"
server:
  # Cache configuration
  cache:
    # Force no cache headers on all responses
    noCacheEnabled: true
```

## Global cache control

<!-- prettier-ignore-start -->
!!! Warning
    This will override all headers applied on a template or a streamed file.
<!-- prettier-ignore-end -->

This will allow to manage all cache headers values for all responses including templates and streamed files.

```yaml linenums="1"
server:
  # Cache configuration
  cache:
    # Force no cache headers on all responses
    noCacheEnabled: false
    # Expires header value
    expires:
    # Cache-control header value
    cacheControl:
    # Pragma header value
    pragma:
    # X-Accel-Expires header value
    xAccelExpires:
```

## Global ETag support

<!-- prettier-ignore-start -->
!!! Warning
    This will override all headers applied on a template or a streamed file.
<!-- prettier-ignore-end -->

This configuration will be applied to all responses, for all requests.

Here is an example of configuration to allow ETag support globally:

```yaml linenums="1"
server:
  # Cache configuration
  cache:
    # Force no cache headers on all responses
    noCacheEnabled: false
    # Cache-control header value
    cacheControl: must-revalidate, max-age=0
```

## ETag specific template file header

You can override the headers on a specific template to add `Cache-Control` and others on it.

The following example will show how to disable the "no-cache" feature and add a `Cache-Control` header that will allow the ETag support in browsers:

```yaml linenums="1"
server:
  # Cache configuration
  cache:
    # Force no cache headers on all responses
    noCacheEnabled: false

# Template configurations
templates:
  targetList:
    path: templates/target-list.tpl
    headers:
      Content-Type: '{{ template "main.headers.contentType" . }}'
      Cache-Control: must-revalidate, max-age=0
```

## ETag specific stream file header

You can override the headers on streamed files to add `Cache-Control` and others on them.

The following example will show how to disable the "no-cache" feature and add a `Cache-Control` header that will allow the ETag support in browsers:

```yaml linenums="1"
server:
  # Cache configuration
  cache:
    # Force no cache headers on all responses
    noCacheEnabled: false

# Targets map
targets:
  first-bucket:
    ## Actions
    actions:
      # Action for GET requests on target
      GET:
        # Will allow GET requests
        enabled: true
        # Configuration for GET requests
        config:
          # Allow to add headers to streamed files (can be templated)
          streamedFileHeaders:
            Cache-Control: must-revalidate, max-age=0
```
