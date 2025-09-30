FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared tracing package
COPY pkg/ /pkg/

# Copy service files
COPY shortener-service/go.mod ./
RUN go mod download

COPY shortener-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o shortener-service .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/shortener-service .

EXPOSE 3001

CMD ["./shortener-service"]
