# Open Policy Agent (OPA)

S3-proxy integrate [Open Policy Agent](https://www.openpolicyagent.org/) for authorization process after OpenID Connect or Header based logins.

## Integration

This project integrate OPA with the REST API. You can see an example [here](https://www.openpolicyagent.org/docs/latest/integration/#integrating-with-the-rest-api). In the project configuration, you just have to put the link to the data endpoint with "allowed rego" path. Here is the example from the OPA website: http://localhost:8181/v1/data/example/authz/allow

## Input data

The following section will present the input data that s3-proxy will send to the Open Policy Agent.

```json linenums="1"
{
  "input": {
    "user": {
      "preferred_username": "user",
      "name": "Sample User",
      "groups": ["group1"],
      "given_name": "Sample",
      "family_name": "User",
      "email": "sample-user@example.com",
      "email_verified": true
    },
    "request": {
      "method": "GET",
      "protocol": "HTTP/1.1",
      "headers": {
        "accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
        "accept-encoding": "gzip, deflate, br",
        "accept-language": "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7",
        "cache-control": "max-age=0",
        "connection": "keep-alive",
        "cookie": "oidc=TOKEN",
        "sec-fetch-dest": "document",
        "sec-fetch-mode": "navigate",
        "sec-fetch-site": "none",
        "sec-fetch-user": "?1",
        "upgrade-insecure-requests": "1",
        "user-agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36"
      },
      "remoteAddr": "[::1]:51092",
      "scheme": "http",
      "host": "localhost:8080",
      "parsed_path": ["v2"],
      "path": "/v2/"
    },
    "tags": {
      "fake": "tag"
    }
  }
}
```

## Output Data

Here is an example of the expected output schema:

```json linenums="1"
{
  "result": true
}
```
