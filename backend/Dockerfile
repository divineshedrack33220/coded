# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Run tidy
RUN go mod tidy

# Build ONLY main.go (exclude generate_vapid.go)
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary
COPY --from=builder /app/main .

# Copy frontend if it exists
COPY --from=builder /app/frontend /frontend 2>/dev/null || true

# Create symlink for frontend
RUN if [ -d "/frontend" ]; then \
        ln -sf /frontend /app/../frontend; \
    else \
        mkdir -p /app/../frontend; \
        echo "Warning: No frontend directory found - API only mode"; \
    fi

USER appuser

ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017
ENV JWT_SECRET=change-this-in-production

EXPOSE 8080

CMD ["./main"]
