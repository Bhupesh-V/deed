# Output configuration
Output deed.gif
Output deed.mp4

# Terminal Resolution & Visual Settings
Set FontSize 16
Set Width 1600
Set Height 900
Set Padding 20
Set Theme "OneDark"
Set TypingSpeed 40ms

# -------------------------------------------------------------------
# Setup hidden environment variables for clean command typing
# -------------------------------------------------------------------
Hide
Type "export DB_URL='postgres://postgres:my_secure_password@127.0.0.1:5433/deed-commerce?sslmode=disable'"
Enter
Type "clear"
Enter
Show

# -------------------------------------------------------------------
# 1. Start Postgres container (db-up target)
# -------------------------------------------------------------------
Set TypingSpeed 1ms
Type "docker run --name deed_postgres --shm-size=2g -e POSTGRES_PASSWORD=my_secure_password -e POSTGRES_DB=deed-commerce -v deed-postgres-v1:/var/lib/postgresql -p 5433:5432 -d postgres:18"
Enter
Sleep 2s

# -------------------------------------------------------------------
# 2. Run pending database migrations (migrate-up target)
# -------------------------------------------------------------------
Type 'migrate -path tests/postgres/migrations -database "$DB_URL" up'
Enter
Sleep 3s


Set TypingSpeed 1ms
Type "clear"
Enter

# -------------------------------------------------------------------
# 3. Execute data seeding (run target)
# -------------------------------------------------------------------
Set TypingSpeed 40ms
Type 'deed seed --dsn "$DB_URL" --tables=proof_verifications,system_event_logs --count=1000000 --config=tests/postgres/deed.json'
Enter

# Allow sufficient time for tree UI rendering & MPB progress bars
Sleep 25s