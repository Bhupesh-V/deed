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

- AST parse the CEHCK clause.
- [algebraic only clauses] Try to satisfy the expression using Z3 https://github.com/mitchellh/go-z3.
- Reject clauses with regex expression, ask user to move to config.


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