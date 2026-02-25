# ----------- Build Stage -----------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git (needed for some Go packages)
RUN apk add --no-cache git

# Copy go mod first (for caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy rest of backend code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go


# ----------- Production Stage -----------
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates
RUN adduser -D appuser

# Copy compiled binary
COPY --from=builder /app/main .

USER appuser

ENV PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["./main"]
