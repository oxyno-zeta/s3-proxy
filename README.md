# s3-proxy

[![CircleCI](https://circleci.com/gh/oxyno-zeta/s3-proxy/tree/master.svg?style=svg)](https://circleci.com/gh/oxyno-zeta/s3-proxy/tree/master) [![Go Report Card](https://goreportcard.com/badge/github.com/oxyno-zeta/s3-proxy)](https://goreportcard.com/report/github.com/oxyno-zeta/s3-proxy) ![Docker Pulls](https://img.shields.io/docker/pulls/oxynozeta/s3-proxy.svg)

## Why ?

Yes this is another S3 proxy written in Golang.

I've created this project because I couldn't find any other that allow to proxy multiple S3 buckets or to have custom templates with OpenID Connect authentication.

## Features

- Multi S3 bucket proxy
- Index document (display index document instead of listing when found)
- Custom templates
- AWS S3 Login from files or environment variables
- Custom S3 endpoints supported
- Basic Authentication support
- OpenID Connect Authentication support

## Documentation

See full documentation [here](tree/master/docs).

## Inspired by

- [pottava/aws-s3-proxy](https://github.com/pottava/aws-s3-proxy)

## Thanks

- My wife BH to support me doing this

## Author

- Oxyno-zeta (Havrileck Alexandre)

## License

Apache 2.0 (See in LICENSE)
