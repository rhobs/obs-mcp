ARG GOLANG_BUILDER=1.25

FROM golang:${GOLANG_BUILDER} AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
COPY cmd/ cmd/
COPY pkg/ pkg/

ENV GOOS=${TARGETOS:-linux}
# GOARCH has no default, so the binary builds for the host. On Apple M1, BUILDPLATFORM is set to
# linux/arm64; on Apple x86, it's linux/amd64. Leaving it empty ensures the container and binary
# match the host platform.
ENV GOARCH=${TARGETARCH}
ENV CGO_ENABLED=1
ENV GOFLAGS=-mod=readonly

RUN --mount=type=cache,target=/go/pkg/mod \
      --mount=type=cache,target=/root/.cache/go-build \
      go mod download -x && go mod verify

RUN --mount=type=cache,target=/go/pkg/mod \
      --mount=type=cache,target=/root/.cache/go-build \
      go build -trimpath -tags strictfipsruntime -ldflags '-s -w' -o obs-mcp ./cmd/obs-mcp

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

LABEL org.opencontainers.image.source="https://github.com/rhobs/obs-mcp" \
      org.opencontainers.image.description="Observability MCP server for Prometheus and Alertmanager" \
      org.opencontainers.image.licenses="Apache-2.0"

WORKDIR /app
COPY --from=builder /app/obs-mcp /app/obs-mcp
USER 65532:65532
ENTRYPOINT ["/app/obs-mcp"]
CMD ["--listen", ":8080", "--auth-mode", "header"]

EXPOSE 8080
