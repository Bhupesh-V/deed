## Count Verification

Depending on deed's dataset, schema constraints and requested rows we need to validate "all counts" before doing any seeding.

1. Length limits of columns of types [char, varchar, text, int, numeric]. For e.g a 2 character limit on a char column limits possible combinations that can be inserted (52^2).
2. UNIQUE constraint.

## Seed Verification

1. We do this by building how many rows will be inserted if we see no failures, initialise `total=0`.
2. Get count of rows user wants to insert. Add that to out `total`.
3. Resolve dependencies, if user has set a hard-limt on count for certain tables, add that to our `total` otherwise add the count obtained from `tables affected` * `count per table`.
4. Generate & Insert mock data for all tables.
5. Grab counts of all rows across tables affected.
6. The count from step 5 should match the count from step 3.

## Constraint Fulfillment

### `UNIQUE`

- Initialise a deed only table inside the user's db, keep track of ids already referenced.
- Feistel Cipher: https://www.youtube.com/watch?v=FGhj3CGxl8I
- Multiplicative Congruential Generator (MCG)

### `CHECK`

Filters:

- Reject following clauses:
  - Enumeration Clause (constraint with `IN`).
  - Regex Clause
    -  `CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')`
    -  `CHECK (email ILIKE '%@%.%')`
    -  `CHECK (email SIMILAR TO '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}')`
    -  `CHECK (username !~ '\s')`
    -  `CHECK (email ~ '^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$')`
    -  `CHECK (regexp_like(email, '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$', 'i'))`

Process:

- AST parse the CEHCK clause.
- Try to satisfy the expression using Z3 https://github.com/mitchellh/go-z3.


## Debugging

1. Start a postgres container
   ```sh
   docker run --name deed_postgres_testing -e POSTGRES_PASSWORD=password -v deed_postgres:/var/lib/postgresql/data -p 5433:5433 -d postgres:14.9
   ```
2. Generate Mermaid diagram from postgres schema
   ```sh
   mermerd -c "postgres://postgres:my_secure_password@127.0.0.1:5433/postgres" -s public --outputFileName er.mmd
   ```
3. Convert Mermaid diagram to SVG
   ```sh
   docker run --rm -u `id -u`:`id -g` -v $PWD:/data minlag/mermaid-cli -i er.mmd
   ```


## Postgres

### Check all connections & their activity state

```sql
SELECT pid, state, query, wait_event_type, wait_event 
FROM pg_stat_activity 
WHERE backend_type = 'client backend';
```

```json
[
    {
        "pid": 93,
        "state": "idle",
        "query": "",
        "wait_event_type": "Client",
        "wait_event": "ClientRead"
    },
    {
        "pid": 94,
        "state": "idle",
        "query": "",
        "wait_event_type": "Client",
        "wait_event": "ClientRead"
    },
    {
        "pid": 95,
        "state": "idle",
        "query": "",
        "wait_event_type": "Client",
        "wait_event": "ClientRead"
    },
    {
        "pid": 96,
        "state": "idle",
        "query": "",
        "wait_event_type": "Client",
        "wait_event": "ClientRead"
    },
    {
        "pid": 97,
        "state": "active",
        "query": "copy \"audit_logs\" ( \"action_type\", \"entity_name\", \"entity_id\", \"ip_address\", \"created_at\" ) from stdin binary;",
        "wait_event_type": "Client",
        "wait_event": "ClientRead"
    },
    {
        "pid": 98,
        "state": "idle",
        "query": "",
        "wait_event_type": "Client",
        "wait_event": "ClientRead"
    },
    {
        "pid": 99,
        "state": "active",
        "query": "\nSELECT pid, state, query, wait_event_type, wait_event \nFROM pg_stat_activity \nWHERE backend_type = 'client backend';",
        "wait_event_type": null,
        "wait_event": null
    }
]
```

### Find all non-constraint Indexes

Consider dropping these indexes before ingestion

```sql
SELECT 
    i.schemaname,
    i.tablename,
    i.indexname,
    idx.indisunique,
    CASE 
        WHEN idx.indisprimary THEN 'Primary Key (Preserved)'
        WHEN con.contype = 'u' THEN 'Unique Constraint (Preserved)'
        WHEN con.contype = 'x' THEN 'Exclusion Constraint (Preserved)'
        ELSE 'Standalone Index (Should be dropped)'
    END AS reason_preserved
FROM pg_indexes i
JOIN pg_class c ON c.relname = i.indexname
JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = i.schemaname
JOIN pg_index idx ON idx.indexrelid = c.oid
LEFT JOIN pg_constraint con ON con.conindid = idx.indexrelid
WHERE i.schemaname = current_schema();
```


deed config init --f file.json
- initialise the deed config, talk to the database, create the structure, add any sensible defaults.

deed config clean --f file.json
- remove properties with no values. Useful for making the config easier to digest and share.