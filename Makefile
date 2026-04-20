.PHONY: migrate-up migrate-down

MIGRATIONS_DB ?= $(DATABASE_URL)

migrate-up:
	@if [ -z "$(MIGRATIONS_DB)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@DATABASE_URL="$(MIGRATIONS_DB)" bash scripts/migrate.sh up

migrate-down:
	@if [ -z "$(MIGRATIONS_DB)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@if [ -z "$(N)" ]; then echo "usage: make migrate-down N=1"; exit 2; fi
	@DATABASE_URL="$(MIGRATIONS_DB)" bash scripts/migrate.sh down "$(N)"

