use serde::{Deserialize, Serialize};
use validator::Validate;

#[derive(Debug, Clone, Serialize, Deserialize, Validate)]
pub struct RedisConfig {
    #[validate(length(min = 1))]
    pub host: String,
    #[validate(range(min = 1, max = 65535))]
    pub port: u16,
    pub username: Option<String>,
    pub password: Option<String>,
    pub database: i64,
    pub max_connections: u32,
    pub connection_timeout: u64,
    pub command_timeout: u64,
}

impl Default for RedisConfig {
    fn default() -> Self {
        Self {
            host: "localhost".to_string(),
            port: 6379,
            username: None,
            password: None,
            database: 0,
            max_connections: 10,
            connection_timeout: 5,
            command_timeout: 5,
        }
    }
}
