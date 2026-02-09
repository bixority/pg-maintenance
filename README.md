# pg-maintenance

PostgreSQL maintenance tool (Rust, sqlx)

Deletes rows from PostgreSQL table(s) that are older than N days, optionally in batches.

# Build

- Native binary:
  - Debug: `cargo build`
  - Release: `cargo build --release` (binary at `target/release/pg-maintenance`)
- Container image: `podman build -t pg-maintenance:local -f Containerfile .`

# Usage

## Plain binary run
```shell
export DB_USERNAME="dev"
export DB_PASSWORD="dev"
./target/release/pg-maintenance --host localhost --port 5432 --sslMode disable \
  --dbName dev \
  --table dev:created_at:365 --table dev2 --batch 100 --timeout 0s
```

### Arguments

- `--host`: Database host (default: `localhost`)
- `--port`: Database port (default: `5432`)
- `--sslMode`: SSL mode: `disable`, `require` (default), `verify-ca`, `verify-full`
- `--dbName`: Database name (required)
- `--table`: Table(s) in format `table[:timestampColumn=created_at[:days=0]]`, can be repeated
- `--batch`: Optional batch size for cleanup (default: `0` means delete all matching rows in one transaction)
- `--timeout`: Single DB operation timeout in seconds, e.g. `60s`, `0s` to disable

Environment variables:
- `DB_USERNAME` (required)
- `DB_PASSWORD` (required)

## Container run
```shell
podman run --rm --network host \
  -e DB_USERNAME="dev" -e DB_PASSWORD="dev" \
  pg-maintenance:local /pg-maintenance --host localhost --port 5432 \
  --sslMode disable --dbName dev --table dev:created_at:365 --table dev2 --batch 100 \
  --timeout 0s
```

## Notes
- The tool deletes using `ctid` of rows selected by timestamp condition, ordered by the timestamp column, which works efficiently with an index on that column.
- For best performance create an index on the timestamp column:

```postgresql
CREATE INDEX ON dev (created_at);
CREATE INDEX ON dev2 (created_at);
```

## Test case

You can create the following tables with data for the last 5 years:

```postgresql
CREATE TABLE dev (id BIGSERIAL, created_at TIMESTAMP WITH TIME ZONE);
CREATE TABLE dev2 (id BIGSERIAL, created_at TIMESTAMP WITH TIME ZONE);

INSERT INTO dev (created_at)
SELECT 
    NOW() - INTERVAL '5 years' + (INTERVAL '5 years' * (i / 100000000.0))
FROM 
    generate_series(1, 100000000) AS g(i);

INSERT INTO dev2 (created_at)
SELECT
    NOW() - INTERVAL '5 years' + (INTERVAL '5 years' * (i / 100000000.0))
FROM
    generate_series(1, 100000000) AS g(i);

CREATE INDEX ON dev (created_at);
CREATE INDEX ON dev2 (created_at);
```
