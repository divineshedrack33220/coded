# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# If go.mod is in the current directory (not in backend folder)
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code (assuming it's in current dir)
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/main .

# If frontend folder exists at same level
COPY frontend /frontend 2>/dev/null || echo "No frontend folder found"

RUN if [ -d "/frontend" ]; then \
        ln -sf /frontend /app/../frontend; \
    else \
        mkdir -p /app/../frontend; \
        echo "Warning: No frontend - API only mode"; \
    fi

USER appuser

ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017/coded
ENV JWT_SECRET=change-this-in-production

EXPOSE 8080

CMD ["./main"]
