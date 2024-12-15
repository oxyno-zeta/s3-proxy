FROM alpine:3.21

ENV USER=proxy
ENV UID=1000
ENV GID=1000

RUN apk add --update ca-certificates && rm -rf /var/cache/apk/* && \
    addgroup -g $GID $USER && \
    adduser -D -g "" -h "/proxy" -G "$USER" -H -u "$UID" "$USER"

WORKDIR /proxy

COPY s3-proxy /proxy/s3-proxy
COPY templates/ /proxy/templates/

RUN chown -R 1000:1000 /proxy

USER proxy

ENTRYPOINT [ "/proxy/s3-proxy" ]
