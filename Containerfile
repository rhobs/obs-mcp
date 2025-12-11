FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd cmd
COPY pkg pkg

ENV GOOS=${TARGETOS:-linux}
# GOARCH has no default, so the binary builds for the host. On Apple M1, BUILDPLATFORM is set to
# linux/arm64; on Apple x86, it's linux/amd64. Leaving it empty ensures the container and binary
# match the host platform.
ENV GOARCH=${TARGETARCH}
ENV CGO_ENABLED=1
ENV GOFLAGS=-mod=readonly
RUN go build -tags strictfipsruntime -o obs-mcp ./cmd/obs-mcp

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
WORKDIR /app
COPY --from=builder /app/obs-mcp /app/obs-mcp
USER 65532:65532

ENTRYPOINT ["/app/obs-mcp", "--listen", ":8080", "--auth-mode", "header"]

EXPOSE 8080
