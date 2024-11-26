#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <argp.h>
#include <libpq-fe.h>
#include <time.h>

// Define command-line arguments
const char *argp_program_version = "pg_maintenance 0.1";
const char *argp_program_bug_address = "<bixority@gmail.com>";

/* Program documentation. */
static char doc[] = "A program to delete rows from a PostgreSQL table with secure parameterized queries.";

/* A description of the arguments we accept. */
static char args_doc[] = "<table_name> <dtcrea_value>";

/* The options we understand. */
static struct argp_option options[] = {
    {"host", 'H', "HOST", 0, "Database host (default: localhost)"},
    {"port", 'P', "PORT", 0, "Database port (default: 5432)"},
    {"dbname", 'D', "DBNAME", 0, "Database name (default: postgres)"},
    {"user", 'U', "USER", 0, "Database user (default: postgres)"},
    {"password", 'W', "PASSWORD", 0, "Database password"},
    {"batch", 'B', "BATCH", 0, "Maximum number of rows to delete per loop iteration"},
    {0}
};

/* Used by main to communicate with parse_opt. */
struct arguments {
    char *table_name;
    char *dtcrea_value;
    char *host;
    char *port;
    char *dbname;
    char *user;
    char *password;
    int batch; // Default: 0 (no batching)
};

/* Parse a single option. */
static error_t parse_opt(int key, char *arg, struct argp_state *state) {
    struct arguments *arguments = state->input;

    switch (key) {
    case 'H': arguments->host = arg; break;
    case 'P': arguments->port = arg; break;
    case 'D': arguments->dbname = arg; break;
    case 'U': arguments->user = arg; break;
    case 'W': arguments->password = arg; break;
    case 'B': arguments->batch = atoi(arg); break;

    case ARGP_KEY_ARG:
        if (state->arg_num == 0) {
            arguments->table_name = arg;
        } else if (state->arg_num == 1) {
            arguments->dtcrea_value = arg;
        } else {
            argp_usage(state);
        }
        break;

    case ARGP_KEY_END:
        if (state->arg_num < 2) {
            argp_usage(state);
        }
        break;

    default: return ARGP_ERR_UNKNOWN;
    }
    return 0;
}

/* Our argp parser. */
static struct argp argp = {options, parse_opt, args_doc, doc};

/* Function to get the current timestamp in ISO 8601 format. */
void get_iso_timestamp(char *buffer, size_t size) {
    time_t now = time(NULL);
    struct tm *tm_info = gmtime(&now);
    strftime(buffer, size, "%Y-%m-%dT%H:%M:%SZ", tm_info);
}

/* Function for JSON logging. */
void log_json(FILE *output, const char *level, const char *message, const char *context) {
    char timestamp[32];
    get_iso_timestamp(timestamp, sizeof(timestamp));

    fprintf(output, "{\"timestamp\": \"%s\", \"level\": \"%s\", \"message\": \"%s\"", timestamp, level, message);
    if (context) {
        fprintf(output, ", \"context\": \"%s\"", context);
    }
    fprintf(output, "}\n");
}

/* Log functions for convenience. */
void log_info(const char *message, const char *context) {
    log_json(stdout, "INFO", message, context);
}

void log_error(const char *message, const char *context) {
    log_json(stderr, "ERROR", message, context);
}

void finish_with_error(PGconn *conn) {
    log_error(PQerrorMessage(conn), "Database connection error");
    PQfinish(conn);
    exit(1);
}

int main(int argc, char *argv[]) {
    struct arguments arguments;

    /* Default values. */
    arguments.host = "localhost";
    arguments.port = "5432";
    arguments.dbname = "postgres";
    arguments.user = "postgres";
    arguments.password = NULL;
    arguments.batch = 0; // No batching by default

    /* Parse command-line arguments. */
    argp_parse(&argp, argc, argv, 0, 0, &arguments);

    if (arguments.password == NULL) {
        log_error("--password is required", "Command-line argument error");
        exit(1);
    }

    /* Build the connection string. */
    char conninfo[256];
    snprintf(conninfo, sizeof(conninfo),
             "host=%s port=%s dbname=%s user=%s password=%s sslmode=require",
             arguments.host, arguments.port, arguments.dbname,
             arguments.user, arguments.password);

    /* Connect to PostgreSQL. */
    PGconn *conn = PQconnectdb(conninfo);
    if (PQstatus(conn) != CONNECTION_OK) {
        finish_with_error(conn);
    }

    log_info("Connected to the database.", NULL);

    /* Begin transaction. */
    PGresult *res = PQexec(conn, "BEGIN");
    if (PQresultStatus(res) != PGRES_COMMAND_OK) {
        PQclear(res);
        finish_with_error(conn);
    }
    PQclear(res);

    log_info("Transaction started.", NULL);

    /* Dynamic DELETE query with batching support. */
    char query[512];
    if (arguments.batch > 0) {
        snprintf(query, sizeof(query),
                 "DELETE FROM \"%s\" WHERE dtcrea < $1::date LIMIT %d",
                 arguments.table_name, arguments.batch);
    } else {
        snprintf(query, sizeof(query),
                 "DELETE FROM \"%s\" WHERE dtcrea < $1::date",
                 arguments.table_name);
    }

    int total_deleted = 0;
    int rows_deleted;

    do {
        /* Execute the parametrized query. */
        const char *param_values[1] = {arguments.dtcrea_value};
        res = PQexecParams(conn, query,      // SQL query
                           1,               // Number of parameters
                           NULL,            // Parameter types (can be NULL)
                           param_values,    // Parameter values
                           NULL,            // Parameter lengths (can be NULL)
                           NULL,            // Parameter formats (can be NULL)
                           0);              // Result format (0 = text)
        if (PQresultStatus(res) != PGRES_COMMAND_OK) {
            log_error(PQerrorMessage(conn), "Query execution error");
            PQclear(res);
            PQexec(conn, "ROLLBACK"); // Rollback in case of failure
            finish_with_error(conn);
        }

        /* Check the number of rows affected in this iteration. */
        rows_deleted = atoi(PQcmdTuples(res));
        total_deleted += rows_deleted;
        PQclear(res);

        char context[64];
        snprintf(context, sizeof(context), "Batch deleted rows: %d", rows_deleted);
        log_info("Rows deleted in this batch.", context);

    } while (arguments.batch > 0 && rows_deleted > 0);

    char total_context[64];
    snprintf(total_context, sizeof(total_context), "Total rows deleted: %d", total_deleted);
    log_info("Deletion completed.", total_context);

    /* Commit transaction. */
    res = PQexec(conn, "COMMIT");
    if (PQresultStatus(res) != PGRES_COMMAND_OK) {
        log_error(PQerrorMessage(conn), "Transaction commit error");
        PQclear(res);
        finish_with_error(conn);
    }
    PQclear(res);

    log_info("Transaction committed successfully.", NULL);

    /* Close the connection. */
    PQfinish(conn);
    log_info("Database connection closed.", NULL);

    return 0;
}
