<div align="center">

# 🌱 deed

`Effortless targeted database seedeing for query optimization & performance testing`

[Key Features](#why) • [Ideal Workflow](#ideal-workflow) • [Configuration](#deed-config) • [Performance](#performance)

</div>

---

## Why?

* **Optimizing at Scale:** Your app is growing and your database is pulling heavy weight. You need realistic volume to continuously benchmark and refine your **SQL queries** and **indexes**.
* **Zero Prod Data Access:** You don't have access to production databases due to strict compliance, privacy, or security regulations.
* **Realistic Load Testing:** You need a quick, painless way to generate massive datasets locally without spending days writing custom seed scripts.

## Why not?

* You already have unrestricted access to a fully populated, compliant production database dump.
* Your workflow relies entirely on strict double-validation of business logic at the UI/API layer for test data.


## 📦 Installation

```bash
# TODO: Not ready for a stable release
```

## 🗄️ Database Coverage

### PostgreSQL

#### **Column Types**

| Category | Status | Supported Types |
| :--- | :---: | :--- |
| **Numeric** | ✅ | `int2`, `int4`, `int8`, `numeric`, `decimal`, `float4`, `float8`, `serial`, `bigserial` |
| **Character** | ✅ | `varchar`, `char`, `text`, `bpchar` |
| **Boolean** | ✅ | `boolean` |
| **UUID** | ✅ | `uuid` |
| **Arrays** | ✅ | `text[]`, `int[]`, etc. |
| **Network** | ✅ | `inet`, `cidr`, `macaddr`, `macaddr8` |
| **Enum** | ✅ | Custom `ENUM` types |
| **Date & Time** | ⏳ | `date`, `time`, `timetz`, `timestamp`, `timestamptz`, `interval` *(Planned)* |
| **JSON** | ⏳ | `json`, `jsonb` *(Planned)* |
| **Binary** | ⏳ | `bytea` *(Planned)* |
| **Ranges** | ⏳ | `int4range`, `numrange`, `tsrange`, `tstzrange`, `daterange` *(Planned)* |
| **Full-Text** | ⏳ | `tsvector`, `tsquery` *(Planned)* |
| **Geometric** | ⏳ | `point`, `line`, `lseg`, `box`, `path`, `polygon`, `circle` *(Planned)* |
| **Bit Strings** | ⏳ | `bit`, `bit varying` *(Planned)* |
| **Composite** | ⏳ | User-defined composite types *(Planned)* |

#### **Constraints & Modifiers**

| Constraint / Feature | Status | Notes / Sub-features |
| :--- | :---: | :--- |
| **Primary Key** | ✅ | Supported |
| **Not Null** | ✅ | Supported |
| **Unique** | ✅ | Supported |
| **Default Expressions** | ✅ | Supported |
| **Generated Columns** | ✅ | `GENERATED AS IDENTITY`, `GENERATED ALWAYS AS (...) STORED` |
| **Simple Foreign Key** | ✅ | Resolved across parent dependencies |
| **Composite Foreign Key** | ⏳ | *(Planned)* |
| **Self-Referencing FK** | ⏳ | *(Planned)* |
| **Check Constraints** | ⏳ | *(Planned)* |
| **Exclusion Constraints** | ⏳ | *(Planned)* |
| **Timing Modifiers** | ⏳ | `DEFERRABLE`, `INITIALLY DEFERRED / IMMEDIATE` *(Planned)* |


## 🚀 Usage

### Ideal Workflow

> [!NOTE]
> **Targeted Ingestion:** Because SQL queries and index strategies change alongside your UI/UX features, **`deed` is built for targeted iterations**—not for dumping generic data into your entire schema once and forgetting about it.

1. **Spin up a clean DB container** with your latest migrations applied.
2. **Identify target tables** involved in the specific query or feature you are optimizing.
3. **Seed mock data** into those specific tables (and their required parent dependencies) using `deed`.
4. **Benchmark & rewrite** your SQL queries and indexes against realistic data scales.
5. **Destroy the container** and repeat for the next iteration.

### Deed Config

To customize how mock data is generated, create a `deed.json` file in your project directory:

```jsonc
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
        // Row count here overrides the CLI flag for this table
        "count": 200,
        "columns": {
          "username": {
            "type": "regex",
            // Define business logic requirements using RE2-compatible regex
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
```

> [!TIP]
> Commit your primary `deed.json` to Git so your team shares base generator rules. When working locally, copy it over, adjust row counts/patterns as needed, and feed your local config to `deed`.

### Seeding Data

Ingest 1,000,000 rows into the `app` and `users` tables (along with any required foreign-key parent dependencies):

```bash
deed seed \
  --dsn "postgres://postgres:my_secure_password@127.0.0.1:5433/postgres" \
  --tables=app,users \
  --count=1000000 \
  --config=deed.json
```


## Performance

Ingestion throughput is impacted by four core variables:
1. **Dependency depth** (number of parent tables requiring resolved FK relationships).
2. **Column complexity** (mix of standard scalars vs. regex/custom types).
3. **Total column count** per table.
4. **Database host system specifications**.

### Summary from test runs

| Target Dataset | Tables Ingested | Total Rows | Execution Time |
| :--- | :---: | :---: | :---: |
| **Complex Relational Tree** | 10 tables | **5.00M+** | **16.27s** |
| **Audit Logs (Single Table)** | 1 table | **20.00M** | **60.08s** |
| **Massive Flat Ingestion** | 1 table | **67.00M** | **~1ms** |

<details>
<summary><b>View Test Run Output</b></summary>

#### Test 1: 1M rows across 5 root tables (10 total resolved dependencies)

```bash
Dependencies for 'proof_verifications'
 
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
                                    
Dependencies for 'system_event_logs'                    
                                    
🔗 system_event_logs
                        
Seeding Data (10 tables)
                                  
 countries                     50 / 50      [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       500000 rows/s  ✔ Done
 users                        200 / 200     [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %        2.00M rows/s  ✔ Done
 orders                       300 / 300     [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %        3.00M rows/s  ✔ Done
 system_event_logs        1000000 / 1000000 [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       422353 rows/s  ✔ Done
 shipping_carriers        1000000 / 1000000 [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       306030 rows/s  ✔ Done
 user_addresses           1000000 / 1000000 [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       248877 rows/s  ✔ Done
 shipments                    100 / 100     [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %        1.00M rows/s  ✔ Done
 shipment_tracking_events 1000000 / 1000000 [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       392016 rows/s  ✔ Done
 delivery_proofs          1000000 / 1000000 [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       190499 rows/s  ✔ Done
 proof_verifications          100 / 100     [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %        1.00M rows/s  ✔ Done
                                                           
✨ Ingestion complete across all tables. Took 16.27 seconds
                                                           
```

#### Test 2: 20 Million rows in 1 table

```bash
                                    
Dependencies for 'system_event_logs'
                        
🔗 system_event_logs

Seeding Data (1 tables)
                               
 system_event_logs        20000000 / 20000000 [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       357122 rows/s  ✔ Done
                                                           
✨ Ingestion complete across all tables. Took 60.08 seconds

```

#### Test 3: 67 Million rows in 1 table

```bash
Dependencies for 'warehouse_shelf_grid'
                                       
🔗 warehouse_shelf_grid
                       
Seeding Data (1 tables)
                       
 warehouse_shelf_grid     67000000 / 67000000 [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 100 %       880227 rows/s  ✔ Done
                                                           
✨ Ingestion complete across all tables. Took 99.01 seconds

```

</details>
