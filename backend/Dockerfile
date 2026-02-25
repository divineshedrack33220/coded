# Build stage - use Go 1.24
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/main .

# Copy frontend if it exists
COPY --from=builder /app/frontend /frontend 2>/dev/null || true

# Create symlink for frontend if it exists
RUN if [ -d "/frontend" ]; then ln -sf /frontend /app/../frontend; fi

USER appuser

ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017
ENV JWT_SECRET=your-secret-key-change-this-in-production

EXPOSE 8080

CMD ["./main"]
