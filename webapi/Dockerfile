FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod init webapi && go get github.com/gorilla/mux@latest
RUN go build -o webapi main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/webapi .
EXPOSE 8000
CMD ["./webapi"]
