FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared tracing package
COPY pkg/ /pkg/

# Copy service files
COPY analytics-service/go.mod ./
RUN go mod download

COPY analytics-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o analytics-service .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/analytics-service .

EXPOSE 3003

CMD ["./analytics-service"]
