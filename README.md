# deed

## Why?

- Your database is a heavy lifter, i.e you now have to opimise your **SQL queries** and **indexes** (congrats on winning in capitalism).
- You don't have access to production data due to compliance issues.
- You are a master procastinator on load testing your APIs.

<!-- Primary personas (in order):

1. Database Engineers.
2. Data Engineers, Backend Engineers
3. DevOps & SRE folks. -->

## Why not?

- You don't know how to use a modern RDBMS.
<!-- - If I spot any issues on recreating postgres functionality from scratch, I'll put you in my hitlist. -->

## Supported databases

- Postgres

## Usage

### Deed Config

deed works well if you already have a config setup as per your use-case.

Create a `deed.yaml` file wherever you plan to invoke the deed CLI with the following content.

```yaml
...
```

The config is pretty intutive:

The deed config can be committed to a git repository and shared with team members.

### Seed data

Ingest a million rows in tables `app` and `users` along with their parents.

```
deed seed \
 --url "postgres://postgres:my_secure_password@127.0.0.1:5433/postgres" \
 --tables=app,users \
 --count=1000000 \
 --config=deed.yaml
```


### Dry run

```
deed seed --url "..." --tables=app --count=1000000 --dry
```