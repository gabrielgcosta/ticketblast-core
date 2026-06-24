# ==============================================================================
# STAGE 1: Builder
# ==============================================================================
FROM golang:1.24-alpine AS builder

# Install system dependencies needed for building Go binaries (CGO-free)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency files first to leverage Docker cache layers
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Target application build argument (defaults to 'api', can be overridden to 'worker')
ARG TARGET=api

# Compile the binary with optimizations:
# - CGO_ENABLED=0 for a fully static binary
# - ldflags="-s -w" to strip debug information and symbols (reducing binary size)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/service \
    ./cmd/${TARGET}/main.go

# ==============================================================================
# STAGE 2: Runner (Minimalist and secure Alpine)
# ==============================================================================
FROM alpine:3.19 AS runner

# Import certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create a non-privileged user and group for security (running as root is a vulnerability)
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /app/bin/service /app/service

# Use the non-root user
USER appuser

# Expose the API gateway port
EXPOSE 8080

# Run the service
ENTRYPOINT ["/app/service"]
