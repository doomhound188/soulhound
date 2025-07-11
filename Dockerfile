# Multi-stage build for SoulHound Discord Music Bot

# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files first to leverage Docker layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o soulhound cmd/main.go

# Runtime stage - using scratch for minimal size, or alpine for debugging
FROM alpine:latest

# Install ffmpeg and yt-dlp for audio processing
RUN apk update && apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip && \
    pip3 install --no-cache-dir --break-system-packages yt-dlp

# Create non-root user for security
RUN addgroup -g 1001 -S soulhound && \
    adduser -u 1001 -S soulhound -G soulhound

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/soulhound .

# Change ownership to non-root user
RUN chown -R soulhound:soulhound /app

# Switch to non-root user
USER soulhound

# Expose port (if needed for health checks or metrics in future)
EXPOSE 8080

# Health check (optional, for future use)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD pgrep soulhound || exit 1

# Run the application
CMD ["./soulhound"]
