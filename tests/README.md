# tests

The tests folder contains different sample schema datasets that can be applied and be used for deed's functional testing.

## postgres

1. Start a Postgres 18 container
   ```
   docker run --name deed_postgres \
     -e POSTGRES_PASSWORD=my_secure_password \
     -e POSTGRES_DB=deed-commerce \
     -v deed-postgres-v1:/var/lib/postgresql \
     -p 5433:5432 \
     -d postgres:18
   ```
   DSN
   ```
   postgres://postgres:my_secure_password@127.0.0.1:5433/deed-commerce
   ```
2. Apply the migration(s) using golang-migrate.
   ```
   migrate -path tests/schemas/postgres -database "postgres://postgres:my_secure_password@127.0.0.1:5433/deed-commerce?sslmode=disable" up
   ```