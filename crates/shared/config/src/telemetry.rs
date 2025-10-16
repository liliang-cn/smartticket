use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TelemetryConfig {
    pub tracing: TracingConfig,
    pub metrics: MetricsConfig,
    pub jaeger: Option<JaegerConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TracingConfig {
    pub enabled: bool,
    pub level: String,
    pub format: LogFormat,
    pub output: LogOutput,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum LogFormat {
    Json,
    Compact,
    Pretty,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum LogOutput {
    Stdout,
    Stderr,
    File(String),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsConfig {
    pub enabled: bool,
    pub port: u16,
    pub path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JaegerConfig {
    pub endpoint: String,
    pub port: u16,
    pub service_name: String,
}

impl Default for TelemetryConfig {
    fn default() -> Self {
        Self {
            tracing: TracingConfig {
                enabled: true,
                level: "info".to_string(),
                format: LogFormat::Json,
                output: LogOutput::Stdout,
            },
            metrics: MetricsConfig {
                enabled: true,
                port: 9090,
                path: "/metrics".to_string(),
            },
            jaeger: None,
        }
    }
}
