FROM golang:alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o saveany-bot .

FROM alpine:latest

RUN addgroup -S saveany && adduser -S saveany -G saveany

WORKDIR /app

RUN mkdir -p /app/data /app/downloads /app/cache && \
    chown -R saveany:saveany /app

COPY --from=builder /app/saveany-bot .

RUN chmod +x /app/saveany-bot

USER saveany

CMD ["./saveany-bot"]
