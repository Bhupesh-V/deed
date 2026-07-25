## Debugging

1. Generate Mermaid diagram from postgres schema
   ```
   mermerd -c "postgres://postgres:my_secure_password@127.0.0.1:5433/postgres" -s public --outputFileName er.mmd
   ```
2. Convert Mermaid diagram to SVG
   ```
   docker run --rm -u `id -u`:`id -g` -v $$PWD/docs/er:/data minlag/mermaid-cli -i er.mmd
   ```