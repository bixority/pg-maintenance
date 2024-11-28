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
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatalf("Environment variable DB_URL is required")
	}

	// Parse CLI arguments
	var tableName string
	var dtcrea string
	var batchSize int

	flag.StringVar(&tableName, "table", "", "Table name for deletion")
	flag.StringVar(&dtcrea, "dtcrea", "", "Delete rows with dtcrea less than this date (YYYY-MM-DD)")
	flag.IntVar(&batchSize, "batch", 0, "Optional batch size for deletion")
	flag.Parse()

	if tableName == "" || dtcrea == "" {
		log.Fatalf("Both --table and --dtcrea arguments are required")
	}

	// Validate table name (it must be alphanumeric or underscore)
	if !isValidTableName(tableName) {
		log.Fatalf("Invalid table name: %s", tableName)
	}

	// Open database connection (use connection pooling)
	db, err := sql.Open("postgres", dbURL)
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
		// Prepare the SQL query using parameterized query for dtcrea
		query := fmt.Sprintf(`DELETE FROM "%s" WHERE dtcrea < $1`, tableName)
		if batchSize > 0 {
			query += fmt.Sprintf(" LIMIT %d", batchSize)
		}

		// Execute the query with dtcrea as the parameter
		result, err := db.ExecContext(ctx, query, dtcrea)
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
