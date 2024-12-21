package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/bixority/pg_maintenance/internal/module/pg"
)

type arrayFlags []string

// String is an implementation of the flag.Value interface
func (i *arrayFlags) String() string {
	return fmt.Sprintf("%v", *i)
}

// Set is an implementation of the flag.Value interface
func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)

	return nil
}

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
	var sslMode string
	var dbName string
	var tableName string
	var timestampColumn string
	var days int
	var batchSize int
	var timeout time.Duration
	var tables arrayFlags

	flag.StringVar(&host, "host", "localhost", "Database host")
	flag.IntVar(&port, "port", 5432, "Database port")
	flag.StringVar(&sslMode, "sslMode", "require", "SSL mode")
	flag.StringVar(&dbName, "dbname", "", "Database name")
	flag.Var(&tables, "table", "Table(s) in a table:[timestampColumn=created_at[:days=0]] format.")
	flag.IntVar(&batchSize, "batch", 0, "Optional batch size for cleanup")
	flag.DurationVar(&timeout, "timeout", 60*time.Second, "Single db operation timeout in seconds")
	flag.Parse()

	if dbName == "" || len(tables) == 0 {
		log.Fatalln("All --dbname and --table arguments are required")
	}

	if sslMode != "require" && sslMode != "disable" && sslMode != "verify-full" && sslMode != "verify-cy" {
		log.Fatalf(
			"Unsupported sslMode %s, \"require\" (default), "+
				"\"verify-full\", \"verify-ca\", and \"disable\" supported\n",
			sslMode,
		)
	}

	var dbDSN = fmt.Sprintf(
		`host=%s port=%d dbname=%s user=%s password=%s sslmode=%s`,
		host,
		port,
		dbName,
		dbUsername,
		dbPassword,
		sslMode,
	)

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

	for _, table := range tables {
		log.Println(table)

		parts := strings.Split(table, ":")
		partCnt := len(parts)

		if partCnt < 1 || partCnt > 3 {
			log.Println("Invalid format: ", parts)
			continue
		}

		for i, part := range parts {
			if part == "" {
				log.Fatalf("Invalid part %d format: %s\n", i, part)
			}
		}

		tableName = parts[0]

		if !pg.IsValidTName(tableName) {
			log.Fatalf("Invalid table name: %s\n", table)
		}

		if partCnt > 1 {
			timestampColumn = parts[1]

			if timestampColumn == "" {
				timestampColumn = "created_at"
			} else {
				if !pg.IsValidTName(timestampColumn) {
					log.Fatalf("Invalid timestamp column name: %s\n", table)
				}
			}

			if partCnt > 2 {
				days, err = strconv.Atoi(parts[2])

				if err != nil {
					log.Fatalln("Error parsing days integer value: ", err)
				}
			} else {
				days = 0
			}
		} else {
			timestampColumn = "created_at"
		}

		log.Printf("Cleaning up table %s by column %s for the records older than %d days with batch=%d\n",
			tableName,
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
			tableName,
			timestampColumn,
			timestampColumn,
		)

		if batchSize > 0 {
			subquery += " LIMIT $2"
		}

		query := fmt.Sprintf(`DELETE FROM %s WHERE ctid IN (%s);`, tableName, subquery)

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
}
