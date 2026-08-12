FROM alpine:3

RUN apk add --no-cache ca-certificates \
    && addgroup -S yaps && adduser -S yaps -G yaps \
    && mkdir -p /data && chown yaps:yaps /data

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/yaps /usr/local/bin/yaps

USER yaps
WORKDIR /data

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/yaps"]