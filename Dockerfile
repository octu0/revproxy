FROM golang:1.15-alpine3.12 as builder

WORKDIR /build
ADD . /go/src/github.com/octu0/revproxy

RUN set -eux && \
    apk add --no-cache --virtual .build-deps git make openssh-client && \
    cd /go/src/github.com/octu0/revproxy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a \
      -tags netgo -installsuffix netgo --ldflags '-extldflags "-static"'  \
      -o /build/revproxy \
        cmd/main.go \
        cmd/server.go \
      && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a \
      -tags netgo -installsuffix netgo --ldflags '-extldflags "-static"'  \
      -o /build/revproxy_darwin \
        cmd/main.go \
        cmd/server.go \
      && \
    /build/revproxy --version && \
    apk del .build-deps

# ----------------------------------

FROM alpine:3.12

RUN addgroup revproxy && \
    adduser -S -G revproxy -u 999 revproxy

WORKDIR /app
COPY --from=builder /build/   /app/
RUN set -eux && \
    apk add --no-cache ca-certificates curl dumb-init openssl su-exec && \
    /app/revproxy --version

EXPOSE 8080
VOLUME [ "/app" ]

COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT [ "docker-entrypoint.sh" ]
