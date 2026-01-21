# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Set workdir to source
WORKDIR /src

# Copy go.mod and go.sum (go.sum optional — skips if missing)
COPY backend/go.mod ./
COPY backend/go.sum ./  # Safe: no error if absent

# Grab deps for cache
RUN go mod download

# Copy all backend code
COPY backend/ ./

# Build static binary — points to your main.go in backend/
# (tweaks: CGO off for tiny size, linux target for containers)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./main.go

# Stage 2: Super small runtime (no OS bloat)
FROM scratch

# Just the binary
COPY --from=builder /app/server /server

# For HTTPS outbound (e.g. API calls) — uncomment if needed
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Your app's port (check main.go — e.g. :8080 or :3000?)
EXPOSE 8080

# Fire it up
CMD ["/server"]
