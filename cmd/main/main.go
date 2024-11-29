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
	// Environment variable for DB connection string
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		log.Fatalf("Environment variable DB_DSN is required")
	}

	// Parse CLI arguments
	var table string
	var timestampColumn string
	var timestamp string
	var batchSize int

	flag.StringVar(&table, "table", "", "Table name for cleanup")
	flag.StringVar(&timestampColumn, "timestampColumn", "created_at", "Name of the timestamp column")
	flag.StringVar(&timestamp, "timestamp", "", "Delete rows older than this date (YYYY-MM-DD)")
	flag.IntVar(&batchSize, "batch", 0, "Optional batch size for cleanup")
	flag.Parse()

	if table == "" || timestamp == "" {
		log.Fatalf("Both --table and --timestamp arguments are required")
	}

	// Validate table name (it must be alphanumeric or underscore)
	if !isValidTableName(table) {
		log.Fatalf("Invalid table name: %s", table)
	}

	// Open database connection (use connection pooling)
	db, err := sql.Open("postgres", dbDSN)

	if err != nil {
		log.Fatalf("ERROR: Failed to connect to database: %v\n", err)
	}

	defer db.Close()

	// Ensure the database connection is valid
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ERROR: Database ping failed: %v\n", err)
	}

	fmt.Println("Connected to the database successfully")

	// Perform the deletion in batches if specified
	for {
		// Prepare the SQL query using parameterized query for timestamp
		query := fmt.Sprintf(`DELETE FROM "%s" WHERE "%s" < $1`, table, timestampColumn)

		if batchSize > 0 {
			query += fmt.Sprintf(" LIMIT %d", batchSize)
		}

		// Execute the query with timestamp as the parameter
		result, err := db.ExecContext(ctx, query, timestamp)

		if err != nil {
			log.Fatalf("ERROR: Failed to execute query: %v\n", err)
		}

		// Check how many rows were affected
		rowsAffected, err := result.RowsAffected()

		if err != nil {
			log.Fatalf("ERROR: Failed to get rows affected: %v\n", err)
		}

		if rowsAffected == 0 {
			fmt.Println("No more rows to delete. Exiting.")
			break
		}

		fmt.Printf("Deleted %d rows\n", rowsAffected)

		// Exit early if batch mode is not enabled
		if batchSize == 0 {
			break
		}
	}
}

// isValidTableName checks if the table name contains only allowed characters (alphanumeric and underscore)
func isValidTableName(name string) bool {
	for _, r := range name {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
	}
	return true
}
