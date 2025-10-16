use smartticket_shared_error::Result;
use tracing::info;

pub struct NotificationService;

impl NotificationService {
    pub fn new() -> Self {
        Self
    }

    pub async fn start(&self) -> Result<()> {
        info!("Starting Notification service");
        // TODO: Implement notification service
        Ok(())
    }
}
