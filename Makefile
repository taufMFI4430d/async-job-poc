SHELL := /bin/sh

COMPOSE ?= docker-compose
GO ?= go
NPM ?= npm

.PHONY: help up down build ps logs api-logs run test test-cover test-integration fmt fmt-check vet check compose-config migrate-up migrate-down migrate-status mysql redis lifecycle-check ui-install ui-dev ui-build ui-test ui-check \
	docker-up docker-down docker-logs docker-ps \
	test-integration test-integration-mysql test-integration-redis \
	queue-list queue-length

help:
	@echo "Available commands:"
	@echo "  make up                Build and start the complete Docker stack"
	@echo "  make down              Stop the stack without deleting data"
	@echo "  make build             Build the API and worker Docker images"
	@echo "  make ps                Show container and health status"
	@echo "  make logs              Follow API, worker, MySQL, and Redis logs"
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
	@echo "  make test-integration       Run all integration tests"
	@echo "  make test-integration-mysql Run MySQL integration tests"
	@echo "  make test-integration-redis Run Redis integration tests"
	@echo "  make queue-list             Show queued job IDs"
	@echo "  make queue-length           Show number of queued jobs"
	@echo "  make worker-logs            Follow worker logs only"
	@echo "  make run-worker             Run one worker locally"
	@echo "  make lifecycle-check        Verify success, retries, and terminal failure"
	@echo "  make ui-install             Install locked React dependencies"
	@echo "  make ui-dev                 Run the Vite development server"
	@echo "  make ui-build               Build production UI assets"
	@echo "  make ui-test                Run UI tests"
	@echo "  make ui-check               Test and build the UI"

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

build:
	$(COMPOSE) build api worker

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f api worker mysql redis

worker-logs:
	$(COMPOSE) logs -f worker

run-worker:
	@test -f .env || (echo ".env is missing; copy .env.example to .env first" && exit 1)
	@set -a; . ./.env; set +a; \
		MYSQL_HOST=127.0.0.1 \
		MYSQL_PORT="$${MYSQL_HOST_PORT:-3306}" \
		REDIS_HOST=127.0.0.1 \
		REDIS_PORT="$${REDIS_HOST_PORT:-6379}" \
		$(GO) run ./cmd/worker

api-logs:
	$(COMPOSE) logs -f api

run: ui-build
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

test-integration: test-integration-mysql test-integration-redis

test-integration-mysql:
	@test -f .env || (echo ".env file is required"; exit 1)
	@set -a; . ./.env; set +a; \
		MYSQL_HOST=127.0.0.1 \
		MYSQL_PORT="$${MYSQL_HOST_PORT:-3306}" \
		$(GO) test -tags=integration ./internal/adapters/mysql \
		-run TestJobRepositoryAgainstMySQL -v

test-integration-redis:
	@test -f .env || (echo ".env file is required"; exit 1)
	@set -a; . ./.env; set +a; \
		REDIS_HOST=127.0.0.1 \
		REDIS_PORT="$${REDIS_HOST_PORT:-6379}" \
		$(GO) test -tags=integration ./internal/adapters/redis \
		-run TestJobQueueAgainstRedis -v

queue-list:
	@test -f .env || (echo ".env file is required"; exit 1)
	@set -a; . ./.env; set +a; \
		$(COMPOSE) exec redis redis-cli \
		-n "$${REDIS_DB:-0}" \
		LRANGE "$${REDIS_QUEUE_NAME:-jobs:pending}" 0 -1

queue-length:
	@test -f .env || (echo ".env file is required"; exit 1)
	@set -a; . ./.env; set +a; \
		$(COMPOSE) exec redis redis-cli \
		-n "$${REDIS_DB:-0}" \
		LLEN "$${REDIS_QUEUE_NAME:-jobs:pending}"

lifecycle-check:
	sh scripts/verify-lifecycle.sh

ui-install:
	cd web && $(NPM) ci

ui-dev: ui-install
	cd web && $(NPM) run dev

ui-build: ui-install
	cd web && $(NPM) run build

ui-test: ui-install
	cd web && $(NPM) run test

ui-check: ui-install
	cd web && $(NPM) run check

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

check: fmt-check test ui-check vet compose-config

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
