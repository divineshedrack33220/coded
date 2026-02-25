# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy from backend folder
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/main .

# Simple approach - just try to copy and ignore errors
COPY frontend /frontend/

# Create symlink (will work even if frontend is empty)
RUN ln -sf /frontend /app/../frontend || mkdir -p /app/../frontend

USER appuser

ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017/coded
ENV JWT_SECRET=change-this-in-production

EXPOSE 8080

CMD ["./main"]
