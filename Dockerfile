# Multi-stage Dockerfile for golang-dhcpcd
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Install git for go generate
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate embedded files and build binary
RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o golang-dhcpcd main.go

# Final stage - minimal runtime image
FROM alpine:latest

# Install necessary packages for network operations
RUN apk --no-cache add ca-certificates iproute2

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/golang-dhcpcd .

# Make binary executable
RUN chmod +x /app/golang-dhcpcd

# Health check (optional)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/golang-dhcpcd", "--help"]

# Set entrypoint
ENTRYPOINT ["/app/golang-dhcpcd"]
CMD ["--help"]
