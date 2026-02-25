FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN adduser -D appuser

WORKDIR /app

COPY --from=builder /app/main .

USER appuser

EXPOSE 8080

CMD ["./main"]
