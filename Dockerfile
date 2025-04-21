# Build stage
FROM golang:1.22-alpine AS builder

# Install git and CA certificates for Go modules
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app

LABEL authors="nikitashilov"

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o cv_builder ./cmd/server/main.go

# Final stage
FROM alpine:3.19

# Install CA certificates for HTTPS
RUN apk update && apk add --no-cache ca-certificates tzdata && update-ca-certificates

# Set up a non-root user for security
RUN adduser -D -g '' appuser
USER appuser

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/cv_builder .

# Copy migrations if your app uses them
COPY --from=builder /app/migrations ./migrations

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./cv_builder"]