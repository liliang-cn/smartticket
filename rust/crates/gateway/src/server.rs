use smartticket_core::services::ticket_service::TicketService;
use smartticket_shared_config::AppConfig;
use smartticket_shared_database::AuthService;
use smartticket_shared_error::Result;
use std::net::SocketAddr;
use std::sync::Arc;
use tonic::transport::Server;
use tracing::{error, info};

use crate::auth_service::AuthGrpcService;
use crate::auth_middleware::JwtAuthInterceptor;
use crate::grpc_service::TicketGrpcService;
use crate::knowledge_service::KnowledgeGrpcService;
use crate::role_permission_service::RolePermissionGrpcService;
use crate::sla_service::SlaGrpcService;
use crate::smartticket_v1::{
    auth_service_server::AuthServiceServer,
    knowledge_service_server::KnowledgeServiceServer,
    role_permission_service_server::RolePermissionServiceServer,
    sla_service_server::SlaServiceServer,
    tenant_service_server::TenantServiceServer, ticket_service_server::TicketServiceServer,
    user_service_server::UserServiceServer,
};
use crate::tenant_service::TenantGrpcService;
use crate::user_service::UserGrpcService;

pub struct GatewayServer {
    config: AppConfig,
    ticket_service: Arc<TicketService>,
    auth_service: Arc<AuthService>,
    db_pool: Arc<sqlx::PgPool>,
}

impl GatewayServer {
    pub fn new(
        config: AppConfig,
        ticket_service: Arc<TicketService>,
        auth_service: Arc<AuthService>,
        db_pool: Arc<sqlx::PgPool>,
    ) -> Self {
        Self {
            config,
            ticket_service,
            auth_service,
            db_pool,
        }
    }

    pub async fn start(&self) -> Result<()> {
        info!(
            "Starting Gateway server on gRPC port: {}",
            self.config.server.grpc_port
        );

        let addr = SocketAddr::from(([127, 0, 0, 1], self.config.server.grpc_port));

        // Create gRPC services
        let ticket_grpc_service =
            TicketGrpcService::new(self.ticket_service.clone(), self.db_pool.clone());
        let user_grpc_service =
            UserGrpcService::new(self.auth_service.clone(), self.db_pool.clone());
        let knowledge_grpc_service = KnowledgeGrpcService::new(self.db_pool.clone());
        let tenant_grpc_service =
            TenantGrpcService::new(self.auth_service.clone(), self.db_pool.clone());
        let role_permission_grpc_service =
            RolePermissionGrpcService::new(self.auth_service.clone(), self.db_pool.clone());
        let sla_grpc_service = SlaGrpcService::new(self.db_pool.clone());
        let auth_grpc_service =
            AuthGrpcService::new(self.auth_service.clone(), self.db_pool.clone());

        info!("Registering gRPC services:");
        info!("  - AuthService (Login, RefreshToken, etc.)");
        info!("  - TicketService (CreateTicket, GetTicket, UpdateTicket, etc.)");
        info!("  - UserService (CreateUser, GetUser, UpdateUser, etc.)");
        info!("  - KnowledgeService (CreateArticle, GetArticle, SearchArticles, etc.)");
        info!("  - TenantService (CreateTenant, GetTenant, UpdateTenant, etc.)");
        info!("  - RolePermissionService (CreateRole, AssignPermissions, etc.)");
        info!("  - SlaService (CreateSlaPolicy, GetSlaPolicy, UpdateSlaPolicy, etc.)");

        // Create JWT authentication interceptor
        info!("Creating JWT authentication interceptor...");
        let jwt_interceptor = JwtAuthInterceptor::new(self.auth_service.clone());
        info!("JWT interceptor created successfully");

        // Build gRPC server with authentication interceptor
        let grpc_server = Server::builder()
            // AuthService doesn't need authentication interceptor (for login)
            .add_service(AuthServiceServer::new(auth_grpc_service))
            // Apply JWT interceptor to all other services
            .add_service(
                TicketServiceServer::with_interceptor(ticket_grpc_service, jwt_interceptor.clone())
            )
            .add_service(
                UserServiceServer::with_interceptor(user_grpc_service, jwt_interceptor.clone())
            )
            .add_service(
                KnowledgeServiceServer::with_interceptor(knowledge_grpc_service, jwt_interceptor.clone())
            )
            .add_service(
                TenantServiceServer::with_interceptor(tenant_grpc_service, jwt_interceptor.clone())
            )
            .add_service(
                RolePermissionServiceServer::with_interceptor(
                    role_permission_grpc_service,
                    jwt_interceptor.clone(),
                )
            )
            .add_service(
                SlaServiceServer::with_interceptor(sla_grpc_service, jwt_interceptor)
            );

        // Start server
        match grpc_server.serve(addr).await {
            Ok(_) => {
                info!("Gateway server stopped gracefully");
                Ok(())
            }
            Err(e) => {
                error!("Gateway server failed: {}", e);
                Err(smartticket_shared_error::SmartTicketError::Internal(
                    e.to_string(),
                ))
            }
        }
    }
}
