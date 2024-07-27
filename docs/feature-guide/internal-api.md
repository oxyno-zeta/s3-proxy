# Internal API

This API is served by the internal server port and shouldn't be public. The aim of this one it to provide internal services like monitoring, health status or debug configuration.

## /health

This endpoint will give the status of the application. This will either answer with

- a 200 status code and a json body when everything is ok
- a 500 status code and a json body explaining why there is a problem

## /metrics

This endpoint will give metrics with the prometheus format.

## /config

This endpoint will show the latest configuration loaded by the application. Data are changed each time application is reloading the configuration.

Everything is shown in json format with the same keys are in the yaml configuration files.

Important note: Secret values aren't displayed. Only environment variable names and file paths.

Here is a payload example:

```json
{
  "config": {
    "log": { "level": "info", "format": "json", "filePath": "" },
    "tracing": {
      "fixedTags": null,
      "flushInterval": "",
      "udpHost": "",
      "queueSize": 0,
      "enabled": false,
      "logSpan": false
    },
    "metrics": { "disableRouterPath": false },
    "server": {
      "timeouts": {
        "readTimeout": "",
        "readHeaderTimeout": "60s",
        "writeTimeout": "",
        "idleTimeout": ""
      },
      "cors": null,
      "cache": null,
      "compress": {
        "enabled": true,
        "types": [
          "text/html",
          "text/css",
          "text/plain",
          "text/javascript",
          "application/javascript",
          "application/x-javascript",
          "application/json",
          "application/atom+xml",
          "application/rss+xml",
          "image/svg+xml"
        ],
        "level": 5
      },
      "ssl": null,
      "listenAddr": "",
      "port": 8080
    },
    "internalServer": {
      "timeouts": {
        "readTimeout": "",
        "readHeaderTimeout": "60s",
        "writeTimeout": "",
        "idleTimeout": ""
      },
      "cors": null,
      "cache": null,
      "compress": {
        "enabled": true,
        "types": [
          "text/html",
          "text/css",
          "text/plain",
          "text/javascript",
          "application/javascript",
          "application/x-javascript",
          "application/json",
          "application/atom+xml",
          "application/rss+xml",
          "image/svg+xml"
        ],
        "level": 5
      },
      "ssl": null,
      "listenAddr": "",
      "port": 9090
    },
    "targets": {
      "test": {
        "bucket": {
          "credentials": null,
          "requestConfig": null,
          "name": "bucket1",
          "prefix": "",
          "region": "us-east-1",
          "s3Endpoint": "",
          "s3ListMaxKeys": 1000,
          "s3MaxUploadParts": 10000,
          "s3UploadPartSize": 5,
          "s3UploadConcurrency": 5,
          "s3UploadLeavePartsOnError": false,
          "disableSSL": false,
          "credentials": {
            "accessKey": { "env": "FAKE", "path": "" },
            "secretKey": { "path": "/secret", "env": "" }
          }
        },
        "resources": null,
        "mount": { "host": "", "path": ["/test/"] },
        "actions": {
          "GET": { "config": null, "enabled": true },
          "PUT": null,
          "DELETE": null
        },
        "templates": {
          "folderList": null,
          "notFoundError": null,
          "internalServerError": null,
          "forbiddenError": null,
          "unauthorizedError": null,
          "badRequestError": null,
          "put": null,
          "delete": null,
          "helpers": null
        },
        "keyRewriteList": null
      }
    },
    "templates": {
      "folderList": {
        "path": "templates/folder-list.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "200"
      },
      "targetList": {
        "path": "templates/target-list.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "200"
      },
      "notFoundError": {
        "path": "templates/not-found-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "404"
      },
      "internalServerError": {
        "path": "templates/internal-server-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "500"
      },
      "unauthorizedError": {
        "path": "templates/unauthorized-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "401"
      },
      "forbiddenError": {
        "path": "templates/forbidden-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "403"
      },
      "badRequestError": {
        "path": "templates/bad-request-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "400"
      },
      "put": { "path": "templates/put.tpl", "headers": {}, "status": "204" },
      "delete": {
        "path": "templates/delete.tpl",
        "headers": {},
        "status": "204"
      },
      "helpers": ["templates/_helpers.tpl"]
    },
    "authProviders": null,
    "listTargets": { "mount": null, "resource": null, "enabled": false }
  }
}
```
