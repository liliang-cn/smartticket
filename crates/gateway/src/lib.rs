pub mod auth_middleware;
pub mod auth_service;
pub mod grpc_service;
pub mod http_server;
pub mod knowledge_service;
pub mod role_permission_service;
pub mod server;
pub mod sla_service;
pub mod tenant_service;
pub mod user_service;

pub use auth_middleware::{
    extract_request_metadata, AuthMiddleware, JwtAuthInterceptor, PermissionCheck, RequestExt, TenantContext,
};
pub use auth_service::AuthGrpcService;
pub use grpc_service::TicketGrpcService;
pub use knowledge_service::KnowledgeGrpcService;
pub use role_permission_service::RolePermissionGrpcService;
pub use server::GatewayServer;
pub use tenant_service::TenantGrpcService;
pub use user_service::UserGrpcService;

// Include generated proto code directly
pub mod smartticket_v1 {
    include!("proto/smartticket.v1.rs");
}

// Include tests module
#[cfg(test)]
mod tests;
