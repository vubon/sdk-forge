# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# Using CGO_ENABLED=0 for static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-w -s" \
    -o sdk-forge \
    ./cmd/cli

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests (if fetching remote schemas)
RUN apk --no-cache add ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 sdkforge && \
    adduser -D -u 1000 -G sdkforge sdkforge

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/sdk-forge /usr/local/bin/sdk-forge

# Create directories for input/output
RUN mkdir -p /app/input /app/output && \
    chown -R sdkforge:sdkforge /app

# Switch to non-root user
USER sdkforge

# Set entrypoint
ENTRYPOINT ["sdk-forge"]

# Default command (show help)
CMD ["--help"]

