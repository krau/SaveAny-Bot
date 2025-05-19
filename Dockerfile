FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/krau/SaveAny-Bot/common.Version=Docker" -o saveany-bot .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/saveany-bot .

ENTRYPOINT  ["/app/saveany-bot"]
