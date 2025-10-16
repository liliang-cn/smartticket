use smartticket_shared_error::Result;
use tracing::info;

pub struct PlatformService;

impl PlatformService {
    pub fn new() -> Self {
        Self
    }

    pub async fn start(&self) -> Result<()> {
        info!("Starting Platform service");
        // TODO: Implement platform service
        Ok(())
    }
}
