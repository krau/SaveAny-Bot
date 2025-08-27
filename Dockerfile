FROM golang:alpine AS builder

ARG VERSION="dev"
ARG GitCommit="Unknown"
ARG BuildTime="Unknown"

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \
    go build -trimpath \
    -ldflags=" \
    -s -w \
    -X 'github.com/krau/SaveAny-Bot/config.Version=${VERSION}' \
    -X 'github.com/krau/SaveAny-Bot/config.GitCommit=${GitCommit}' \
    -X 'github.com/krau/SaveAny-Bot/config.BuildTime=${BuildTime}' \
    " \
    -o saveany-bot .

FROM alpine:latest

RUN apk add --no-cache curl

WORKDIR /app

COPY --from=builder /app/saveany-bot .
COPY entrypoint.sh .

RUN chmod +x /app/saveany-bot && \
    chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
