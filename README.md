# deed

## Why?

- You are scaling and your database is a heavy lifter i.e, you have to constantly optimise your **SQL queries** and **Indexes**.
- You don't have access to production data due to compliance issues.
- You are a master procastinator on load testing your APIs.

## Why not?

- You have un-restricted access to the production database.
- You have enabled strict (or double) validation of business data on API/UI level.

## For whom?

1. Database Engineers.
2. Data Engineers, Backend Engineers.

## Installation

TODO

## Databases Coverage

### Postgres

<!-- 
| Constraint Type | Description | Common SQL Syntax Example | Tool Coverage |
| :--- | :--- | :--- | :---: |
| **Primary Key** | Uniquely identifies each row in a table. Implicitly enforces `NOT NULL` and `UNIQUE`. | `PRIMARY KEY (id)` | [x] Supported |
| **Foreign Key** | Links data in one table to a primary/unique key in another, maintaining relationship consistency. | `FOREIGN KEY (user_id) REFERENCES users(id)` | [x] Supported |
| **Unique** | Ensures all values in a column or set of columns are distinct (typically allows `NULL`s depending on SQL engine). | `UNIQUE (email)` | [x] Supported |
| **Not Null** | Prevents `NULL` (missing) values from being stored in a column. | `email VARCHAR(255) NOT NULL` | [ ] Supported |
| **Check** | Evaluates a Boolean expression against inserted or updated row data. | `CHECK (age >= 18)` | [ ] Supported |
| **Default** | Automatically assigns a preset value if no explicit value is supplied during `INSERT`. | `created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP` | [ ] Supported |
| **Exclusion Constraint** | Guarantees that if two rows are compared on specified columns using specific operators, at least one returns false (e.g., preventing overlapping time intervals in PostgreSQL). | `EXCLUDE USING gist (room_id WITH =, reservation_period WITH &&)` | [ ] Supported |
| **Domain Constraint** | Restricts data values to a named domain with custom constraints/types. | `CREATE DOMAIN pos_int AS INT CHECK (VALUE > 0);` | [ ] Supported | -->

## Usage

### Ideal Workflow

Since the choice of how you write SQL queries and indexes completely depends on how your UI/UX looks like (which changes often), **each deed run is supposed to be targetted, meaning you don't ingest data in your whole schema in one go and never look back.**

Instead, your ideal workflow should look like,

1. Start a container with your up-to-date schema applied. Run any and all DDL/DML migrations.
2. Have list of tables you want to generate data for, these are the tables that you have to optimise your queries/indexes for.
3. Seed the data using deed.
4. Rewrite and test your queries/indexes.
5. Destroy the db container.

Deed can definitely be used to fill up your schema with test data and by extension be used to demo your apps, however that was not my original intent while building it.

### Deed Config

deed works well if you already have a config setup as per your use-case. Create a `deed.json` file wherever you plan to invoke the deed CLI with the following content.

```json
{
    "version": "1",
    "database": {
        "name": "postgres"
    },
    "rules": {
        "ignore_tables": [
            "schema_migrations"
        ],
        "tables": {
            "users": {
                // strict limit on no.of rows that should be ingested in users, this takes precedence over CLI
                "count": 200,
                "columns": {
                    "username": {
                        "type": "regex",
                        // define how your business data looks like using RE2 compatible regex expressions
                        "pattern": "^[a-zA-Z0-9_-]{3,30}$"
                    },
                    "password_hash": {
                        "type": "regex",
                        "pattern": "^\\$2[ayb]\\$[0-9]{2}\\$[A-Za-z0-9./]{53}$"
                    }
                }
            },
            "countries": {
                "count": 50
            }
        }
    }
}
...
```

The config is pretty intutive:


The **deed config can be committed to a git repository** and shared with your team. However, its higly recommended that you copy your team config first before using deed, tune any parameters and supply the copy to deed, this aligns well with deed's [workflow](#ideal-workflow).

### Seed data

Ingest a million rows in tables `app` and `users` along with their parents (aka dependencies).

```
deed seed \
 --dsn "postgres://postgres:my_secure_password@127.0.0.1:5433/postgres" \
 --tables=app,users \
 --count=1000000 \
 --config=deed.json
```

Sample Output

```
```

## Performance

Performance for deed depends on following factors.

1. No.of dependent tables in the schema. 
2. Variation of column data types.
3. No.of columns

Having said that here are some reference runs:

### Inserting `1M` rows in 4 tables

```
--- Dependencies for 'proof_verifications' ---

🔗 proof_verifications
╰── 🔗 delivery_proofs
    ╰── 🔗 shipment_tracking_events
        ╰── 🔗 shipments
            ├── 🔗 orders
            │   ╰── 🔗 users
            ├── 🔗 shipping_carriers
            ╰── 🔗 user_addresses
                ├── 🔗 users
                ╰── 🔗 countries

--- Starting Ingestion (9 tables) ---

✅ Inserted 50 rows into countries
✅ Inserted 200 rows into users
✅ Inserted 300 rows into orders
✅ Inserted 1000000 rows into shipping_carriers
✅ Inserted 1000000 rows into user_addresses
✅ Inserted 100 rows into shipments
✅ Inserted 1000000 rows into shipment_tracking_events
✅ Inserted 1000000 rows into delivery_proofs
✅ Inserted 100 rows into proof_verifications

real    0m35.714s
user    2m20.426s
sys     0m12.127s
```


### Inserting `20M` rows in 1 table

```
--- Dependencies for 'audit_logs' ---

🔗 audit_logs

--- Starting Ingestion (1 tables) ---

✅ Inserted 20000000 rows into audit_logs

real    0m33.542s
user    0m29.421s
sys     0m1.465s
```

### Inserting `67M` rows in 1 table

```
--- Dependencies for 'audit_logs' ---

🔗 audit_logs

--- Starting Ingestion (1 tables) ---

✅ Inserted 67000000 rows into audit_logs

real    2m0.645s
user    1m40.859s
sys     0m4.647s
```