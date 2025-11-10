# API

## GET

This kind of requests will allow to get files or to perform a directory listing.

There is 2 different management cases:

- If path ends with a slash, the backend will consider this as a directory and will perform a directory listing or will display index document.
  Example: `GET /dir1/`

- If path doesn't end with a slash, the backend will consider this as a file request. Example: `GET /file.pdf`

## HEAD

Those kind of requests is similar to `GET` ones but won't provide any result body.

There are working the same way for management cases for directories (eg: `HEAD /dir1/`) or files (eg: `HEAD /file.pdf`).

GET and HEAD requests include standard S3 headers (`Cache-Control`, `Content-Type`, `ETag`, `Last-Modified`, etc.) and add a [`Content-Digest`](https://developer.mozilla.org/docs/Web/HTTP/Reference/Headers/Content-Digest) header when the S3 object has a `FULL_OBJECT` checksum of algorithm SHA-256, SHA-1, CRC32C, or CRC32 (but not CRC64NVME or MD5).

## PUT

This kind of requests will allow to send file in directory (so to upload a file in S3).

The PUT request path must be a directory and must be a multipart form with a key named `file` with a file inside.
Example: `PUT --form file:@file.pdf /dir1/`

## DELETE

This kind of requests will allow to delete files (**only**). Folder removal is forbidden at this moment.

The DELETE request path must contain the file name. Example: `DELETE /dir1/dir2/file.pdf`.
