FROM node:26-alpine AS web
WORKDIR /ui
RUN npm install -g pnpm
COPY web/ui/package.json web/ui/pnpm-lock.yaml web/ui/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ui ./
RUN pnpm run build

FROM golang:1.27-alpine AS builder

ARG VERSION="dev"
ARG GitCommit="Unknown"
ARG BuildTime="Unknown"

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
COPY --from=web /dist ./web/dist
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \
    go build -trimpath \
    -ldflags=" \
    -s -w \
    -X 'github.com/krau/SaveAny-Bot/config.Version=${VERSION}' \
    -X 'github.com/krau/SaveAny-Bot/config.GitCommit=${GitCommit}' \
    -X 'github.com/krau/SaveAny-Bot/config.BuildTime=${BuildTime}' \
    -X 'github.com/krau/SaveAny-Bot/config.Docker=true' \
    " \
    -o saveany-bot .

FROM alpine:latest

RUN apk add --no-cache curl ffmpeg yt-dlp

WORKDIR /app

COPY --from=builder /app/saveany-bot .
COPY entrypoint.sh .

RUN chmod +x /app/saveany-bot && \
    chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
