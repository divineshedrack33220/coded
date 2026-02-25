# ===== Builder Stage =====
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy backend module files
COPY backend/go.mod backend/go.sum ./ 
RUN go mod download

# Copy backend source
COPY backend/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go


# ===== Runtime Stage =====
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D appuser

WORKDIR /app

# Copy backend binary
COPY --from=builder /app/main .

# Copy frontend folder (IMPORTANT ⭐)
COPY frontend ./frontend

USER appuser

ENV PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["./main"]
