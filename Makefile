# Variables
DB_CONTAINER_NAME = deed_postgres
DB_USER = postgres
DB_PASSWORD = my_secure_password
DB_NAME = deed-commerce
DB_PORT = 5433
DB_VOLUME = deed-postgres-v1
MIGRATIONS_PATH = tests/postgres/migrations
DB_URL = "postgres://$(DB_USER):$(DB_PASSWORD)@127.0.0.1:$(DB_PORT)/$(DB_NAME)?sslmode=disable"

.PHONY: env clean db-up db-down db-logs migrate-up migrate-down migrate-version migrate-force help

## help: Show available commands
help:
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  env             Start container, wait for DB, and apply migrations (ALL-IN-ONE)"
	@echo "  clean           Tear down db environment AND wipe volume data"
	@echo "  db-up           Start the PostgreSQL container only"
	@echo "  db-down         Stop and remove the container"
	@echo "  db-logs         View container logs"
	@echo "  migrate-up      Apply all 'up' migrations"
	@echo "  migrate-down    Roll back 1 migration (usage: make migrate-down N=1)"


setup-mermaid: ## Install mermaid-cli via docker to generate ER diagrams
	docker pull minlag/mermaid-cli:latest

setup-mermerd:
	go install github.com/KarnerTh/mermerd@latest

## Install all the build and lint dependencies
setup: setup-mermaid setup-mermaid

## env: One-step command to start database and run migrations
env: db-up
	@echo "Waiting for PostgreSQL to be ready on port $(DB_PORT)..."
	@until docker exec $(DB_CONTAINER_NAME) pg_isready -U $(DB_USER) -d $(DB_NAME) > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "PostgreSQL is ready! Running migrations..."
	@$(MAKE) migrate-up
	@echo "Environment setup complete!"

## clean: Complete tear down (stops container & deletes volume)
clean: db-down
	-docker volume rm $(DB_VOLUME)

## db-up: Start PostgreSQL 18 container
db-up:
	@if [ $$(docker ps -aq -f name=^$(DB_CONTAINER_NAME)$$) ]; then \
		echo "Container $(DB_CONTAINER_NAME) already exists. Starting it..."; \
		docker start $(DB_CONTAINER_NAME) > /dev/null; \
	else \
		docker run --name $(DB_CONTAINER_NAME) \
			-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
			-e POSTGRES_DB=$(DB_NAME) \
			-v $(DB_VOLUME):/var/lib/postgresql \
			-p $(DB_PORT):5432 \
			-d postgres:18 > /dev/null; \
	fi

## db-down: Stop and remove the database container
db-down:
	-docker stop $(DB_CONTAINER_NAME)
	-docker rm $(DB_CONTAINER_NAME)

## db-logs: Stream database container logs
db-logs:
	docker logs -f $(DB_CONTAINER_NAME)

## migrate-up: Run all pending migrations
migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) up

## migrate-down: Roll back the last migration
N ?= 1
migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) down $(N)

## migrate-version: Check current database migration version
migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) version

## migrate-force: Force set version (usage: make migrate-force V=1)
migrate-force:
	@if [ -z "$(V)" ]; then echo "Error: V is required. Example: make migrate-force V=1"; exit 1; fi
	migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) force $(V)

## test: Run the seed command with parameters matching the local DB setup
test:
	go run main.go seed \
		--dsn $(DB_URL) \
		--tables=delivery_proofs \
		--count=1000000 \
		--config=tests/postgres/deed.json