# Key Rewrite

## What is a "key" ?

In S3, "path" are called "key". This means for S3-Proxy that when a client is calling with this type of request: `GET /test/file.html`, the S3 key will be `/test/file.html` (this example consider that S3 prefix and mount path are empty).

## General description

The key rewrite feature is here to add the idea of url rewrite for S3 calls.

The list is provided per target and don't skip the S3 prefix configured in the bucket section.

Moreover, this list is tested over all actions (GET, PUT or DELETE) without any filter. This is done on purpose in order to be agnostic of client requests.

This can be considered as similar as symlinks in Unix.

## How this is working ?

The list of key rewrite is called over all incoming keys that will be sent to S3 (S3 prefix excluded).

The first source regexp matching the key will select the target template that will be used. If nothing is matching, the incoming key will be untouched.

There are some things that should be known about this feature:

- As the project is done with Golang, the regexp must be compatible with this language. The group namings are supported in source and can be used in target template.
- S3 Prefix declared in the bucket configuration won't be present in the source key. This is done on purpose because this is configuration that should be added to all keys before calling S3.
- The index document feature is called **after** the key rewrite feature. So the source key won't contains any `indexDocument` inside.
- The `redirectWithTrailingSlashForNotFoundFile` feature is also called **after** the key rewrite feature.
- For PUT and DELETE actions, the entire key is provided before calling the key rewrite feature. This means that the file name is included in the source key. This is done on purpose because DELETE and PUT actions are done on files, so it should be included in source key before calling the key rewrite feature.

## For which situations ?

This feature is done because it can happened that it is needed to rewrite keys before calling S3.

This can happened when:

- An old application has been delivered and not updated that try to get a file on a specific URL and you moved this file to another location. Now, you can create a key rewrite in order to "redirect" the call to the new place.
- A website has been moved in the S3 storage and you want the old URL to continue to work for a given amount of time.

And other examples...

## Examples

Examples below will have a small configuration containing just the needed keys related to the example.

### Ignored key

This example will show the ignore key behavior explained before.

In this example, we will consider this request `GET /file.html` and the following configuration:

```yaml
# ...
targets:
  - name: test
    # ...
    keyRewriteList:
      - source: ^/my-super-complicated-path$
        target: /redirected
    bucket:
      # ...
      prefix: ""
```

The S3 key result of this request will be : `/file.html`.

### Simple key rewrite

This example will show a simple key rewrite without any group name or capture.

In this example, we will consider this request `GET /file.html` and the following configuration:

```yaml
# ...
targets:
  - name: test
    # ...
    keyRewriteList:
      - source: ^/file.html$
        target: /redirected/file.html
    bucket:
      # ...
      prefix: ""
```

The S3 key result of this request will be : `/redirected/file.html`.

### Golang naming groups

This example will show a capture with name reused in template.

In this example, we will consider this request `GET /folder1/file.html` and the following configuration:

```yaml
# ...
targets:
  - name: test
    # ...
    keyRewriteList:
      - source: ^/(?P<one>\w+)/file.html$
        target: /$one/fake/$one/file.html
    bucket:
      # ...
      prefix: ""
```

The S3 key result of this request will be : `/folder1/fake/folder1/file.html`.

### S3 prefix

This example will show a simple key rewrite with the S3 prefix data.

In this example, we will consider this request `GET /file.html` and the following configuration:

```yaml
# ...
targets:
  - name: test
    # ...
    keyRewriteList:
      - source: ^/file.html$
        target: /redirected/file.html
    bucket:
      # ...
      prefix: "/folder1"
```

The S3 key result of this request will be : `/folder1/redirected/file.html`.
