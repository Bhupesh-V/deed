# deed

## Why?

- Your database is a heavy lifter, i.e you have to constantly optimise your **SQL queries** and **indexes**.
- You don't have access to production data due to compliance issues.
- You are a master procastinator on load testing your APIs.

## Why not?

- You don't know how to use a modern RDBMS.
- Strict or double validation of business data on API/UI level.

## For whom?

1. Database Engineers.
2. Data Engineers, Backend Engineers.

## Installation

TODO

## Databases Coverage

### Postgres


| Constraint Type | Description | Common SQL Syntax Example | Tool Coverage |
| :--- | :--- | :--- | :---: |
| **Primary Key** | Uniquely identifies each row in a table. Implicitly enforces `NOT NULL` and `UNIQUE`. | `PRIMARY KEY (id)` | [ ] Supported |
| **Foreign Key** | Links data in one table to a primary/unique key in another, maintaining relationship consistency. | `FOREIGN KEY (user_id) REFERENCES users(id)` | [ ] Supported |
| **Unique** | Ensures all values in a column or set of columns are distinct (typically allows `NULL`s depending on SQL engine). | `UNIQUE (email)` | [ ] Supported |
| **Not Null** | Prevents `NULL` (missing) values from being stored in a column. | `email VARCHAR(255) NOT NULL` | [ ] Supported |
| **Check** | Evaluates a Boolean expression against inserted or updated row data. | `CHECK (age >= 18)` | [ ] Supported |
| **Default** | Automatically assigns a preset value if no explicit value is supplied during `INSERT`. | `created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP` | [ ] Supported |
| **Exclusion Constraint** | Guarantees that if two rows are compared on specified columns using specific operators, at least one returns false (e.g., preventing overlapping time intervals in PostgreSQL). | `EXCLUDE USING gist (room_id WITH =, reservation_period WITH &&)` | [ ] Supported |
| **Domain Constraint** | Restricts data values to a named domain with custom constraints/types. | `CREATE DOMAIN pos_int AS INT CHECK (VALUE > 0);` | [ ] Supported |
| **Generated/Computed Column** | Constrains a column value to always be derived from a specific expression or formula. | `total DECIMAL(10,2) GENERATED ALWAYS AS (price * qty)` | [ ] Supported |

## Usage

### Ideology

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


The **deed config can be committed to a git repository** and shared with team members. However, its higly recommended that you copy your team config, tune any parameters and supply the copy to deed, this aligns with the deed [ideology](#ideology).

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

Reference run from `make test`

```
✅ Inserted 50 rows into countries
✅ Inserted 200 rows into users
✅ Inserted 300 rows into orders
✅ Inserted 20000000 rows into shipping_carriers
✅ Inserted 20000000 rows into user_addresses
✅ Inserted 100 rows into shipments
✅ Inserted 20000000 rows into shipment_tracking_events
```

- For 3 tables with 20 million rows it took deed almost 6 min 20 seconds to complete ingestion, leading to **~2 minutes per 20M rows**.