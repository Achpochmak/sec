# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o proxy main.go

# Runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/proxy .
EXPOSE 8080
CMD ["./proxy"]
