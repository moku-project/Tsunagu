FROM gradle:8.9-jdk21 AS sandbox-build
WORKDIR /src
COPY sandbox/ ./sandbox/
COPY proto/ ./proto/
WORKDIR /src/sandbox
RUN gradle shadowJar jlinkRuntime -x test --no-daemon

FROM golang:1.23-bookworm AS backend-build
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential libvips-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY backend/go.mod ./
RUN go mod download || true
COPY backend/ ./
RUN go build -o /out/tsunagu ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates libvips libstdc++6 curl gosu \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r tsunagu && useradd -r -g tsunagu -d /app tsunagu

WORKDIR /app

COPY --from=backend-build /out/tsunagu /app/tsunagu-server
COPY --from=sandbox-build /src/sandbox/build/libs/*-all.jar /app/sandbox/sandbox.jar
COPY --from=sandbox-build /src/sandbox/build/runtime /app/sandbox/runtime
COPY docker/docker-entrypoint.sh /app/docker-entrypoint.sh

RUN mkdir -p /app/sandbox/extensions /app/jar-cache /app/data \
    && chown -R tsunagu:tsunagu /app \
    && chmod +x /app/docker-entrypoint.sh

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/app/docker-entrypoint.sh"]
