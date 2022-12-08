# Deployment

## Requirements

To deploy, you must configure the application. You can read more about this [here](../configuration/structure.md).

## Kubernetes - Helm

A helm chart have been created to deploy this in a Kubernetes cluster.

You can find it here: [https://artifacthub.io/packages/helm/oxyno-zeta/s3-proxy](https://artifacthub.io/packages/helm/oxyno-zeta/s3-proxy)

Or directly from source: [https://github.com/oxyno-zeta/helm-charts-v2/tree/master/charts/s3-proxy](https://github.com/oxyno-zeta/helm-charts-v2/tree/master/charts/s3-proxy)

<!-- prettier-ignore-start -->
!!! Note
    This chart allow the configuration hot reload. If you change a configuration file and apply it, the Configmap will be updated in Kubernetes and Kubernetes will change the file mounted and linked to Configmap. This will take around 1 minute (according to my tests).
<!-- prettier-ignore-end -->

## Docker

First, write the configuration file in a config folder. That one will be mounted.

Run this command:

```bash
docker run -d --name s3-proxy -p 8080:8080 -p 9090:9090 -v $PWD/conf:/proxy/conf oxynozeta/s3-proxy
```
