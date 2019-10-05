# Templates

All following templates are Golang templates. These are here to template HTML pages for managed errors or listings.

In all these templates, all [Masterminds/sprig](https://github.com/Masterminds/sprig) functions are available and another one for `Folder list` case called `humanSize` in order to transform bytes to human size.

## Target List

This template is used in order to list all targets buckets declared in the configuration file.

Variables:

| Name    | Type                                 | Description                             |
| ------- | ------------------------------------ | --------------------------------------- |
| Targets | [[Target]](configuration.md#Targets) | The target object list in configuration |

## Folder List

This template is used in order to list files and folders in a bucket folder.

Variables:

| Name       | Type    | Description    |
| ---------- | ------- | -------------- |
| Entries    | [Entry] | Folder entries |
| BucketName | String  | Bucket name    |
| Name       | String  | Target name    |
| Path       | String  | Request path   |

Entry:

| Name         | Type    | Description                   |
| ------------ | ------- | ----------------------------- |
| Type         | String  | Entry type (FOLDER or FILE)   |
| Name         | String  | Entry name                    |
| ETag         | String  | ETag from bucket (file only)  |
| LastModified | Time    | Last modified entry           |
| Size         | Integer | Entry file (file only)        |
| Key          | String  | Full key from S3 response     |
| Path         | String  | Access path to entry from web |

## Not found

This template is used for all `Not found` errors.

Variables:

| Name | Type   | Description  |
| ---- | ------ | ------------ |
| Path | String | Request Path |

## Internal Server Error

This template is used for all `Internal server error` errors.

Variables:

| Name  | Type   | Description            |
| ----- | ------ | ---------------------- |
| Path  | String | Request Path           |
| Error | Error  | Error raised and catch |
