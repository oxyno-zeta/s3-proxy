# Migrate from v3 to v4

Several breaking changes have been done during this release on the configuration:

- Targets
- Templates structure and naming
- TargetTemplates structure and naming
- Default template names
- Index document

Those changes have been made in order to have a better merge feature on the configuration, for new features and to keep things in sync and organized.

Here is a diff of the configuration with highlighted keys you will have to change during the upgrade:

```diff linenums="1"
 # Template configurations
 # templates:
-#   badRequest: templates/bad-request.tpl
-#   folderList: templates/folder-list.tpl
-#   forbidden: templates/forbidden.tpl
-#   internalServerError: templates/internal-server-error.tpl
-#   notFound: templates/not-found.tpl
-#   targetList: templates/target-list.tpl
-#   unauthorized: templates/unauthorized.tpl
+#   helpers:
+#     - templates/_helpers.tpl
+#   targetList:
+#     path: templates/target-list.tpl
+#     headers:
+#       Content-Type: '{{ template "main.headers.contentType" . }}'
+#   folderList:
+#     path: templates/folder-list.tpl
+#     headers:
+#       Content-Type: '{{ template "main.headers.contentType" . }}'
+#   badRequestError:
+#     path: templates/bad-request-error.tpl
+#     headers:
+#       Content-Type: '{{ template "main.headers.contentType" . }}'
+#   forbiddenError:
+#     path: templates/forbidden-error.tpl
+#     headers:
+#       Content-Type: '{{ template "main.headers.contentType" . }}'
+#   internalServerError:
+#     path: templates/internal-server-error.tpl
+#     headers:
+#       Content-Type: '{{ template "main.headers.contentType" . }}'
+#   notFoundError:
+#     path: templates/not-found-error.tpl
+#     headers:
+#       Content-Type: '{{ template "main.headers.contentType" . }}'
+#   unauthorizedError:
+#     path: templates/unauthorized-error.tpl
+#     headers:
+#       Content-Type: '{{ template "main.headers.contentType" . }}'

 # Authentication Providers
 # authProviders:
@@ -136,9 +159,9 @@ log:
 #           password:
 #             path: password1-in-file

-# Targets
+# Targets map
 targets:
-  - name: first-bucket
+  first-bucket:
     ## Mount point
     mount:
       path:
@@ -189,18 +212,20 @@ targets:
     #       authorizationOPAServer:
     #         # OPA server url with data path
     #         url: http://localhost:8181/v1/data/example/authz/allowed
-    # ## DEPRECATED Index document to display if exists in folder
-    # indexDocument: index.html
     # ## Actions
     # actions:
     #   # Action for GET requests on target
     #   GET:
     #     # Will allow GET requests
     #     enabled: true
-    #     # Redirect with trailing slash when a file isn't found
-    #     redirectWithTrailingSlashForNotFoundFile: true
-    #     # Index document to display if exists in folder
-    #     indexDocument: index.html
+    #     # Configuration for GET requests
+    #     config:
+    #       # Redirect with trailing slash when a file isn't found
+    #       redirectWithTrailingSlashForNotFoundFile: true
+    #       # Index document to display if exists in folder
+    #       indexDocument: index.html
+    #       # Allow to add headers to streamed files (can be templated)
+    #       streamedFileHeaders: {}
     #   # Action for PUT requests on target
     #   PUT:
     #     # Will allow PUT requests
@@ -229,30 +254,40 @@ targets:
     #     target: /$two/$one/$three/$one/
     ## Target custom templates
     # templates:
+    #   # Helpers
+    #   helpers:
+    #   - inBucket: false
+    #     path: ""
     #   # Folder list template
     #   folderList:
     #     inBucket: false
     #     path: ""
-    #   # Not found template
-    #   notFound:
+    #     headers: {}
+    #   # Not found error template
+    #   notFoundError:
     #     inBucket: false
     #     path: ""
+    #     headers: {}
     #   # Internal server error template
     #   internalServerError:
     #     inBucket: false
     #     path: ""
-    #   # Forbidden template
-    #   forbidden:
+    #     headers: {}
+    #   # Forbidden error template
+    #   forbiddenError:
     #     inBucket: false
     #     path: ""
-    #   # Unauthorized template
-    #   unauthorized:
+    #     headers: {}
+    #   # Unauthorized error template
+    #   unauthorizedError:
     #     inBucket: false
     #     path: ""
-    #   # BadRequest template
-    #   badRequest:
+    #     headers: {}
+    #   # Bad Request error template
+    #   badRequestError:
     #     inBucket: false
     #     path: ""
+    #     headers: {}
     ## Bucket configuration
     bucket:
       name: super-bucket
```
