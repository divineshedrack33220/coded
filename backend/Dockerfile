# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Copy go.mod (go.sum optional, but you have it now → good)
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy all backend source code
COPY backend/ ./

# Build the static binary
# Adjust path if your main is not directly in backend/main.go
# (e.g. ./cmd/main if using cmd/ structure)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./main.go

# Stage 2: Minimal runtime image (scratch = super small)
FROM scratch

# Copy only the binary
COPY --from=builder /app/server /server

# If your app makes outbound HTTPS calls (e.g. to APIs), uncomment this:
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Tell Docker what port your app listens on (check your main.go for the actual one!)
# Common defaults: 8080, 3000, 80 — change if needed
EXPOSE 8080

# Run the app
CMD ["/server"]
