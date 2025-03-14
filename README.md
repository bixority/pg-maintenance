# pg-maintenance

PostgreSQL maintenance tool

Deletes rows from PostgreSQL table that are older than N days.

# Usage

## Plain binary run
```shell
export DB_USERNAME="dev"
export DB_PASSWORD="dev"
pg-maintenance --host localhost --port 5432 --sslMode disable \
--table dev:created_at:365 --table dev2 --batch 100 --timeout 0s
```

### Arguments

`--host`: Database host

`--port`: Database port

`--sslMode`: SSL mode (default: require)

`--dbName`: Database name

`--table`: table(s) name for cleanup in a format "tableName\[:timestampColumn=created_at\[:days=0\]\]"

`--batch`: Optional batch size for cleanup (default: `0`)

`--timeout`: Single db operation timeout in seconds (default: `60s`)


## Container run
```shell
podman run --network host -e DB_USERNAME="dev" -e DB_PASSWORD="dev" \
       ghcr.io/bixority/pg-maintenance:v0.0.8 /pg-maintenance --host localhost --port 5432 \
       --sslMode disable --dbName dev --table dev:created_at:365 --table dev2 --batch 100 \
       --timeout 0s
```


## Test case

You can create a following table with data for the last 5 years:

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
