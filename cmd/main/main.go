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
)

func main() {
	now := time.Now()
	dbDSN := os.Getenv("DB_DSN")

	if dbDSN == "" {
		log.Fatal("Environment variable DB_DSN is required")
	}

	var table string
	var timestampColumn string
	var days time.Duration
	var batchSize int
	var timeout time.Duration

	flag.StringVar(&table, "table", "", "Table name for cleanup")
	flag.StringVar(&timestampColumn, "timestampColumn", "created_at", "Name of the timestamp column")
	flag.DurationVar(&days, "days", 0, "Delete rows older than N days")
	flag.IntVar(&batchSize, "batch", 0, "Optional batch size for cleanup")
	flag.DurationVar(&timeout, "timeout", 60, "Single db operation timeout in seconds")
	flag.Parse()

	if table == "" || days <= 0 {
		log.Fatalln("Both --table and --days arguments are required")
	}

	if !isValidTableName(table) {
		log.Fatalf("Invalid table name: %s\n", table)
	}

	db, err := sql.Open("postgres", dbDSN)

	if err != nil {
		log.Fatalf("ERROR: Failed to connect to database: %v\n", err)
	}

	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	// TODO: implement WithoutCancel, ctx isn't available outside the if-else scope
	//if timeout > 0 {
	//	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	//	defer cancel()
	//} else {
	//	ctx := context.WithoutCancel(context.Background())
	//}

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ERROR: Database ping failed: %v\n", err)
	}

	log.Println("Connected to the database successfully")
	log.Printf("Cleaning up table %s by column %s for the last %d days with batch=%d\n",
		table,
		timestampColumn,
		days,
		batchSize,
	)

	for {
		tx, err := db.BeginTx(ctx, nil)

		if err != nil {
			log.Fatalf("ERROR: Failed to begin transaction: %v\n", err)
		}

		query := fmt.Sprintf(
			`DELETE FROM "%s" WHERE "%s" < $1`,
			table,
			timestampColumn,
		)

		// TODO: get a cutoff timestamp from now - days
		var args = []interface{}{now - days}

		// TODO: implement batching, DELETE doesn't support LIMIT directly.
		//if batchSize > 0 {
		//	query += " LIMIT $2"
		//	args = append(args, batchSize)
		//}

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

func isValidTableName(name string) bool {
	for _, r := range name {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
	}

	return true
}
