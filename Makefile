MIGRATIONS_DIR := internal/database/migrations
GOOSE_ARGS := -dir $(MIGRATIONS_DIR)
GOOSE_ENV := GOOSE_DRIVER=sqlite3 GOOSE_DBSTRING=./data/yaps.db

.PHONY: generate
generate:
	sqlc generate
	templ generate

.PHONY: migrate/up
migrate/up:
	$(GOOSE_ENV) goose $(GOOSE_ARGS) up

.PHONY: migrate/down
migrate/down:
	$(GOOSE_ENV) goose $(GOOSE_ARGS) down

.PHONY: migrate/status
migrate/status:
	$(GOOSE_ENV) goose $(GOOSE_ARGS) status

.PHONY: migrate/version
migrate/version:
	$(GOOSE_ENV) goose $(GOOSE_ARGS) version

.PHONY: migrate/reset
migrate/reset:
	$(GOOSE_ENV) goose $(GOOSE_ARGS) reset

.PHONY: migrate/up-to # usage: make migrate/up-to V=3
migrate/up-to:
	@test -n "$(V)" || (echo "usage: make migrate/up-to V=<version>" && exit 1)
	$(GOOSE_ENV) goose $(GOOSE_ARGS) up-to $(V)

.PHONY: migrate/create
migrate/create:
	goose $(GOOSE_ARGS) create $(filter-out $@,$(MAKECMDGOALS)) sql

%:
	@if [ -z "$(filter migrate/create,$(MAKECMDGOALS))" ]; then \
		echo "make: *** No rule to make target '$@'.  Stop." >&2; exit 2; \
	fi