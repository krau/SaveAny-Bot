.PHONY: *

run:
	-make down
	docker compose -f docker-compose.local.yml up --build

down:
	docker compose -f docker-compose.local.yml down --remove-orphans

web:
	cd web/ui && pnpm install --frozen-lockfile && pnpm run build && touch ../dist/.gitkeep

build: web
	CGO_ENABLED=0 go build -trimpath -o saveany-bot .

testenv:
	go run . testenv

e2e: e2e-api e2e-telegram e2e-web

e2e-api:
	go test -tags=e2e ./e2e -count=1

e2e-telegram:
	go test -tags=e2e ./e2e/telegram -count=1

e2e-web-setup:
	pnpm --dir web/ui exec playwright install chromium

e2e-web: web
	mkdir -p tmp
	go build -o tmp/saveany-bot-e2e .
	pnpm --dir web/ui exec playwright test
