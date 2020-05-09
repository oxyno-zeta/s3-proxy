# Developer notes

## How to have a Keycloak for testing purpose

Run a docker container:

```bash
docker run -d --rm --name keycloak -p 9000:8080 -e KEYCLOAK_USER=admin -e KEYCLOAK_PASSWORD=admin jboss/keycloak:10.0.0
```
