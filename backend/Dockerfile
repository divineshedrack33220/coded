# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Copy only go.mod first (go.sum optional)
COPY backend/go.mod ./

# If go.sum exists, copy it too — but don't fail if missing
# Use --mount=type=bind to avoid error, but for simplicity we skip go.sum copy
RUN go mod download

# Now copy the entire backend source
COPY backend/ ./

# Build - adjust entry point if main is not at backend/main.go
# Here assuming main.go is directly in backend/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./main.go

# Stage 2: Tiny runtime
FROM scratch

COPY --from=builder /app/server /server

# If your app needs ca-certificates for HTTPS outbound calls, uncomment:
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080   # ← change to your actual port (check main.go)

CMD ["/server"]
