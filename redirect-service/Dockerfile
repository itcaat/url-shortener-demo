FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared tracing package
COPY pkg/ /pkg/

# Copy service files
COPY redirect-service/go.mod ./
RUN go mod download

COPY redirect-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o redirect-service .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/redirect-service .

EXPOSE 3002

CMD ["./redirect-service"]
