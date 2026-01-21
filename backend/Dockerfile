# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Set working dir to where your code lives
WORKDIR /src

# Copy go mod files first → better caching
COPY backend/go.mod backend/go.sum ./

# Download deps
RUN go mod download

# Copy entire backend source
COPY backend/ ./

# Build the binary (adjust -o path/name if your main is elsewhere)
# -ldflags="-s -w" makes it smaller
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./main.go   # ← change ./main.go to your actual entry file if different, e.g. ./cmd/main.go

# Stage 2: Tiny runtime image
FROM scratch

# Copy the binary from builder
COPY --from=builder /app/server /server

# If you need env vars, certs, etc. copy them here too
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose your app port (change if not 8080)
EXPOSE 8080

# Run the binary
CMD ["/server"]
