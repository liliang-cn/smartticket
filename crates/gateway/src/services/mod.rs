//! HTTP Service Handlers
//!
//! This module contains HTTP handlers for all SmartTicket services.
//! Each handler translates HTTP requests to gRPC calls and back.

use axum::Router;
use crate::gateway::HttpToGrpcGateway;

/// Create all service routes and combine them into a single router
pub fn create_service_routes(_gateway: &HttpToGrpcGateway) -> Router {
    Router::new()
    // TODO: Add service routes when implemented
    // .nest("/auth/v1", auth_service::create_routes(gateway))
    // .nest("/v1", user_service::create_routes(gateway))
    // .nest("/v1", tenant_service::create_routes(gateway))
    // .nest("/v1", ticket_service::create_routes(gateway))
    // .nest("/v1", knowledge_service::create_routes(gateway))
    // .nest("/v1", sla_service::create_routes(gateway))
    // .nest("/v1", role_permission_service::create_routes(gateway))
}