FROM golang:alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o saveany-bot .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/saveany-bot .

CMD ["./saveany-bot"]