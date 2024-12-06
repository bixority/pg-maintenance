# pg-maintenance

PostgreSQL maintenance tool

Deletes rows from PostgreSQL table that are older than N days.

# Usage

## Plain binary run
```shell
    export DB_DSN="host=localhost port=5432 user=dev password=dev dbname=dev"
pg-maintenance --table dev --days 365 --batch 100 --timeout 0
```

### Arguments
`--table`: table name for cleanup
`--timestampColumn`: Name of the timestamp column (default: `created_at`)
`--days`: Delete rows older than N days (default: `0`)
`--batch`: Optional batch size for cleanup (default: `0`)
`--timeout`: Single db operation timeout in seconds (default: `60`)


## Container run
```shell
docker run --network host -e DB_DSN="host=localhost port=5432 user=dev password=dev dbname=dev" \
       ghcr.io/bixority/pg-maintenance:0.0.1 /pg_maintenance --table dev --days 10 --batch 100 --timeout 0
```
