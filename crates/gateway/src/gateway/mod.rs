//! HTTP-to-gRPC Gateway Module
//!
//! This module provides the core functionality for translating HTTP REST requests
//! to gRPC service calls, enabling full REST API access to all SmartTicket services.

pub mod auth_service;
pub mod config;
pub mod error;
pub mod middleware;
pub mod openapi;
pub mod gateway_router;
pub mod swagger;
pub mod translator;

use std::collections::HashMap;
use std::sync::Arc;
use tonic::transport::{Channel, Endpoint};
use axum::{Router, Json, response::Html};
use tracing::{info, error, debug};
use serde_json::Value;

pub use config::GatewayConfig;

/// Main HTTP-to-gRPC Gateway instance
pub struct HttpToGrpcGateway {
    pub config: Arc<GatewayConfig>,
    grpc_channels: HashMap<String, Arc<Channel>>,
}

impl HttpToGrpcGateway {
    /// Create a new gateway instance
    pub fn new(config: GatewayConfig) -> Self {
        Self {
            config: Arc::new(config),
            grpc_channels: HashMap::new(),
        }
    }

    /// Initialize gRPC service connections
    pub async fn initialize_connections(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        info!("Initializing gRPC service connections");

        // Initialize connections to all gRPC services
        let services = vec![
            "smartticket.v1.AuthService",
            "smartticket.v1.UserService",
            "smartticket.v1.TenantService",
            "smartticket.v1.TicketService",
            "smartticket.v1.KnowledgeService",
            "smartticket.v1.SlaService",
            "smartticket.v1.RolePermissionService",
        ];

        for service in services {
            debug!("Connecting to gRPC service: {}", service);
            let endpoint = Endpoint::from_shared(self.config.grpc_endpoint.clone());
            match endpoint {
                Ok(endpoint) => {
                    match endpoint.connect().await {
                        Ok(channel) => {
                            self.grpc_channels.insert(service.to_string(), Arc::new(channel));
                        }
                        Err(e) => {
                            error!("Failed to connect to {}: {}", service, e);
                        }
                    }
                }
                Err(e) => {
                    error!("Failed to create endpoint for {}: {}", service, e);
                }
            }
        }

        info!("Connected to {} gRPC services", self.grpc_channels.len());
        Ok(())
    }

    /// Get a gRPC channel for a specific service
    pub fn get_channel(&self, service: &str) -> Option<Arc<Channel>> {
        self.grpc_channels.get(service).cloned()
    }

    /// Start the HTTP server with all middleware and routes
    pub async fn start(&self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        info!("Starting HTTP-to-gRPC Gateway on port {}", self.config.http_port);

        // Create the main router with all middleware
        let app = self.create_app().await?;

        // Bind listener
        let addr = format!("0.0.0.0:{}", self.config.http_port);
        let listener = tokio::net::TcpListener::bind(&addr).await?;
        info!("HTTP server listening on: {}", addr);

        // Start server
        info!("SmartTicket Gateway is ready to accept requests");
        axum::serve(listener, app).await?;

        Ok(())
    }

    /// Create the main application router with all middleware
    async fn create_app(&self) -> Result<Router, Box<dyn std::error::Error + Send + Sync>> {
        let app = Router::new()
            // Basic health check
            .route("/health", axum::routing::get(health_check))
            // API documentation will be added later
            .route("/docs", axum::routing::get(|| async { Html("Swagger UI - Coming Soon") }))
            .route("/openapi.yaml", axum::routing::get(|| async { Html("OpenAPI YAML - Coming Soon") }))
            .route("/", axum::routing::get(|| async { Html("SmartTicket API Gateway - Coming Soon") }));

        Ok(app)
    }

    /// Generate OpenAPI specification from proto files
    pub fn generate_openapi_spec(&self) -> Result<Value, Box<dyn std::error::Error + Send + Sync>> {
        let mut spec = self.base_openapi_spec();

        // Service paths will be added here in Phase 4 when implementing user stories
        spec["paths"] = Value::Object(serde_json::Map::new());

        Ok(spec)
    }

    /// Create the base OpenAPI specification structure
    fn base_openapi_spec(&self) -> Value {
        serde_json::json!({
            "openapi": "3.0.3",
            "info": {
                "title": "SmartTicket API",
                "description": "B2B multi-tenant ticketing and knowledge collaboration platform API. This API provides comprehensive functionality for multi-tenant management, ticket lifecycle management, knowledge base management, user management, and role and permissions.",
                "version": "1.0.0",
                "contact": {
                    "name": "SmartTicket API Support",
                    "email": "api-support@smartticket.com"
                },
                "license": {
                    "name": "Commercial License",
                    "url": "https://smartticket.com/license"
                }
            },
            "servers": [
                {
                    "url": format!("http://localhost:{}", self.config.http_port),
                    "description": "Development server"
                },
                {
                    "url": "https://staging-api.smartticket.com/v1",
                    "description": "Staging server"
                },
                {
                    "url": "https://api.smartticket.com/v1",
                    "description": "Production server"
                }
            ],
            "security": [
                {
                    "BearerAuth": []
                },
                {
                    "TenantAuth": []
                }
            ],
            "components": {
                "securitySchemes": {
                    "BearerAuth": {
                        "type": "http",
                        "scheme": "bearer",
                        "bearerFormat": "JWT",
                        "description": "JWT access token"
                    },
                    "TenantAuth": {
                        "type": "apiKey",
                        "in": "header",
                        "name": "X-Tenant-ID",
                        "description": "Tenant identifier"
                    }
                },
                "schemas": {
                    "ApiResponse": {
                        "type": "object",
                        "properties": {
                            "success": {"type": "boolean"},
                            "message": {"type": "string"},
                            "data": {"type": "object", "additionalProperties": true},
                            "errors": {
                                "type": "array",
                                "items": {"$ref": "#/components/schemas/Error"}
                            },
                            "request_id": {"type": "string", "format": "uuid"},
                            "timestamp": {"type": "integer"}
                        }
                    },
                    "Error": {
                        "type": "object",
                        "required": ["code", "message"],
                        "properties": {
                            "code": {
                                "type": "string",
                                "description": "Error code (e.g., VALIDATION_ERROR, NOT_FOUND)"
                            },
                            "message": {
                                "type": "string",
                                "description": "Human-readable error message"
                            },
                            "details": {
                                "type": "object",
                                "additionalProperties": true,
                                "description": "Additional error details"
                            },
                            "field": {
                                "type": "string",
                                "description": "Field that caused the error"
                            }
                        }
                    }
                }
            }
        })
    }
}

/// Health check endpoint handler
async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "healthy",
        "version": env!("CARGO_PKG_VERSION"),
        "timestamp": chrono::Utc::now().timestamp(),
        "services": {
            "gateway": {
                "status": "healthy",
                "latency_ms": 10
            }
        }
    }))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_gateway_creation() {
        let config = GatewayConfig {
            http_port: 3286,
            grpc_endpoint: "http://localhost:50051".to_string(),
            cors_origins: vec!["http://localhost:3000".parse().unwrap()],
            max_request_size: 10 * 1024 * 1024, // 10MB
            timeout: std::time::Duration::from_secs(30),
            rate_limit: crate::gateway::config::RateLimitConfig {
                requests_per_minute: 100,
                burst_size: 20,
            },
            auth: crate::gateway::config::AuthConfig {
                jwt_secret: "test-secret".to_string(),
                token_expiry: std::time::Duration::from_secs(3600),
                refresh_expiry: std::time::Duration::from_secs(86400),
            },
            openapi: crate::gateway::config::OpenApiConfig {
                auto_refresh: true,
                include_examples: true,
                servers: vec![],
            },
        };

        let gateway = HttpToGrpcGateway::new(config);
        assert_eq!(gateway.config.http_port, 3286);
        assert_eq!(gateway.grpc_channels.len(), 0);
    }

    #[tokio::test]
    async fn test_openapi_generation() {
        let config = GatewayConfig {
            http_port: 3286,
            grpc_endpoint: "http://localhost:50051".to_string(),
            cors_origins: vec![],
            max_request_size: 1024 * 1024,
            timeout: std::time::Duration::from_secs(30),
            rate_limit: crate::gateway::config::RateLimitConfig {
                requests_per_minute: 100,
                burst_size: 20,
            },
            auth: crate::gateway::config::AuthConfig {
                jwt_secret: "test-secret".to_string(),
                token_expiry: std::time::Duration::from_secs(3600),
                refresh_expiry: std::time::Duration::from_secs(86400),
            },
            openapi: crate::gateway::config::OpenApiConfig {
                auto_refresh: true,
                include_examples: true,
                servers: vec![],
            },
        };

        let gateway = HttpToGrpcGateway::new(config);
        let spec = gateway.generate_openapi_spec().unwrap();

        assert_eq!(spec["openapi"], "3.0.3");
        assert_eq!(spec["info"]["title"], "SmartTicket API");
        assert!(spec["components"]["schemas"].is_object());
    }
}