FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

WORKDIR /workspace

COPY go.mod .
COPY cmd/ cmd/
COPY pkg/ pkg/

RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -o /bin/dexsidecar ./cmd/dexsidecar

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/dexsidecar /bin/dexsidecar

ENTRYPOINT ["/bin/dexsidecar"]
