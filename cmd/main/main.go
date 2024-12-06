package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/bixority/pg_maintenance/internal/module/pg"
)

func main() {
	now := time.Now()
	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")

	if dbUsername == "" {
		log.Fatal("Environment variable DB_USERNAME is required")
	}

	if dbPassword == "" {
		log.Fatal("Environment variable DB_PASSWORD is required")
	}

	var host string
	var port int
	var dbName string
	var table string
	var timestampColumn string
	var days int
	var batchSize int
	var timeout time.Duration

	flag.StringVar(&host, "host", "localhost", "Database host")
	flag.IntVar(&port, "port", 5432, "Database port")
	flag.StringVar(&dbName, "dbname", "", "Database name")
	flag.StringVar(&table, "table", "", "Table name for cleanup")
	flag.StringVar(&timestampColumn, "timestampColumn", "created_at", "Name of the timestamp column")
	flag.IntVar(&days, "days", 0, "Delete rows older than N days")
	flag.IntVar(&batchSize, "batch", 0, "Optional batch size for cleanup")
	flag.DurationVar(&timeout, "timeout", 60*time.Second, "Single db operation timeout in seconds")
	flag.Parse()

	if dbName == "" || table == "" || days <= 0 {
		log.Fatalln("All --dbname, --table and --days arguments are required")
	}

	var dbDSN = fmt.Sprintf(
		`host=%s port=%d dbname=%s user=%s password=%s`,
		host,
		port,
		dbName,
		dbUsername,
		dbPassword,
	)

	if !pg.IsValidTableName(table) {
		log.Fatalf("Invalid table name: %s\n", table)
	}

	db, err := sql.Open("postgres", dbDSN)

	if err != nil {
		log.Fatalf("ERROR: Failed to connect to database: %v\n", err)
	}

	defer db.Close()

	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ERROR: Database ping failed: %v\n", err)
	}

	log.Println("Connected to the database successfully")

	log.Printf("Cleaning up table %s by column %s for the records older than %d days with batch=%d\n",
		table,
		timestampColumn,
		days,
		batchSize,
	)

	args := []interface{}{now.AddDate(0, 0, -days)}

	if batchSize > 0 {
		args = append(args, batchSize)
	}

	subquery := fmt.Sprintf(
		`SELECT ctid FROM %s WHERE %s < $1 ORDER BY %s`,
		table,
		timestampColumn,
		timestampColumn,
	)

	if batchSize > 0 {
		subquery += " LIMIT $2"
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE ctid IN (%s);`, table, subquery)

	for {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), timeout)
		} else {
			ctx = context.WithoutCancel(context.Background())
			cancel = nil
		}

		tx, err := db.BeginTx(ctx, nil)

		if err != nil {
			log.Fatalf("ERROR: Failed to begin transaction: %v\n", err)
		}

		log.Println(query, args)

		result, err := db.ExecContext(ctx, query, args...)

		if err != nil {
			_ = tx.Rollback()
			log.Fatalf("ERROR: Failed to execute query: %v\n", err)
		}

		rowsAffected, err := result.RowsAffected()

		if err != nil {
			_ = tx.Rollback()
			log.Fatalf("ERROR: Failed to get rows affected: %v\n", err)
		}

		if err := tx.Commit(); err != nil {
			log.Fatalf("ERROR: Failed to commit transaction: %v\n", err)
		}

		if cancel != nil {
			cancel()
		}

		if rowsAffected == 0 {
			log.Println("No more rows to delete. Exiting.")

			break
		}

		log.Printf("Deleted %d rows\n", rowsAffected)

		if batchSize == 0 {
			break
		}
	}
}
