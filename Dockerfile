# ==========================================
# Stage 1: Build binary using official Go
# ==========================================
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary with stripped symbols for minimal image size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=1.0.0" \
    -o /bin/pulseops \
    ./cmd/server

# ==========================================
# Stage 2: Minimal hardened production image
# ==========================================
FROM alpine:3.20

# Install CA certificates, tzdata, create dedicated non-root user, and prepare data directory
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup && \
    mkdir -p /data && chown -R appuser:appgroup /data

WORKDIR /app

# Copy binary from builder
COPY --from=builder /bin/pulseops /bin/pulseops

# Environment configurations
ENV PORT=8080 \
    DB_PATH=/data/pulseops.db \
    PROBE_INTERVAL_SECONDS=30 \
    DEFAULT_TARGET_URL=https://portfolioyusufjaelani.vercel.app \
    DEFAULT_TARGET_NAME="Yusuf Portfolio"

USER appuser:appgroup

EXPOSE 8080

VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/bin/pulseops"]
