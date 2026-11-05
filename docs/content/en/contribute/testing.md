---
title: "Testing"
weight: 10
---

# Testing

The project ships a seeded local environment and two end-to-end suites that do
not need Telegram credentials.

## Seeded environment

```bash
make testenv
```

The command creates a disposable data directory and prints:

- the Web UI URL
- the admin password (`saveany-test` by default)
- a seeded local storage and whitelisted user
- a sample URL that can be used as a direct-links task

Use `--keep` to retain the generated data directory, or pass `--data-dir` to
reuse an existing one. Telegram is left stopped, so the web console and the
non-Telegram task pipeline are available immediately.

## Backend end-to-end tests

```bash
make e2e-api
```

The suite builds the binary, starts the seeded environment on an ephemeral
port and drives the public API with a real HTTP client. It covers authentication
and CSRF, onboarding, storage and user lifecycle, settings updates, the SPA
asset fallback, and a direct-links task that downloads from a local HTTP server
into the seeded storage.

The tests live under `e2e/` behind the `e2e` build tag, so `go test ./...` does
not start the environment.

## Telegram end-to-end tests

```bash
make e2e-telegram
```

This suite runs the real bot handlers, dispatcher, task queue, and storage
backends against an in-memory `tg.Invoker`. It covers `/start`, group admission,
saving a Telegram document to local storage (including progress edits), and
uploading it to a Telegram storage backend. No Telegram account or network is
used: gotd's client, sender, downloader, and uploader run unchanged against the
fake RPC layer.

## Web end-to-end tests

```bash
make e2e-web-setup   # once, downloads the Chromium build
make e2e-web
```

The suite starts the seeded environment and drives the built web UI with
Playwright. It covers login, storage testing and creation, and creating a
direct-links task that reaches the completed state. Test artifacts are written
to `tmp/e2e-web-results/`.

Playwright is a dev dependency of `web/ui` and its version is locked by
`pnpm-lock.yaml`. `make e2e-web-setup` installs the matching Chromium build.

## Combined

```bash
make e2e
```

Runs the API, Telegram, and browser suites after building the web UI. The fake
Telegram layer covers commands, downloads, uploads, and group admission without
an account. Real Telegram server semantics, userbot login, and live rate limits
still need a test account.
