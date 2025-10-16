use smartticket_shared_config::AppConfig;
use smartticket_shared_error::Result;
use tracing::info;

pub struct CoreServer {
    config: AppConfig,
}

impl CoreServer {
    pub fn new(config: AppConfig) -> Self {
        Self { config }
    }

    pub async fn start(&self) -> Result<()> {
        info!("Starting Core server on port {}", self.config.server.grpc_port);
        // TODO: Implement core services
        Ok(())
    }
}