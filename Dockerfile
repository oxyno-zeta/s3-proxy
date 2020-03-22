FROM alpine:3.11

RUN apk add --update ca-certificates && rm -rf /var/cache/apk/*

COPY s3-proxy /s3-proxy
COPY templates/ /templates/

ENTRYPOINT [ "/s3-proxy" ]