# Use Go 1.25.0 (matches Render)
FROM golang:1.25.0-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]
