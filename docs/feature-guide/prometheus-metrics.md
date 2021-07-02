# Prometheus Metrics

This section will describe the prometheus metrics that the application is serving.

Here is also an example of Prometheus dashboard that you can import as JSON file: [dashboard](./s3-proxy-dashboard.json).

This dashboard has been done and tested on Grafana 7.0.

## http_requests_total

Type: Counter

Prometheus data:

- `http_requests_total`

Description: How many HTTP requests have been processed ?

Fields:

| Field name    | Description                                                                                                                                 |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `server`      | Which Server is handling the request ? Can be "business" or "internal". Internal is the server handling requests for metrics, health check. |
| `status_code` | Request status code                                                                                                                         |
| `method`      | Request method                                                                                                                              |
| `host`        | Hostname used for the request                                                                                                               |
| `path`        | Path used for the request                                                                                                                   |

## http_request_duration_seconds

Type: Summary

Prometheus data:

- `http_request_duration_seconds_sum`
- `http_request_duration_seconds_count`

Description: The HTTP request latencies in seconds.

Fields:

| Field name    | Description                                                                                                                                 |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `server`      | Which Server is handling the request ? Can be "business" or "internal". Internal is the server handling requests for metrics, health check. |
| `status_code` | Request status code                                                                                                                         |
| `method`      | Request method                                                                                                                              |
| `host`        | Hostname used for the request                                                                                                               |
| `path`        | Path used for the request                                                                                                                   |

## http_request_size_bytes

Type: Summary

Prometheus data:

- `http_request_size_bytes_sum`
- `http_request_size_bytes_count`

Description: The HTTP request sizes in bytes.

Fields:

| Field name    | Description                                                                                                                                 |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `server`      | Which Server is handling the request ? Can be "business" or "internal". Internal is the server handling requests for metrics, health check. |
| `status_code` | Request status code                                                                                                                         |
| `method`      | Request method                                                                                                                              |
| `host`        | Hostname used for the request                                                                                                               |
| `path`        | Path used for the request                                                                                                                   |

## http_response_size_bytes

Type: Summary

Prometheus data:

- `http_response_size_bytes_sum`
- `http_response_size_bytes_count`

Description: The HTTP response sizes in bytes.

Fields:

| Field name    | Description                                                                                                                                 |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `server`      | Which Server is handling the request ? Can be "business" or "internal". Internal is the server handling requests for metrics, health check. |
| `status_code` | Request status code                                                                                                                         |
| `method`      | Request method                                                                                                                              |
| `host`        | Hostname used for the request                                                                                                               |
| `path`        | Path used for the request                                                                                                                   |

## up

Type: Gauge

Prometheus data:

- `up`

Description: 1 = up (hardcoded)

| Field name | Description                                |
| ---------- | ------------------------------------------ |
| component  | Component name (hardcoded with `s3-proxy`) |

## s3_operations_total

Type: Counter

Prometheus data:

- `s3_operations_total`

Description: How many operations are generated to s3 in total ?

Fields:

| Field name    | Description  |
| ------------- | ------------ |
| `target_name` | Target name  |
| `bucket_name` | Bucket name  |
| `operation`   | S3 operation |

## authenticated_total

Type: Counter

Prometheus data:

- `authenticated_total`

Description: How many users have been authenticated ?

Fields:

| Field name      | Description                                        |
| --------------- | -------------------------------------------------- |
| `provider_type` | Provider type (`oidc` or `basic-auth` for example) |
| `provider_name` | Provider name                                      |

## authorized_total

Type: Counter

Prometheus data:

- `authorized_total`

Description: How many users have been authorized ?

Fields:

| Field name      | Description                                            |
| --------------- | ------------------------------------------------------ |
| `provider_type` | Provider type (`oidc-opa` or `basic-auth` for example) |

## succeed_webhooks_total

Type: Counter

Prometheus data:

- `succeed_webhooks_total`

Description: How many webhooks have been succeed ?

Fields:

| Field name    | Description                                         |
| ------------- | --------------------------------------------------- |
| `target_name` | Target name containing the webhook definition       |
| `action_name` | Webhook action triggered (`GET`, `PUT` or `DELETE`) |

## failed_webhooks_total

Type: Counter

Prometheus data:

- `failed_webhooks_total`

Description: How many webhooks have been failed ?

Fields:

| Field name    | Description                                         |
| ------------- | --------------------------------------------------- |
| `target_name` | Target name containing the webhook definition       |
| `action_name` | Webhook action triggered (`GET`, `PUT` or `DELETE`) |
