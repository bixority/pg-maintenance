# pg-maintenance

PostgreSQL maintenance tool

Deletes rows from PostgreSQL table that are older than N days.

# Usage

## Plain binary run
```shell
export DB_DSN="host=localhost port=5432 user=dev password=dev dbname=dev"
pg-maintenance --table dev --days 365 --batch 100000 --timeout 0
```

Note: you might want to disable TLS by adding `"sslmode=disable"` into `DB_DSN`.


## Container run
```shell

```