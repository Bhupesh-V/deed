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

## Database Coverage

### Postgres

<!-- * **Column Types:**
  * [x] Numeric *(int, int4, int8, decimal, float, real)*
  * [x] Character *(varchar, char, text, bpchar)*
  * [ ] JSONB
  * [x] ENUM
  * [x] Time
  * [x] Arrays
* **Constraints:**
  * [x] UNIQUE
  * [ ] Foreign Key
    * [x] Simple
    * [ ] Composite
    * [ ] Self-Referencing
  * [ ] CHECK
  * [ ] EXCLUSION -->


* **Column Types:**
  * [x] Numeric *(int2/smallint, int4/integer, int8/bigint, numeric/decimal, float4/real, float8/double precision, serial/bigserial)*
  * [x] Character *(varchar, char, text, bpchar)*
  * [x] Boolean *(boolean)*
  * [x] UUID *(uuid)*
  * [ ] Date & Time *(date, time, timetz, timestamp, timestamptz, interval)*
  * [ ] JSON *(json, jsonb)*
  * [ ] Binary *(bytea)*
  * [x] Arrays *(e.g., text[], int[])*
  * [x] Network Address *(inet, cidr, macaddr, macaddr8)*
  * [ ] Ranges & Multiranges *(int4range, numrange, tsrange, tstzrange, daterange)*
  * [ ] Full-Text Search *(tsvector, tsquery)*
  * [ ] Geometric *(point, line, lseg, box, path, polygon, circle)*
  * [ ] Bit Strings *(bit, bit varying)*
  * [ ] Custom / User-Defined
    * [x] ENUM
    * [ ] Composite

* **Constraints:**
  * [x] PRIMARY KEY
  * [x] NOT NULL
  * [x] UNIQUE
  * [ ] CHECK
  * [ ] EXCLUSION
  * [ ] Foreign Key
    * [x] Simple
    * [ ] Composite
    * [ ] Self-Referencing
  * [x] DEFAULT Expressions
  * [x] Identity & Generated Columns
    * [x] GENERATED AS IDENTITY
    * [x] GENERATED ALWAYS AS (...) STORED
  * [ ] Timing Modifiers *(DEFERRABLE, INITIALLY DEFERRED / IMMEDIATE)*

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

```jsonl
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

## Performance

Ingestion speed for deed depends on following factors.

1. No.of dependent tables in the schema. 
2. Variation of column data types.
3. No.of columns.
4. Database host resources.

Having said that here are some reference runs:

### Inserting `1M` rows across 4 tables

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

real    0m16.064s
user    0m42.597s
sys     0m1.887s
```


### Inserting `20M` rows in 1 table

```
--- Dependencies for 'audit_logs' ---

🔗 audit_logs

--- Starting Ingestion (1 tables) ---

✅ Inserted 20000000 rows into audit_logs

real    0m40.765s
user    0m30.115s
sys     0m1.537s
```

### Inserting `67M` rows in 1 table

```
--- Dependencies for 'audit_logs' ---

🔗 audit_logs

--- Starting Ingestion (1 tables) ---

✅ Inserted 67000000 rows into audit_logs

real    2m13.385s
user    1m43.081s
sys     0m5.010s
```