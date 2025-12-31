# Use the official Go 1.25.5 image
FROM golang:1.25.5

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the app
RUN go build -tags netgo -ldflags "-s -w" -o app ./backend

# Expose the port your app uses (replace with your PORT)
EXPOSE 10000

# Run the app
CMD ["./backend/app"]
