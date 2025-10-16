use redis::{Client, Connection, RedisError, RedisResult};
use smartticket_shared_config::RedisConfig;
use smartticket_shared_error::{Result, SmartTicketError};
use std::time::Duration;
use tokio::time::timeout;
use tracing::{info, warn};

pub struct RedisPool {
    client: Client,
    config: RedisConfig,
}

impl RedisPool {
    pub fn new(config: &RedisConfig) -> Result<Self> {
        let connection_string = match (&config.username, &config.password) {
            (Some(username), Some(password)) => {
                format!(
                    "redis://{}:{}@{}:{}/{}",
                    username, password, config.host, config.port, config.database
                )
            }
            (None, Some(password)) => {
                format!(
                    "redis://:{}@{}:{}/{}",
                    password, config.host, config.port, config.database
                )
            }
            (Some(username), None) => {
                format!(
                    "redis://{}@{}:{}/{}",
                    username, config.host, config.port, config.database
                )
            }
            (None, None) => {
                format!(
                    "redis://{}:{}/{}",
                    config.host, config.port, config.database
                )
            }
        };

        let client = Client::open(connection_string).map_err(|e| {
            SmartTicketError::Configuration(format!("Invalid Redis configuration: {}", e))
        })?;

        info!(
            "Redis pool configured for {}:{}/{}",
            config.host, config.port, config.database
        );

        Ok(Self {
            client,
            config: config.clone(),
        })
    }

    pub async fn get_connection(&self) -> Result<Connection> {
        let conn = self
            .client
            .get_connection()
            .map_err(|e| SmartTicketError::Redis(e))?;

        Ok(conn)
    }

    pub async fn health_check(&self) -> Result<()> {
        let mut conn = timeout(
            Duration::from_secs(self.config.connection_timeout),
            self.get_connection(),
        )
        .await
        .map_err(|_| {
            SmartTicketError::Redis(RedisError::from((
                redis::ErrorKind::IoError,
                "Redis connection timeout",
            )))
        })??;

        let pong: String = redis::cmd("PING")
            .query(&mut conn)
            .map_err(|e| SmartTicketError::Redis(e))?;

        if pong != "PONG" {
            return Err(SmartTicketError::Redis(RedisError::from((
                redis::ErrorKind::ResponseError,
                "Unexpected PING response",
            ))));
        }

        info!("Redis health check passed");
        Ok(())
    }

    pub async fn test_connection(&self) -> Result<()> {
        match self.health_check().await {
            Ok(()) => {
                info!("Redis connection test successful");
                Ok(())
            }
            Err(e) => {
                warn!("Redis connection test failed: {}", e);
                Err(e)
            }
        }
    }

    pub fn config(&self) -> &RedisConfig {
        &self.config
    }
}

pub struct RedisService {
    pool: RedisPool,
}

impl RedisService {
    pub fn new(config: &RedisConfig) -> Result<Self> {
        let pool = RedisPool::new(config)?;
        Ok(Self { pool })
    }

    pub async fn get_connection(&self) -> Result<Connection> {
        self.pool.get_connection().await
    }

    pub async fn set<T: redis::ToRedisArgs>(
        &self,
        key: &str,
        value: T,
        ttl_seconds: Option<u64>,
    ) -> Result<()> {
        let mut conn = self.get_connection().await?;

        if let Some(ttl) = ttl_seconds {
            redis::cmd("SETEX")
                .arg(key)
                .arg(ttl)
                .arg(value)
                .query::<()>(&mut conn)
                .map_err(|e| SmartTicketError::Redis(e))?;
        } else {
            redis::cmd("SET")
                .arg(key)
                .arg(value)
                .query::<()>(&mut conn)
                .map_err(|e| SmartTicketError::Redis(e))?;
        }

        Ok(())
    }

    pub async fn get<T: redis::FromRedisValue>(&self, key: &str) -> Result<Option<T>> {
        let mut conn = self.get_connection().await?;

        let result: RedisResult<T> = redis::cmd("GET").arg(key).query(&mut conn);

        match result {
            Ok(value) => Ok(Some(value)),
            Err(e) => {
                if e.kind() == redis::ErrorKind::TypeError {
                    // Key doesn't exist
                    Ok(None)
                } else {
                    Err(SmartTicketError::Redis(e))
                }
            }
        }
    }

    pub async fn delete(&self, key: &str) -> Result<bool> {
        let mut conn = self.get_connection().await?;

        let deleted: i32 = redis::cmd("DEL")
            .arg(key)
            .query(&mut conn)
            .map_err(|e| SmartTicketError::Redis(e))?;

        Ok(deleted > 0)
    }

    pub async fn exists(&self, key: &str) -> Result<bool> {
        let mut conn = self.get_connection().await?;

        let exists: i32 = redis::cmd("EXISTS")
            .arg(key)
            .query(&mut conn)
            .map_err(|e| SmartTicketError::Redis(e))?;

        Ok(exists > 0)
    }

    pub async fn expire(&self, key: &str, seconds: u64) -> Result<bool> {
        let mut conn = self.get_connection().await?;

        let result: i32 = redis::cmd("EXPIRE")
            .arg(key)
            .arg(seconds)
            .query(&mut conn)
            .map_err(|e| SmartTicketError::Redis(e))?;

        Ok(result > 0)
    }

    pub async fn increment(&self, key: &str) -> Result<i64> {
        let mut conn = self.get_connection().await?;

        let value: i64 = redis::cmd("INCR")
            .arg(key)
            .query(&mut conn)
            .map_err(|e| SmartTicketError::Redis(e))?;

        Ok(value)
    }

    pub async fn increment_by(&self, key: &str, increment: i64) -> Result<i64> {
        let mut conn = self.get_connection().await?;

        let value: i64 = redis::cmd("INCRBY")
            .arg(key)
            .arg(increment)
            .query(&mut conn)
            .map_err(|e| SmartTicketError::Redis(e))?;

        Ok(value)
    }

    pub async fn health_check(&self) -> Result<()> {
        self.pool.health_check().await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_redis_service() {
        let config = RedisConfig::default();

        // This test requires a running Redis instance
        match RedisService::new(&config) {
            Ok(service) => {
                // Test basic operations
                let test_key = "test_key";
                let test_value = "test_value";

                // Set value
                service.set(test_key, test_value, Some(60)).await.unwrap();

                // Get value
                let retrieved: Option<String> = service.get(test_key).await.unwrap();
                assert_eq!(retrieved, Some(test_value.to_string()));

                // Check existence
                assert!(service.exists(test_key).await.unwrap());

                // Delete value
                assert!(service.delete(test_key).await.unwrap());
                assert!(!service.exists(test_key).await.unwrap());
            }
            Err(e) => {
                warn!("Skipping Redis test - no Redis available: {}", e);
            }
        }
    }
}
