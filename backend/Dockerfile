# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary
COPY --from=builder /app/main .

# Handle frontend properly - using RUN commands instead of COPY with echo
RUN mkdir -p /frontend
COPY frontend /frontend/ 2>/dev/null || true

# Check if frontend files were copied and create symlink
RUN if [ -f "/frontend/index.html" ] || [ -d "/frontend/css" ]; then \
        echo "✅ Frontend files found, creating symlink"; \
        ln -sf /frontend /app/../frontend; \
    else \
        echo "⚠️  No frontend files - running in API-only mode"; \
        mkdir -p /app/../frontend; \
    fi

USER appuser

ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017/coded
ENV JWT_SECRET=change-this-in-production

EXPOSE 8080

CMD ["./main"]
