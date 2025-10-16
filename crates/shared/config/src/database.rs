use serde::{Deserialize, Serialize};
use validator::Validate;

#[derive(Debug, Clone, Serialize, Deserialize, Validate)]
pub struct DatabaseConfig {
    #[validate(length(min = 1))]
    pub host: String,
    #[validate(range(min = 1, max = 65535))]
    pub port: u16,
    #[validate(length(min = 1))]
    pub database_name: String,
    #[validate(length(min = 1))]
    pub username: String,
    pub password: String,
    pub ssl_mode: SslMode,
    pub max_connections: u32,
    pub min_connections: u32,
    pub connect_timeout: u64,
    pub idle_timeout: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SslMode {
    Disable,
    Prefer,
    Require,
}

impl Default for DatabaseConfig {
    fn default() -> Self {
        Self {
            host: "localhost".to_string(),
            port: 5432,
            database_name: "smartticket".to_string(),
            username: "postgres".to_string(),
            password: "postgres".to_string(),
            ssl_mode: SslMode::Prefer,
            max_connections: 20,
            min_connections: 5,
            connect_timeout: 30,
            idle_timeout: 600,
        }
    }
}
