FROM --platform=$BUILDPLATFORM golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab AS build

WORKDIR /src
ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/framoryx-worker-extract ./cmd

FROM scratch AS runtime

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chmod=0555 /out/framoryx-worker-extract /usr/local/bin/framoryx-worker-extract

USER 65532:65532
WORKDIR /
ENTRYPOINT ["/usr/local/bin/framoryx-worker-extract"]
