FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

WORKDIR /workspace

COPY go.mod .
COPY cmd/ cmd/
COPY pkg/ pkg/

RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -o /bin/dexsidecar .

FROM debian:bookworm-slim

COPY --from=builder /bin/dexsidecar /bin/dexsidecar

ENTRYPOINT ["/bin/dexsidecar"]
