FROM golang:1.24-bullseye AS builder

WORKDIR /app

# Copy go.mod and go.sum to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -o podstat .

# Use minimal alpine image for the final container
FROM alpine:latest

# Install certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /app

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/podstat /app/podstat

# Set environment variables
ENV PODSTAT_DOCKER=true

ENTRYPOINT ["/app/podstat"]

# Default command (can be overridden at runtime)
CMD ["--help"]