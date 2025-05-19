FROM golang:alpine AS builder

ARG VERSION="dev"
ARG GiTCommit="Unknown"
ARG BuildTime="Unknown"

WORKDIR /app

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \ 
    go build -trimpath \ 
    -ldflags "-s -w \ 
    -X github.com/krau/SaveAny-Bot/common.Version=${VERSION} \
    -X github.com/krau/SaveAny-Bot/common.GitCommit=${GiTCommit} \
    -X github.com/krau/SaveAny-Bot/common.BuildTime=${BuildTime}" \
    -o saveany-bot .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/saveany-bot .

ENTRYPOINT  ["/app/saveany-bot"]
