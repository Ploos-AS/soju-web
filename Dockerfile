# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.24
ARG GO_ALPINE_VERSION=3.21
ARG ALPINE_VERSION=3.22
ARG VERSION=dev
ARG REVISION=unknown

FROM golang:${GO_VERSION}-alpine${GO_ALPINE_VERSION} AS build
ARG VERSION
ARG REVISION
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go test ./... && \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X main.version=${VERSION} -X main.revision=${REVISION}" \
      -o /out/soju-web .

FROM alpine:${ALPINE_VERSION}
ARG VERSION=dev
ARG REVISION=unknown
LABEL org.opencontainers.image.title="soju-web" \
      org.opencontainers.image.description="Go WebAdmin for the soju IRC bouncer" \
      org.opencontainers.image.source="https://github.com/Ploos-AS/soju-web" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$REVISION" \
      org.opencontainers.image.licenses="MIT"
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 -S sojuweb \
    && adduser -u 1000 -S -D -H -h /nonexistent -s /sbin/nologin -G sojuweb sojuweb
COPY --from=build /out/soju-web /usr/local/bin/soju-web
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/soju-web"]
