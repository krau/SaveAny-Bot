.PHONY: *

run:
	-make down
	docker compose -f docker-compose.local.yml up --build

down:
	docker compose -f docker-compose.local.yml down --remove-orphans
