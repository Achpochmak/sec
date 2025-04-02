# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o webapi main.go

# Runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/webapi .
EXPOSE 8000
CMD ["./webapi"]
