FROM scratch

COPY s3-proxy /s3-proxy
COPY templates/ /templates/

ENTRYPOINT [ "/s3-proxy" ]
