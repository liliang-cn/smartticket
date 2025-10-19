use serde::{Deserialize, Serialize};
use validator::Validate;

#[derive(Debug, Clone, Serialize, Deserialize, Validate)]
pub struct ServerConfig {
    #[validate(range(min = 1, max = 65535))]
    pub grpc_port: u16,
    #[validate(range(min = 1, max = 65535))]
    pub http_port: u16,
    #[validate(length(min = 1))]
    pub host: String,
    pub max_request_size: usize,
    pub request_timeout: u64,
    pub keep_alive: u64,
    pub tls: Option<TlsConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TlsConfig {
    pub enabled: bool,
    pub cert_path: String,
    pub key_path: String,
    pub ca_path: Option<String>,
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            grpc_port: 6533,
            http_port: 7218,
            host: "0.0.0.0".to_string(),
            max_request_size: 10 * 1024 * 1024, // 10MB
            request_timeout: 30,
            keep_alive: 60,
            tls: None,
        }
    }
}
