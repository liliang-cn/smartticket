use lazy_static::lazy_static;
use prometheus::{
    register_counter_vec, register_gauge_vec, register_histogram_vec, CounterVec, Encoder,
    GaugeVec, HistogramVec, TextEncoder,
};
use std::time::{Duration, Instant};
use tracing::info;

lazy_static! {
    // HTTP request metrics
    static ref HTTP_REQUESTS_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_http_requests_total",
        "Total number of HTTP requests",
        &["method", "endpoint", "status_code"]
    ).unwrap();

    static ref HTTP_REQUEST_DURATION: HistogramVec = register_histogram_vec!(
        "smartticket_http_request_duration_seconds",
        "HTTP request duration in seconds",
        &["method", "endpoint"],
        vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
    ).unwrap();

    // Database metrics
    static ref DB_CONNECTIONS_ACTIVE: GaugeVec = register_gauge_vec!(
        "smartticket_db_connections_active",
        "Number of active database connections",
        &["pool"]
    ).unwrap();

    static ref DB_QUERIES_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_db_queries_total",
        "Total number of database queries",
        &["pool", "query_type", "status"]
    ).unwrap();

    static ref DB_QUERY_DURATION: HistogramVec = register_histogram_vec!(
        "smartticket_db_query_duration_seconds",
        "Database query duration in seconds",
        &["pool", "query_type"],
        vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
    ).unwrap();

    // gRPC metrics
    static ref GRPC_REQUESTS_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_grpc_requests_total",
        "Total number of gRPC requests",
        &["service", "method", "status_code"]
    ).unwrap();

    static ref GRPC_REQUEST_DURATION: HistogramVec = register_histogram_vec!(
        "smartticket_grpc_request_duration_seconds",
        "gRPC request duration in seconds",
        &["service", "method"],
        vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
    ).unwrap();

    // Business metrics
    static ref TICKETS_CREATED_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_tickets_created_total",
        "Total number of tickets created",
        &["tenant_id", "priority", "severity"]
    ).unwrap();

    static ref TICKETS_RESOLVED_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_tickets_resolved_total",
        "Total number of tickets resolved",
        &["tenant_id", "priority", "severity"]
    ).unwrap();

    static ref KNOWLEDGE_ARTICLES_VIEWED_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_knowledge_articles_viewed_total",
        "Total number of knowledge article views",
        &["tenant_id", "article_id"]
    ).unwrap();

    // Authentication metrics
    static ref AUTH_TOKENS_ISSUED_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_auth_tokens_issued_total",
        "Total number of auth tokens issued",
        &["tenant_id", "user_role"]
    ).unwrap();

    static ref AUTH_LOGIN_ATTEMPTS_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_auth_login_attempts_total",
        "Total number of login attempts",
        &["tenant_id", "status"]
    ).unwrap();

    // SLA metrics
    static ref SLA_BREACHES_TOTAL: CounterVec = register_counter_vec!(
        "smartticket_sla_breaches_total",
        "Total number of SLA breaches",
        &["tenant_id", "sla_type", "priority", "severity"]
    ).unwrap();

    static ref SLA_COMPLIANCE_RATE: GaugeVec = register_gauge_vec!(
        "smartticket_sla_compliance_rate",
        "SLA compliance rate percentage",
        &["tenant_id", "sla_type", "priority"]
    ).unwrap();
}

pub struct MetricsService;

impl MetricsService {
    /// Record HTTP request
    pub fn record_http_request(method: &str, endpoint: &str, status_code: u16, duration: Duration) {
        let method_str = method.to_uppercase();
        let status_str = status_code.to_string();

        HTTP_REQUESTS_TOTAL
            .with_label_values(&[&method_str, endpoint, &status_str])
            .inc();

        HTTP_REQUEST_DURATION
            .with_label_values(&[&method_str, endpoint])
            .observe(duration.as_secs_f64());
    }

    /// Record gRPC request
    pub fn record_grpc_request(service: &str, method: &str, status_code: &str, duration: Duration) {
        GRPC_REQUESTS_TOTAL
            .with_label_values(&[service, method, status_code])
            .inc();

        GRPC_REQUEST_DURATION
            .with_label_values(&[service, method])
            .observe(duration.as_secs_f64());
    }

    /// Record database query
    pub fn record_db_query(pool: &str, query_type: &str, status: &str, duration: Duration) {
        DB_QUERIES_TOTAL
            .with_label_values(&[pool, query_type, status])
            .inc();

        DB_QUERY_DURATION
            .with_label_values(&[pool, query_type])
            .observe(duration.as_secs_f64());
    }

    /// Update active database connections
    pub fn set_db_connections_active(pool: &str, count: f64) {
        DB_CONNECTIONS_ACTIVE.with_label_values(&[pool]).set(count);
    }

    /// Record ticket creation
    pub fn record_ticket_created(tenant_id: &str, priority: &str, severity: &str) {
        TICKETS_CREATED_TOTAL
            .with_label_values(&[tenant_id, priority, severity])
            .inc();
    }

    /// Record ticket resolution
    pub fn record_ticket_resolved(tenant_id: &str, priority: &str, severity: &str) {
        TICKETS_RESOLVED_TOTAL
            .with_label_values(&[tenant_id, priority, severity])
            .inc();
    }

    /// Record knowledge article view
    pub fn record_knowledge_article_viewed(tenant_id: &str, article_id: &str) {
        KNOWLEDGE_ARTICLES_VIEWED_TOTAL
            .with_label_values(&[tenant_id, article_id])
            .inc();
    }

    /// Record auth token issuance
    pub fn record_auth_token_issued(tenant_id: &str, user_role: &str) {
        AUTH_TOKENS_ISSUED_TOTAL
            .with_label_values(&[tenant_id, user_role])
            .inc();
    }

    /// Record login attempt
    pub fn record_auth_login_attempt(tenant_id: &str, status: &str) {
        AUTH_LOGIN_ATTEMPTS_TOTAL
            .with_label_values(&[tenant_id, status])
            .inc();
    }

    /// Record SLA breach
    pub fn record_sla_breach(tenant_id: &str, sla_type: &str, priority: &str, severity: &str) {
        SLA_BREACHES_TOTAL
            .with_label_values(&[tenant_id, sla_type, priority, severity])
            .inc();
    }

    /// Update SLA compliance rate
    pub fn set_sla_compliance_rate(tenant_id: &str, sla_type: &str, priority: &str, rate: f64) {
        SLA_COMPLIANCE_RATE
            .with_label_values(&[tenant_id, sla_type, priority])
            .set(rate);
    }

    /// Export metrics in Prometheus format
    pub fn export_metrics() -> Result<String, Box<dyn std::error::Error>> {
        let encoder = TextEncoder::new();
        let metric_families = prometheus::gather();
        let mut buffer = Vec::new();
        encoder.encode(&metric_families, &mut buffer)?;
        Ok(String::from_utf8(buffer)?)
    }
}

/// RAII-style timer for measuring operation duration
pub struct Timer {
    start_time: Instant,
    operation_type: String,
    labels: Vec<String>,
}

impl Timer {
    pub fn new(operation_type: &str, labels: Vec<String>) -> Self {
        Self {
            start_time: Instant::now(),
            operation_type: operation_type.to_string(),
            labels,
        }
    }

    pub fn duration(&self) -> Duration {
        self.start_time.elapsed()
    }

    pub fn finish(self) {
        let duration = self.duration();
        info!(
            operation_type = self.operation_type,
            duration_ms = duration.as_millis(),
            "Operation completed"
        );

        // Record the appropriate metric based on operation type
        match self.operation_type.as_str() {
            "http_request" => {
                if self.labels.len() >= 2 {
                    HTTP_REQUEST_DURATION
                        .with_label_values(&[&self.labels[0], &self.labels[1]])
                        .observe(duration.as_secs_f64());
                }
            }
            "grpc_request" => {
                if self.labels.len() >= 2 {
                    GRPC_REQUEST_DURATION
                        .with_label_values(&[&self.labels[0], &self.labels[1]])
                        .observe(duration.as_secs_f64());
                }
            }
            "db_query" => {
                if self.labels.len() >= 2 {
                    DB_QUERY_DURATION
                        .with_label_values(&[&self.labels[0], &self.labels[1]])
                        .observe(duration.as_secs_f64());
                }
            }
            _ => {
                // Generic timing
                info!("Generic operation timer completed");
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn test_timer() {
        let timer = Timer::new(
            "test_operation",
            vec!["label1".to_string(), "label2".to_string()],
        );

        // Simulate some work
        std::thread::sleep(Duration::from_millis(10));

        let duration = timer.duration();
        assert!(duration.as_millis() >= 10);

        timer.finish();
    }

    #[test]
    fn test_metrics_export() {
        // Record some test metrics
        MetricsService::record_http_request("GET", "/api/v1/test", 200, Duration::from_millis(100));
        MetricsService::record_grpc_request(
            "TestService",
            "TestMethod",
            "OK",
            Duration::from_millis(50),
        );

        // Export metrics
        let metrics_output = MetricsService::export_metrics().unwrap();

        // Verify metrics are included in output
        assert!(metrics_output.contains("smartticket_http_requests_total"));
        assert!(metrics_output.contains("smartticket_grpc_requests_total"));
    }

    #[test]
    fn test_business_metrics() {
        MetricsService::record_ticket_created("tenant-123", "high", "critical");
        MetricsService::record_ticket_resolved("tenant-123", "high", "critical");
        MetricsService::record_knowledge_article_viewed("tenant-123", "article-456");
        MetricsService::record_auth_token_issued("tenant-123", "admin");
        MetricsService::record_auth_login_attempt("tenant-123", "success");
        MetricsService::record_sla_breach("tenant-123", "response", "high", "critical");
        MetricsService::set_sla_compliance_rate("tenant-123", "response", "high", 95.5);

        // These should not panic and should increment the metrics
        let metrics_output = MetricsService::export_metrics().unwrap();
        assert!(metrics_output.contains("smartticket_tickets_created_total"));
        assert!(metrics_output.contains("smartticket_tickets_resolved_total"));
    }
}
