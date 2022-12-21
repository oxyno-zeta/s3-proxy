# Signed URL

This feature have been done to avoid steaming files through S3-Proxy application and to redirect directly to S3 to perform this operation. This will help when you have a small throughput or stability issues.

## Limitations

- This is supported only for GET requests.

## Configuration

Simply enable it in the configuration:

```yaml
#...
targets:
  target1:
    #...
    actions:
      GET:
        enabled: true
        config:
          # Redirect to a S3 signed URL
          redirectToSignedUrl: true
          # Signed URL expiration time
          signedUrlExpiration: 15m
          # ...
```
