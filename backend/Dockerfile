FROM golang:1.25-alpine

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN go build -o app .

EXPOSE 8080

CMD ["./app"]
