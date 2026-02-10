use anyhow::{Context, Result, bail};
use chrono::{Duration, Utc};
use clap::Parser;
use log::{error, info};
use sqlx::postgres::{PgConnectOptions, PgPoolOptions, PgSslMode};
use std::str::FromStr;
use std::time::Duration as StdDuration;

mod module;
use module::pg::is_valid_tname;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Database host
    #[arg(long, default_value = "localhost")]
    host: String,

    /// Database port
    #[arg(long, default_value_t = 5432)]
    port: u16,

    /// SSL mode
    #[arg(long, default_value = "require")]
    ssl_mode: String,

    /// Database name
    #[arg(long)]
    db_name: String,

    /// Table(s) in a table[:`timestampColumn=created_at`[:days=0]] format
    #[arg(long = "table")]
    tables: Vec<String>,

    /// Optional batch size for cleanup
    #[arg(long, default_value_t = 1000)]
    batch: i64,

    /// Single db operation timeout in seconds
    #[arg(long, default_value = "60s", value_parser = parse_duration)]
    timeout: StdDuration,

    /// Database username
    #[arg(long, env = "DB_USERNAME")]
    db_username: String,

    /// Database password
    #[arg(long, env = "DB_PASSWORD")]
    db_password: String,
}

fn parse_duration(s: &str) -> Result<StdDuration, String> {
    let Some(s) = s.strip_suffix('s') else {
        return Err("Duration must end with 's' (e.g., '60s')".to_string());
    };

    s.parse::<u64>()
        .map(StdDuration::from_secs)
        .map_err(|e| format!("Invalid duration: {e}"))
}

struct TableConfig {
    name: String,
    timestamp_column: String,
    days: i64,
}

impl FromStr for TableConfig {
    type Err = anyhow::Error;

    fn from_str(s: &str) -> Result<Self> {
        let parts: Vec<&str> = s.split(':').collect();

        let name = parts[0].to_string();
        if !is_valid_tname(&name) {
            bail!("Invalid table name: {name}");
        }

        let timestamp_column = parts
            .get(1)
            .filter(|s| !s.is_empty())
            .map(|s| {
                if is_valid_tname(s) {
                    Ok(s.to_string())
                } else {
                    bail!("Invalid timestamp column: {s}")
                }
            })
            .transpose()?
            .unwrap_or_else(|| "created_at".to_string());

        let days = parts
            .get(2)
            .map(|s| s.parse::<i64>())
            .transpose()
            .context("Failed to parse days")?
            .unwrap_or(0);

        Ok(TableConfig {
            name,
            timestamp_column,
            days,
        })
    }
}

fn parse_ssl_mode(mode: &str) -> Result<PgSslMode> {
    match mode {
        "disable" => Ok(PgSslMode::Disable),
        "require" => Ok(PgSslMode::Require),
        "verify-ca" => Ok(PgSslMode::VerifyCa),
        "verify-full" => Ok(PgSslMode::VerifyFull),
        _ => bail!("Unsupported SSL mode: {mode}"),
    }
}

async fn with_timeout<F, T, E>(timeout: StdDuration, fut: F) -> Result<T>
where
    F: Future<Output = Result<T, E>>,
    E: Into<anyhow::Error>,
{
    if timeout.as_secs() > 0 {
        tokio::time::timeout(timeout, fut)
            .await
            .map_err(|_| anyhow::anyhow!("Operation timed out"))?
            .map_err(Into::into)
    } else {
        fut.await.map_err(Into::into)
    }
}

async fn cleanup_table(
    pool: &sqlx::PgPool,
    config: &TableConfig,
    batch_size: i64,
    timeout: StdDuration,
) -> Result<()> {
    info!(
        "Cleaning up table {} by column {} for records older than {} days (batch={})",
        config.name, config.timestamp_column, config.days, batch_size
    );

    let cutoff = Utc::now() - Duration::days(config.days);

    // Build SQL query once with identifiers (which must use format!)
    // Then bind values using proper parameterization
    let stmt = if batch_size > 0 {
        format!(
            r#"DELETE FROM "{table}"
               WHERE ctid IN (
                   SELECT ctid FROM "{table}"
                   WHERE "{col}" < $1
                   ORDER BY "{col}"
                   LIMIT $2
               )"#,
            table = config.name,
            col = config.timestamp_column
        )
    } else {
        format!(
            r#"DELETE FROM "{table}"
               WHERE ctid IN (
                   SELECT ctid FROM "{table}"
                   WHERE "{col}" < $1
                   ORDER BY "{col}"
               )"#,
            table = config.name,
            col = config.timestamp_column
        )
    };

    loop {
        let mut tx = with_timeout(timeout, pool.begin()).await?;

        // Build query with proper .bind() for values
        let query = if batch_size > 0 {
            sqlx::query(&stmt).bind(cutoff).bind(batch_size)
        } else {
            sqlx::query(&stmt).bind(cutoff)
        };

        let result = with_timeout(timeout, query.execute(&mut *tx)).await;

        match result {
            Ok(res) => {
                let rows_affected = res.rows_affected();

                with_timeout(timeout, tx.commit()).await?;

                if rows_affected == 0 {
                    info!(
                        "No more rows to delete in table {}. Moving to next table.",
                        config.name
                    );
                    break;
                }

                info!("Deleted {} rows from {}", rows_affected, config.name);

                if batch_size == 0 {
                    break;
                }
            }
            Err(e) => {
                error!("Failed to execute query for table {}: {}", config.name, e);
                let _ = tx.rollback().await;
                return Err(e);
            }
        }
    }

    Ok(())
}

#[tokio::main]
async fn main() -> Result<()> {
    env_logger::init_from_env(env_logger::Env::default().default_filter_or("info"));

    let args = Args::parse();

    if args.tables.is_empty() {
        bail!("At least one --table argument is required");
    }

    let opts = PgConnectOptions::new()
        .host(&args.host)
        .port(args.port)
        .username(&args.db_username)
        .password(&args.db_password)
        .database(&args.db_name)
        .ssl_mode(parse_ssl_mode(&args.ssl_mode)?);

    let pool = PgPoolOptions::new()
        .max_connections(5)
        .acquire_timeout(StdDuration::from_secs(10))
        .connect_with(opts)
        .await
        .context("Failed to connect to database")?;

    info!("Connected to the database successfully");

    for table_str in &args.tables {
        let config = match TableConfig::from_str(table_str) {
            Ok(c) => c,
            Err(e) => {
                error!("Invalid table format {table_str}: {e}");
                continue;
            }
        };

        if let Err(e) = cleanup_table(&pool, &config, args.batch, args.timeout).await {
            error!("Failed to cleanup table {}: {}", config.name, e);
        }
    }

    Ok(())
}
