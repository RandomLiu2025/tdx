# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG ALPINE_MIRROR=mirrors.aliyun.com
ARG GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct
ARG TARGETOS
ARG TARGETARCH

RUN sed -i "s/dl-cdn.alpinelinux.org/${ALPINE_MIRROR}/g" /etc/apk/repositories

ENV GOPROXY=${GOPROXY}

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/tdx-api ./cmd/tdx-api

FROM alpine:3.20

ARG ALPINE_MIRROR=mirrors.aliyun.com

RUN sed -i "s/dl-cdn.alpinelinux.org/${ALPINE_MIRROR}/g" /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata \
    && addgroup -S tdx \
    && adduser -S -G tdx tdx

COPY --from=builder --chown=tdx:tdx /out/tdx-api /usr/local/bin/tdx-api

USER tdx

ENV TDX_HTTP_ADDR=:8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=30s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/ready || exit 1

STOPSIGNAL SIGTERM

ENTRYPOINT ["/usr/local/bin/tdx-api"]
