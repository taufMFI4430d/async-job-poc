SHELL := /bin/sh

COMPOSE ?= docker-compose
GO ?= go

.PHONY: help up down build ps logs api-logs run test test-cover test-integration fmt fmt-check vet check compose-config migrate-up migrate-down migrate-status mysql redis

help:
	@echo "Available commands:"
	@echo "  make up                Build and start the complete Docker stack"
	@echo "  make down              Stop the stack without deleting data"
	@echo "  make build             Build the API Docker image"
	@echo "  make ps                Show container and health status"
	@echo "  make logs              Follow API, MySQL, and Redis logs"
	@echo "  make api-logs          Follow API logs only"
	@echo "  make run               Run the API locally against Docker services"
	@echo "  make test              Run unit tests"
	@echo "  make test-cover        Run unit tests with a coverage report"
	@echo "  make test-integration  Run the real MySQL repository test"
	@echo "  make fmt               Format Go source files"
	@echo "  make fmt-check         Check whether Go source is formatted"
	@echo "  make vet               Run Go static analysis"
	@echo "  make check             Run format, test, vet, and Compose checks"
	@echo "  make migrate-up        Apply all pending database migrations"
	@echo "  make migrate-down      Roll back one database migration"
	@echo "  make migrate-status    Show the current migration version"
	@echo "  make mysql             Open a MySQL shell"
	@echo "  make redis             Open a Redis CLI"

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

build:
	$(COMPOSE) build api

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f api mysql redis

api-logs:
	$(COMPOSE) logs -f api

run:
	@test -f .env || (echo ".env is missing; copy .env.example to .env first" && exit 1)
	@set -a; . ./.env; set +a; \
		MYSQL_HOST=127.0.0.1 \
		MYSQL_PORT="$${MYSQL_HOST_PORT:-3306}" \
		REDIS_HOST=127.0.0.1 \
		REDIS_PORT="$${REDIS_HOST_PORT:-6379}" \
		$(GO) run ./cmd/api

test:
	$(GO) test ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

test-integration:
	@test -f .env || (echo ".env is missing; copy .env.example to .env first" && exit 1)
	@set -a; . ./.env; set +a; \
		MYSQL_HOST=127.0.0.1 \
		MYSQL_PORT="$${MYSQL_HOST_PORT:-3306}" \
		$(GO) test -tags=integration ./internal/adapters/mysql -run TestJobRepositoryAgainstMySQL -v

fmt:
	gofmt -w cmd internal

fmt-check:
	@files="$$(gofmt -l cmd internal)"; \
		if [ -n "$$files" ]; then \
			echo "The following Go files need formatting:"; \
			echo "$$files"; \
			exit 1; \
		fi

vet:
	$(GO) vet ./...

compose-config:
	$(COMPOSE) config --quiet

check: fmt-check test vet compose-config

migrate-up:
	$(COMPOSE) run --rm migrate

migrate-down:
	$(COMPOSE) run --rm --entrypoint /bin/sh migrate -c 'migrate -path=/migrations -database="mysql://$${MYSQL_USER}:$${MYSQL_PASSWORD}@tcp(mysql:3306)/$${MYSQL_DATABASE}?multiStatements=true" down 1'

migrate-status:
	$(COMPOSE) run --rm --entrypoint /bin/sh migrate -c 'migrate -path=/migrations -database="mysql://$${MYSQL_USER}:$${MYSQL_PASSWORD}@tcp(mysql:3306)/$${MYSQL_DATABASE}?multiStatements=true" version'

mysql:
	$(COMPOSE) exec mysql sh -c 'mysql -u"$$MYSQL_USER" -p"$$MYSQL_PASSWORD" "$$MYSQL_DATABASE"'

redis:
	$(COMPOSE) exec redis redis-cli
