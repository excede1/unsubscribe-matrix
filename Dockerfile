# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

# Install ca-certificates and curl for HTTPS requests and health checks
RUN apk --no-cache add ca-certificates curl

# Create app directory
WORKDIR /app

# Create logs directory for persistent logging
RUN mkdir -p /app/logs

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy views directory (templates)
COPY --from=builder /app/views ./views

# Copy assets directory
COPY --from=builder /app/assets ./assets

# Create a non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

# Change ownership of app directory to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port 3000
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/ping || exit 1

# Command to run the application
CMD ["./main"]