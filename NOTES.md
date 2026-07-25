## Debugging

1. Start a postgres container
   ```bash
   docker run --name deed_postgres_testing -e POSTGRES_PASSWORD=password -v deed_postgres:/var/lib/postgresql/data -p 5433:5433 -d postgres:14.9
   ```
2. Generate Mermaid diagram from postgres schema
   ```
   mermerd -c "postgres://postgres:my_secure_password@127.0.0.1:5433/postgres" -s public --outputFileName er.mmd
   ```
3. Convert Mermaid diagram to SVG
   ```
   docker run --rm -u `id -u`:`id -g` -v $$PWD/docs/er:/data minlag/mermaid-cli -i er.mmd
   ```