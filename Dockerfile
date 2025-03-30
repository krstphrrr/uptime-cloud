# stage 1: build the app 
FROM golang:1.24.1 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o uptime-monitor
# stage 2: install ca-certificates
FROM alpine AS certs
RUN apk --no-cache add ca-certificates

# stage 3: copy the ca-certificates and the binary to a scratch image
FROM scratch
COPY --from=builder /app/config.development.json /config.development.json
COPY --from=builder /app/uptime-monitor /uptime-monitor
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

CMD ["/uptime-monitor"]
