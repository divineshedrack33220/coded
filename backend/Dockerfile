# Build stage - use a newer Go version
FROM golang:1.23-alpine AS builder

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

# Copy frontend if it exists (adjust path as needed)
# If frontend is in the same repo:
COPY --from=builder /app/frontend /frontend

# Create symlink for frontend (since code expects ../frontend)
RUN ln -s /frontend /app/../frontend || true

# Set environment variables (can be overridden at runtime)
ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017
ENV JWT_SECRET=your-secret-key-change-this-in-production

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
