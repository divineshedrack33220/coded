# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

# Set working directory to /app
WORKDIR /app

# Copy go mod files from backend folder
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the entire backend source
COPY backend/ .

# Run tidy to ensure dependencies are clean
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy frontend files to the location your app expects (../frontend)
COPY frontend /frontend

# Create the symlink that your app expects (../frontend)
RUN ln -s /frontend /app/../frontend

USER appuser

ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017/coded
ENV JWT_SECRET=change-this-in-production

EXPOSE 8080

CMD ["./main"]
