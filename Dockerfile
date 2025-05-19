FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/krau/SaveAny-Bot/common.Version=Docker" -o saveany-bot .

FROM alpine:latest

RUN addgroup -S saveany && adduser -S saveany -G saveany

WORKDIR /app

COPY --from=builder /app/saveany-bot .

RUN mkdir -p /app/data /app/downloads /app/cache && \
    chown -R saveany:saveany /app /app/downloads /app/cache /app/data

RUN chmod +x /app/saveany-bot

USER saveany

ENTRYPOINT  ["/app/saveany-bot"]
