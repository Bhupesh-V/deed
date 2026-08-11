# Variables
DB_CONTAINER_NAME := "deed_postgres"
DB_USER           := "postgres"
DB_PASSWORD       := "my_secure_password"
DB_NAME           := "deed-commerce"
DB_PORT           := "5433"
DB_VOLUME         := "deed-postgres-v1"
PG_CONFIG_PATH    := "tests/postgres"
MIGRATIONS_PATH   := "tests/postgres/migrations"
DB_URL            := "postgres://" + DB_USER + ":" + DB_PASSWORD + "@127.0.0.1:" + DB_PORT + "/" + DB_NAME + "?sslmode=disable"
COVERAGE_FILE     := "coverage.out"
COVERAGE_HTML     := "coverage.html"

# Default behavior: Show help screen
default:
    just --list

# Install mermaid-cli via docker to generate ER diagrams
setup-mermaid:
    docker pull minlag/mermaid-cli:latest

# Install mermerd tool via Go
setup-mermerd:
    go install github.com/KarnerTh/mermerd@latest

# Install all the build and lint dependencies
setup: setup-mermaid setup-mermerd

# Start container, wait for DB, and apply migrations (ALL-IN-ONE)
env: db-up
    @echo "Waiting for PostgreSQL to be ready on port {{DB_PORT}}..."
    @until docker exec {{DB_CONTAINER_NAME}} pg_isready -U {{DB_USER}} -d {{DB_NAME}} > /dev/null 2>&1; do \
        sleep 1; \
    done
    @echo "PostgreSQL is ready! Running migrations..."
    just migrate-up
    @echo "Environment setup complete!"

# Complete tear down (stops container & deletes volume)
clean: db-down
    -docker volume rm {{DB_VOLUME}}

# Start the PostgreSQL container only
db-up:
    @if [ $(docker ps -aq -f name=^{{DB_CONTAINER_NAME}}$) ]; then \
        echo "Container {{DB_CONTAINER_NAME}} already exists. Starting it..."; \
        docker start {{DB_CONTAINER_NAME}} > /dev/null; \
    else \
        docker run --name {{DB_CONTAINER_NAME}} \
            --shm-size=2g \
            -e POSTGRES_PASSWORD={{DB_PASSWORD}} \
            -e POSTGRES_DB={{DB_NAME}} \
            -v {{DB_VOLUME}}:/var/lib/postgresql \
            -p {{DB_PORT}}:5432 \
            -d postgres:18 \
            -c shared_buffers=2GB \
            -c max_wal_size=16GB \
            -c checkpoint_timeout=15min \
            -c wal_buffers=64MB \
            -c maintenance_work_mem=512MB \
            -c synchronous_commit=off \
            -c max_connections=200 > /dev/null; \
    fi

# Stop and remove the database container
db-down:
    -docker stop {{DB_CONTAINER_NAME}}
    -docker rm {{DB_CONTAINER_NAME}}

# Stream database container logs
db-logs:
    docker logs -f {{DB_CONTAINER_NAME}}

# Run all pending migrations
migrate-up:
    migrate -path {{MIGRATIONS_PATH}} -database "{{DB_URL}}" up

# Roll back migrations (usage: just migrate-down 1)
migrate-down n="1":
    migrate -path {{MIGRATIONS_PATH}} -database "{{DB_URL}}" down {{n}}

# Check current database migration version
migrate-version:
    migrate -path {{MIGRATIONS_PATH}} -database "{{DB_URL}}" version

# Force set version (usage: just migrate-force 1)
migrate-force v:
    migrate -path {{MIGRATIONS_PATH}} -database "{{DB_URL}}" force {{v}}

# Build deed binary
build:
    go build -o deed

# Recreates test db & builds deed binary
build-env: clean env build

# Ingest 4 tables with 1 million rows (use this for basic sanity)
run: build-env
    time ./deed seed \
        --dsn "{{DB_URL}}" \
        --tables=proof_verifications \
        --count=1000000 \
        --config=tests/postgres/deed.json

# Ingest one table with 20M rows
load: build-env
    time ./deed seed \
        --dsn "{{DB_URL}}" \
        --tables=audit_logs \
        --count=20000000 \
        --config=tests/postgres/deed.json

# Run unit-tests suite
test:
    go test -race ./... -coverpkg=./... -coverprofile={{COVERAGE_FILE}}

# Generate unit-tests coverage report
coverage:
    go tool cover -html {{COVERAGE_FILE}} -o {{COVERAGE_HTML}}