# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# If your go.mod is in the root
COPY go.mod go.sum ./
# OR if it's in a backend subdirectory (as your previous tree showed):
# COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the source code - adjust path as needed
COPY . .
# OR if code is in backend folder:
# COPY backend/ .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/main .

# Adjust frontend path based on your structure
# If frontend is in root:
COPY frontend /frontend 2>/dev/null || true
# OR if frontend is elsewhere, adjust the path

RUN if [ -d "/frontend" ]; then ln -s /frontend /app/../frontend; fi

USER appuser

ENV PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["./main"]
