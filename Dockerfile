# SmartTicket Dockerfile
# Multi-stage build for Go backend application

# Build stage
FROM m.daocloud.io/docker.io/library/golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o smartticket \
    cmd/server/main.go

# Production stage
FROM m.daocloud.io/docker.io/library/alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata sqlite

# Create non-root user
RUN addgroup -g 1001 -S smartticket && \
    adduser -u 1001 -S smartticket -G smartticket

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/smartticket /app/smartticket

# Copy configuration files
COPY --from=builder /build/configs /app/configs

# Create necessary directories
RUN mkdir -p /app/data /app/logs /app/backups && \
    chown -R smartticket:smartticket /app

# Switch to non-root user
USER smartticket

# Expose port (using non-standard port 6533)
EXPOSE 6533

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:6533/api/v1/health || exit 1

# Set environment variables
ENV GIN_MODE=release
ENV PORT=6533

# Run the application
CMD ["./smartticket", "serve", "--config", "configs/config.prod.yaml"]