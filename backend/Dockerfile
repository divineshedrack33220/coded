# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files (no backend/ prefix anymore)
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source (current directory)
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary
COPY --from=builder /app/main .

USER appuser

ENV PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["./main"]
