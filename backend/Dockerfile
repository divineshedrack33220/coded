# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Create a directory for frontend files (if you want to include them)
# Note: Your code expects frontend at ../frontend, but in Docker it's better to adjust
# or copy frontend files to the expected location
RUN mkdir -p /frontend

# If you have frontend files, copy them (adjust path as needed)
# COPY --from=builder /app/../frontend /frontend

# Set environment variables (can be overridden at runtime)
ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://mongodb:27017
ENV JWT_SECRET=your-secret-key-change-this

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
