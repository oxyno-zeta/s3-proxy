<h1 align="center"><img width="350" height="350" src="https://raw.githubusercontent.com/oxyno-zeta/s3-proxy/master/docs/logo/logo.png" /></h1>

<p align="center">
  <a href="https://github.com/avelino/awesome-go" rel="noopener noreferer" target="_blank"><img src="https://awesome.re/mentioned-badge.svg" alt="Mentioned in Awesome Go" /></a>
  <a href="http://godoc.org/github.com/oxyno-zeta/s3-proxy" rel="noopener noreferer" target="_blank"><img src="https://img.shields.io/badge/godoc-reference-blue.svg" alt="Go Doc" /></a>
  <a href="https://circleci.com/gh/oxyno-zeta/s3-proxy" rel="noopener noreferer" target="_blank"><img src="https://circleci.com/gh/oxyno-zeta/s3-proxy.svg?style=svg" alt="CircleCI" /></a>
  <a href="https://goreportcard.com/report/github.com/oxyno-zeta/s3-proxy" rel="noopener noreferer" target="_blank"><img src="https://goreportcard.com/badge/github.com/oxyno-zeta/s3-proxy" alt="Go Report Card" /></a>
</p>
<p align="center">
  <a href="https://coveralls.io/github/oxyno-zeta/s3-proxy?branch=master" rel="noopener noreferer" target="_blank"><img src="https://coveralls.io/repos/github/oxyno-zeta/s3-proxy/badge.svg?branch=master" alt="Coverage Status" /></a>
  <a href="https://hub.docker.com/r/oxynozeta/s3-proxy" rel="noopener noreferer" target="_blank"><img src="https://img.shields.io/docker/pulls/oxynozeta/s3-proxy.svg" alt="Docker Pulls" /></a>
  <a href="https://github.com/oxyno-zeta/s3-proxy/blob/master/LICENSE" rel="noopener noreferer" target="_blank"><img src="https://img.shields.io/github/license/oxyno-zeta/s3-proxy" alt="GitHub license" /></a>
  <a href="https://github.com/oxyno-zeta/s3-proxy/releases" rel="noopener noreferer" target="_blank"><img src="https://img.shields.io/github/v/release/oxyno-zeta/s3-proxy" alt="GitHub release (latest by date)" /></a>
</p>

## Why ?

First of all, yes, this is another S3 proxy written in Golang.

I've created this project because I couldn't find any other that allow to proxy multiple S3 buckets or to have custom templates with OpenID Connect authentication and also to get, upload and delete files.

## Features

- Multi S3 bucket proxy
- Index document (display index document instead of listing when found)
- Custom templates
- AWS S3 Login from files or environment variables
- Custom S3 endpoints supported
- Basic Authentication support
- Multiple Basic Authentication support
- OpenID Connect Authentication support
- Multiple OpenID Connect Provider support
- Redirect to original host and path with OpenID Connect authentication
- Target mount point configuration with hostname and multiple path support
- Authentication by path and http method on each bucket
- Allow to publish files on S3 bucket
- Allow to delete files on S3 bucket
- Open Policy Agent integration for authorizations
- Configuration hot reload
- CORS support
- Prometheus metrics
- S3 Key Rewrite possibility

See more information on these features in the "Feature Guide".

## Advanced interfaces

Looking for more advanced interfaces. Take a look on this project: [s3-proxy-interfaces](https://github.com/oxyno-zeta/s3-proxy-interfaces).

Provided interfaces in the project are really simple and based on NGinX template. Those are React based, with Material design and with customization.

## Want to contribute ?

- Read the [CONTRIBUTING guide](./CONTRIBUTING.md)

## Inspired by

- [pottava/aws-s3-proxy](https://github.com/pottava/aws-s3-proxy)

## Thanks

- My wife BH to support me doing this

## Author

- Oxyno-zeta (Havrileck Alexandre)

## License

Apache 2.0 (See in LICENSE)
