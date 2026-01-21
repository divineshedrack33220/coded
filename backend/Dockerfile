# Stage 1: Build the Go app
FROM golang:1.23-alpine AS builder

# Work in a temp dir
WORKDIR /src

# Copy go.mod & go.sum from backend/ subfolder
COPY backend/go.mod backend/go.sum ./

# Download dependencies (caching layer)
RUN go mod download

# Copy the entire backend source code
COPY backend/ ./

# Build the binary from main.go in this dir
# -o /app/server = output binary name/path
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./main.go

# Stage 2: Very small final image
FROM scratch

# Copy only the compiled binary
COPY --from=builder /app/server /server

# If you need outbound HTTPS (Google APIs, etc.), uncomment:
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Port your Gin app listens on (check main.go for r.Run(":xxxx"))
# Default Gin is :8080 — change if yours is different (e.g. :3000)
EXPOSE 8080

# Run the app
CMD ["/server"]
