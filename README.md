# deed

## Why?

- Your database is a heavy lifter, i.e you now have to opimise your **SQL queries** and **indexes**.
- You don't have access to production data due to compliance issues.
- You are a master procastinator on load testing your APIs.

## Why not?

- You don't know how to use a modern RDBMS.
- Strict validation of business data on frontend.

## For whom?

1. Database Engineers.
2. Data Engineers, Backend Engineers.
3. DevOps Engineers & SREs.

## Installation

TODO

## Supported Databases

- Postgres

## Usage

### Ideology

Since the choice of how you write SQL queries and indexes completely depends on how your UI/UX looks like (which changes often), **each deed run is supposed to be targetted, meaning you don't ingest data in your whole schema in one go and never look back.**

Instead, your ideal workflow should look like,

1. Start a container with your up-to-date schema applied.
2. Have list of tables you want to generate data for, these are the tables that you have to optimise your queries/indexes for.
3. Seed the data using deed.
4. Rewrite and test your queries/indexes.
5. Destroy the db container.

Deed can definitely be used to fill up your schema with test data and by extension be used to demo your apps, however that was not my original intent while building it.

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

Sample Output

```
```