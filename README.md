# pg-maintenance

PostgreSQL maintenance tool

Deletes rows from PostgreSQL table that are older than N days.

# Usage

## Plain binary run
```shell
export DB_USERNAME="dev"
export DB_PASSWORD="dev"
pg-maintenance --table dev --days 365 --batch 100 --timeout 0s
```

### Arguments

`--host`: Database host

`--port`: Database port

`--dbname`: Database name

`--table`: table name for cleanup

`--timestampColumn`: Name of the timestamp column (default: `created_at`)

`--days`: Delete rows older than N days (default: `0`)

`--batch`: Optional batch size for cleanup (default: `0`)

`--timeout`: Single db operation timeout in seconds (default: `60s`)


## Container run
```shell
podman run --network host -e DB_USERNAME="dev" -e DB_PASSWORD="dev" \
       ghcr.io/bixority/pg-maintenance:0.0.1 /pg_maintenance --host localhost --port 5432 --dbname dev \
       --table dev --days 10 --batch 100 --timeout 0s
```


## Test case

You can create a following table with data for the last 5 years:

```postgresql
CREATE TABLE dev (id BIGSERIAL, created_at TIMESTAMP WITH TIME ZONE);

INSERT INTO dev (created_at)
SELECT 
    NOW() - INTERVAL '5 years' + (INTERVAL '5 years' * (i / 100000000.0))
FROM 
    generate_series(1, 100000000) AS g(i);

CREATE INDEX ON dev (created_at);
```
