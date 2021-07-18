# Webhooks

This feature have been created to have hooks on all type of requests managed by s3-proxy.

These comes with some limitations:

<!-- prettier-ignore -->
- Only HTTP is supported.
    - Why ? Because we aren't too much managing the maintenance of the app and managing different type of hooks will take a lot of efforts and time.
    - If you need another type of message, we recommend you to develop a sidecar to this application that will transform the request.
- mTLS isn't supported.
    - Same reason as before.
- All hooks are run in a Go routine and in a sequential way in this one.
- Webhook body isn't open to customization.
<!-- prettier-ignore-end -->

## Body

There is a common way in the application of creating a webhook body.

Every hook will have a body built on a common part and only "input" section is different.

The main body is called the [HookBody](#hookbody).

Here are all cased for input metadata:

- GET: [GetInputMetadataHookBody](#getinputmetadatahookbody)
- PUT: [PutInputMetadataHookBody](#putinputmetadatahookbody)
- DELETE: [DeleteInputMetadataHookBody](#deleteinputmetadatahookbody)

### HookBody

| Field          | Type                                              | Description                                                   |
| -------------- | ------------------------------------------------- | ------------------------------------------------------------- |
| action         | String                                            | Webhook action (`GET`, `PUT` or `DELETE`)                     |
| requestPath    | String                                            | Request path                                                  |
| inputMetadata  | Input metadata cases                              | Input metadata                                                |
| outputMetadata | [OutputMetadataHookBody](#outputmetadatahookbody) | Output / result metadata (metadata generated from S3 results) |
| target         | [TargetHookBody](#targethookbody)                 | Target data                                                   |

### OutputMetadataHookBody

| Field      | Type   | Description                        |
| ---------- | ------ | ---------------------------------- |
| bucket     | String | Bucket name requested              |
| region     | String | Region where the bucket is located |
| s3Endpoint | String | S3 endpoint used                   |
| key        | String | Key requested from S3 bucket       |

### TargetHookBody

| Field | Type   | Description                      |
| ----- | ------ | -------------------------------- |
| name  | String | Target name matching the request |

### GetInputMetadataHookBody

| Field             | Type   | Description                        |
| ----------------- | ------ | ---------------------------------- |
| ifModifiedSince   | String | `If-Modified-Since` header value   |
| ifMatch           | String | `If-Match` header value            |
| ifNoneMatch       | String | `If-None-Match` header value       |
| ifUnmodifiedSince | String | `If-Unmodified-Since` header value |
| range             | String | `Range` header value               |

### PutInputMetadataHookBody

| Field       | Type    | Description                                 |
| ----------- | ------- | ------------------------------------------- |
| filename    | String  | Uploaded file name coming from request form |
| contentType | String  | Uploaded file content type                  |
| contentSize | Integer | Uploaded file size                          |

### DeleteInputMetadataHookBody

In this specific case, there is nothing. The input data is null.
