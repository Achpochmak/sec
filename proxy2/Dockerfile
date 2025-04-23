FROM golang:1.20.7-alpine

COPY . project

RUN apk add --no-cache openssl && \
    mkdir -p /app/https && \
    cd /app/https && \
    openssl req -new -newkey rsa:4096 -days 365 -nodes -x509 \
    -subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=localhost" \
    -keyout cert.key -out cert.crt

RUN cd project && \
    go build -o http-proxy http-proxy/app && \
    chmod +x https/gen.sh && \
    mkdir certs && \
    apk add openssl

EXPOSE 8080/tcp
EXPOSE 8000/tcp

CMD cd project && ./http-proxy