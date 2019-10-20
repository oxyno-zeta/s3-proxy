# Deployment

## Kubernetes - Helm

A helm chart have been created to deploy this in a Kubernetes cluster.

You can find it here: [https://github.com/oxyno-zeta/helm-charts/tree/master/stable/s3-proxy](https://github.com/oxyno-zeta/helm-charts/tree/master/stable/s3-proxy)

## Docker

First, write the configuration file in a config folder. That one will be mounted.

Run this command:

```shell
docker run -d --name s3-proxy -p 8080:8080 -p 9090:9090 -v $PWD/config:/config oxynozeta/s3-proxy
```
