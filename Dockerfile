FROM golang:1.17-alpine
WORKDIR /src/
COPY . /src
RUN apk add -U make bash curl git && make code/build

FROM alpine:3.16

ENV USER=proxy
ENV UID=1000
ENV GID=1000

RUN apk add --update ca-certificates && rm -rf /var/cache/apk/* && \
    addgroup -g $GID $USER && \
    adduser -D -g "" -h "/proxy" -G "$USER" -H -u "$UID" "$USER"

WORKDIR /proxy

COPY --from=0 /src/bin/s3-proxy /proxy/s3-proxy
COPY templates/ /proxy/templates/

RUN chown -R 1000:1000 /proxy

USER proxy

EXPOSE 8080
ENTRYPOINT [ "/proxy/s3-proxy" ]
