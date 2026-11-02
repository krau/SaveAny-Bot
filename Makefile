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

e2e-api:
	go test -tags=e2e ./e2e -count=1
