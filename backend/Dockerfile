# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files from the correct location
# Based on your tree output, go.mod is in the backend folder
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy all backend source code
COPY backend/ .

# Run tidy to ensure dependencies are clean
RUN go mod tidy

# Build the application - pointing to main.go in current dir (which is backend/)
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# IMPORTANT FIX: Copy frontend from the correct location in build context
# The frontend folder is at the same level as backend in your repo
COPY frontend /frontend

# Create the symlink that your app expects (../frontend)
RUN ln -sf /frontend /app/../frontend

# Ensure the symlink works and set permissions
RUN chown -R appuser:appuser /app && \
    chown -R appuser:appuser /frontend

USER appuser

ENV PORT=8080
ENV GIN_MODE=release
ENV MONGODB_URI=mongodb://localhost:27017/coded
ENV JWT_SECRET=change-this-in-production

EXPOSE 8080

CMD ["./main"]
