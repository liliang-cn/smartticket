use crate::{AuthConfig, DatabaseConfig, RedisConfig, ServerConfig, TelemetryConfig};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

pub type Result<T> = std::result::Result<T, ConfigError>;

#[derive(Debug, thiserror::Error)]
pub enum ConfigError {
    #[error("Configuration error: {0}")]
    Message(String),

    #[error("Failed to build config: {0}")]
    BuildError(#[from] config::ConfigError),

    #[error("Failed to deserialize config: {0}")]
    DeserializeError(#[from] serde_json::Error),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppConfig {
    pub environment: String,
    pub service_name: String,
    pub version: String,
    pub database: DatabaseConfig,
    pub redis: RedisConfig,
    pub server: ServerConfig,
    pub auth: AuthConfig,
    pub telemetry: TelemetryConfig,
    pub log_level: String,
    pub data_dir: PathBuf,
}

impl Default for AppConfig {
    fn default() -> Self {
        Self {
            environment: "development".to_string(),
            service_name: "smartticket".to_string(),
            version: "0.1.0".to_string(),
            database: DatabaseConfig::default(),
            redis: RedisConfig::default(),
            server: ServerConfig::default(),
            auth: AuthConfig::default(),
            telemetry: TelemetryConfig::default(),
            log_level: "info".to_string(),
            data_dir: PathBuf::from("./data"),
        }
    }
}

impl AppConfig {
    pub fn load() -> Result<Self> {
        let environment = "development"; // Fixed to development for now

        let config = config::Config::builder()
            .add_source(config::File::with_name(&format!("config/{}.yaml", environment)).required(false))
            .build()?;

        let app_config: AppConfig = config.try_deserialize().unwrap_or_else(|_| {
            tracing::warn!("Failed to load config file, using defaults");
            AppConfig::default()
        });

        tracing::info!("Loaded configuration for environment: {}", environment);

        Ok(app_config)
    }

    pub fn is_production(&self) -> bool {
        self.environment == "production"
    }

    pub fn is_development(&self) -> bool {
        self.environment == "development"
    }

    pub fn get_database_url(&self) -> String {
        format!(
            "postgresql://{}:{}@{}:{}/{}",
            self.database.username,
            self.database.password,
            self.database.host,
            self.database.port,
            self.database.database_name
        )
    }

    pub fn get_redis_url(&self) -> String {
        match (&self.redis.username, &self.redis.password) {
            (Some(username), Some(password)) => {
                format!(
                    "redis://{}:{}@{}:{}/{}",
                    username, password, self.redis.host, self.redis.port, self.redis.database
                )
            }
            (None, Some(password)) => {
                format!(
                    "redis://:{}@{}:{}/{}",
                    password, self.redis.host, self.redis.port, self.redis.database
                )
            }
            (Some(username), None) => {
                format!(
                    "redis://{}@{}:{}/{}",
                    username, self.redis.host, self.redis.port, self.redis.database
                )
            }
            (None, None) => {
                format!(
                    "redis://{}:{}/{}",
                    self.redis.host, self.redis.port, self.redis.database
                )
            }
        }
    }
}
