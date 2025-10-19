//! gRPC Client Management
//!
//! Manages connections to various gRPC services used by the HTTP gateway

use std::collections::HashMap;
use std::sync::Arc;
use tonic::transport::{Channel, Endpoint};
use tracing::{info, error, debug};
use anyhow::Result;

use crate::proto::smartticket_v1::{
    auth_service_client::AuthServiceClient,
    user_service_client::UserServiceClient,
    tenant_service_client::TenantServiceClient,
    ticket_service_client::TicketServiceClient,
    knowledge_service_client::KnowledgeServiceClient,
    sla_service_client::SlaServiceClient,
    role_permission_service_client::RolePermissionServiceClient,
};

/// gRPC client manager that maintains connections to all services
pub struct GrpcClientManager {
    auth_client: Option<AuthServiceClient<Channel>>,
    user_client: Option<UserServiceClient<Channel>>,
    tenant_client: Option<TenantServiceClient<Channel>>,
    ticket_client: Option<TicketServiceClient<Channel>>,
    knowledge_client: Option<KnowledgeServiceClient<Channel>>,
    sla_client: Option<SlaServiceClient<Channel>>,
    role_permission_client: Option<RolePermissionServiceClient<Channel>>,
    grpc_endpoint: String,
}

impl GrpcClientManager {
    /// Create a new gRPC client manager
    pub fn new(grpc_endpoint: String) -> Self {
        Self {
            auth_client: None,
            user_client: None,
            tenant_client: None,
            ticket_client: None,
            knowledge_client: None,
            sla_client: None,
            role_permission_client: None,
            grpc_endpoint,
        }
    }

    /// Initialize connections to all gRPC services
    pub async fn initialize(&mut self) -> Result<()> {
        info!("Initializing gRPC client connections to {}", self.grpc_endpoint);

        let endpoint = Endpoint::from_shared(self.grpc_endpoint.clone())?;

        // Connect to AuthService
        match endpoint.connect().await {
            Ok(channel) => {
                self.auth_client = Some(AuthServiceClient::new(channel));
                info!("Successfully connected to AuthService");
            }
            Err(e) => {
                error!("Failed to connect to AuthService: {}", e);
            }
        }

        // Connect to UserService
        match endpoint.connect().await {
            Ok(channel) => {
                self.user_client = Some(UserServiceClient::new(channel));
                info!("Successfully connected to UserService");
            }
            Err(e) => {
                error!("Failed to connect to UserService: {}", e);
            }
        }

        // Connect to TenantService
        match endpoint.connect().await {
            Ok(channel) => {
                self.tenant_client = Some(TenantServiceClient::new(channel));
                info!("Successfully connected to TenantService");
            }
            Err(e) => {
                error!("Failed to connect to TenantService: {}", e);
            }
        }

        // Connect to TicketService
        match endpoint.connect().await {
            Ok(channel) => {
                self.ticket_client = Some(TicketServiceClient::new(channel));
                info!("Successfully connected to TicketService");
            }
            Err(e) => {
                error!("Failed to connect to TicketService: {}", e);
            }
        }

        // Connect to KnowledgeService
        match endpoint.connect().await {
            Ok(channel) => {
                self.knowledge_client = Some(KnowledgeServiceClient::new(channel));
                info!("Successfully connected to KnowledgeService");
            }
            Err(e) => {
                error!("Failed to connect to KnowledgeService: {}", e);
            }
        }

        // Connect to SlaService
        match endpoint.connect().await {
            Ok(channel) => {
                self.sla_client = Some(SlaServiceClient::new(channel));
                info!("Successfully connected to SlaService");
            }
            Err(e) => {
                error!("Failed to connect to SlaService: {}", e);
            }
        }

        // Connect to RolePermissionService
        match endpoint.connect().await {
            Ok(channel) => {
                self.role_permission_client = Some(RolePermissionServiceClient::new(channel));
                info!("Successfully connected to RolePermissionService");
            }
            Err(e) => {
                error!("Failed to connect to RolePermissionService: {}", e);
            }
        }

        info!("gRPC client initialization completed");
        Ok(())
    }

    /// Get a reference to the AuthService client
    pub fn auth_client(&self) -> Option<&AuthServiceClient<Channel>> {
        self.auth_client.as_ref()
    }

    /// Get a reference to the UserService client
    pub fn user_client(&self) -> Option<&UserServiceClient<Channel>> {
        self.user_client.as_ref()
    }

    /// Get a reference to the TenantService client
    pub fn tenant_client(&self) -> Option<&TenantServiceClient<Channel>> {
        self.tenant_client.as_ref()
    }

    /// Get a reference to the TicketService client
    pub fn ticket_client(&self) -> Option<&TicketServiceClient<Channel>> {
        self.ticket_client.as_ref()
    }

    /// Get a reference to the KnowledgeService client
    pub fn knowledge_client(&self) -> Option<&KnowledgeServiceClient<Channel>> {
        self.knowledge_client.as_ref()
    }

    /// Get a reference to the SlaService client
    pub fn sla_client(&self) -> Option<&SlaServiceClient<Channel>> {
        self.sla_client.as_ref()
    }

    /// Get a reference to the RolePermissionService client
    pub fn role_permission_client(&self) -> Option<&RolePermissionServiceClient<Channel>> {
        self.role_permission_client.as_ref()
    }
}