//! gRPC Service Implementation for SmartTicket
//!
//! This module implements the gRPC service handlers for ticket management.

use std::result::Result as StdResult;
use std::str::FromStr;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::{error, info, instrument};
use uuid::Uuid;

use crate::smartticket_v1::{
    ticket_service_server::TicketService, AddCommentRequest, AddCommentResponse,
    AssignTicketRequest, AssignTicketResponse, CreateTicketRequest, CreateTicketResponse,
    DeleteTicketRequest, DeleteTicketResponse, GetCommentsRequest, GetCommentsResponse,
    GetTicketRequest, GetTicketResponse, ListTicketsRequest, ListTicketsResponse,
    PaginationResponse, Response as ApiResponse, SearchTicketsRequest, SearchTicketsResponse,
    Ticket as GrpcTicket, TicketPriority as GrpcTicketPriority,
    TicketSeverity as GrpcTicketSeverity, TicketStatus as GrpcTicketStatus, UpdateTicketRequest,
    UpdateTicketResponse, UpdateTicketStatusRequest, UpdateTicketStatusResponse,
};
use crate::{PermissionCheck, RequestExt};
use smartticket_core::models::ticket::{
    CreateTicketRequest as CoreCreateTicketRequest, Ticket, TicketPriority, TicketStatus,
    TicketType, UpdateTicketRequest as CoreUpdateTicketRequest,
};
use smartticket_core::services::ticket_service::TicketService as CoreTicketService;
use smartticket_shared_database::{AuthUser, TenantContext};

/// gRPC Ticket Service implementation
pub struct TicketGrpcService {
    core_service: Arc<CoreTicketService>,
    #[allow(dead_code)]
    db_pool: Arc<sqlx::PgPool>,
}

impl TicketGrpcService {
    /// Create a new gRPC ticket service
    pub fn new(core_service: Arc<CoreTicketService>, db_pool: Arc<sqlx::PgPool>) -> Self {
        Self {
            core_service,
            db_pool,
        }
    }

    /// Create success response
    fn create_success_response(message: &str, request_id: &str) -> ApiResponse {
        ApiResponse {
            success: true,
            message: message.to_string(),
            data: None,
            errors: vec![],
            request_id: request_id.to_string(),
        }
    }

    /// Create error response
    fn create_error_response(message: &str, request_id: &str) -> ApiResponse {
        ApiResponse {
            success: false,
            message: message.to_string(),
            data: None,
            errors: vec![crate::smartticket_v1::Error {
                code: "VALIDATION_ERROR".to_string(),
                message: message.to_string(),
                details: None,
            }],
            request_id: request_id.to_string(),
        }
    }

    /// Create tenant context from authenticated user
    fn create_tenant_context(auth_user: &AuthUser) -> TenantContext {
        TenantContext {
            tenant_id: auth_user.tenant_id,
            user_id: auth_user.id,
            user_role: format!("{:?}", auth_user.role),
        }
    }

    /// Convert gRPC ticket priority to core ticket priority
    fn grpc_priority_to_core(priority: i32) -> smartticket_shared_error::Result<TicketPriority> {
        match GrpcTicketPriority::try_from(priority) {
            Ok(GrpcTicketPriority::Low) => Ok(TicketPriority::Low),
            Ok(GrpcTicketPriority::Normal) => Ok(TicketPriority::Normal),
            Ok(GrpcTicketPriority::High) => Ok(TicketPriority::High),
            Ok(GrpcTicketPriority::Critical) => Ok(TicketPriority::Critical),
            _ => Ok(TicketPriority::Normal), // Default to Normal for unspecified
        }
    }

    /// Convert gRPC ticket status to core ticket status
    fn grpc_status_to_core(status: i32) -> smartticket_shared_error::Result<TicketStatus> {
        match GrpcTicketStatus::try_from(status) {
            Ok(GrpcTicketStatus::New) => Ok(TicketStatus::Open),
            Ok(GrpcTicketStatus::Open) => Ok(TicketStatus::Open),
            Ok(GrpcTicketStatus::InProgress) => Ok(TicketStatus::InProgress),
            Ok(GrpcTicketStatus::PendingCustomer) => Ok(TicketStatus::PendingCustomer),
            Ok(GrpcTicketStatus::PendingThirdParty) => Ok(TicketStatus::PendingThirdParty),
            Ok(GrpcTicketStatus::Resolved) => Ok(TicketStatus::Resolved),
            Ok(GrpcTicketStatus::Closed) => Ok(TicketStatus::Closed),
            Ok(GrpcTicketStatus::Reopened) => Ok(TicketStatus::Reopened),
            _ => Ok(TicketStatus::Open), // Default to Open for unspecified
        }
    }

    /// Convert core ticket to gRPC ticket
    async fn core_ticket_to_grpc(&self, ticket: Ticket) -> GrpcTicket {
        GrpcTicket {
            id: ticket.id.to_string(),
            tenant_id: ticket.tenant_id.to_string(),
            ticket_number: format!("TICKET-{}", ticket.id.to_string()[..8].to_uppercase()),
            title: ticket.title,
            description: ticket.description,
            status: match ticket.status {
                TicketStatus::Open => GrpcTicketStatus::Open as i32,
                TicketStatus::InProgress => GrpcTicketStatus::InProgress as i32,
                TicketStatus::PendingCustomer => GrpcTicketStatus::PendingCustomer as i32,
                TicketStatus::PendingThirdParty => GrpcTicketStatus::PendingThirdParty as i32,
                TicketStatus::Resolved => GrpcTicketStatus::Resolved as i32,
                TicketStatus::Closed => GrpcTicketStatus::Closed as i32,
                TicketStatus::Reopened => GrpcTicketStatus::Reopened as i32,
                _ => GrpcTicketStatus::Unspecified as i32,
            },
            priority: match ticket.priority {
                TicketPriority::Low => GrpcTicketPriority::Low as i32,
                TicketPriority::Normal => GrpcTicketPriority::Normal as i32,
                TicketPriority::High => GrpcTicketPriority::High as i32,
                TicketPriority::Urgent => GrpcTicketPriority::High as i32, // Map Urgent to High
                TicketPriority::Critical => GrpcTicketPriority::Critical as i32,
                _ => GrpcTicketPriority::Unspecified as i32,
            },
            severity: GrpcTicketSeverity::Medium as i32, // Default severity
            category_id: ticket
                .category_id
                .map(|id| id.to_string())
                .unwrap_or_default(),
            contact_id: ticket.customer_id.to_string(),
            assigned_to_id: ticket
                .assigned_agent_id
                .map(|id| id.to_string())
                .unwrap_or_default(),
            created_by_id: ticket.created_by.clone().unwrap_or_default(),
            resolved_at: ticket.resolved_at.map(|dt| prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            }),
            closed_at: ticket.closed_at.map(|dt| prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            }),
            due_at: ticket.due_date.map(|dt| prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            }),
            resolution: ticket.resolution.unwrap_or_default(),
            tags: ticket.tags,
            created_at: Some(prost_types::Timestamp {
                seconds: ticket.created_at.timestamp(),
                nanos: ticket.created_at.timestamp_subsec_nanos() as i32,
            }),
            updated_at: Some(prost_types::Timestamp {
                seconds: ticket.updated_at.timestamp(),
                nanos: ticket.updated_at.timestamp_subsec_nanos() as i32,
            }),
            contact: {
                // Simplified contact info - don't fetch from database to avoid RLS issues
                Some(crate::smartticket_v1::User {
                    id: ticket.customer_id.to_string(),
                    tenant_id: ticket.tenant_id.to_string(),
                    email: "contact@example.com".to_string(),
                    username: "contact".to_string(),
                    full_name: "Contact User".to_string(),
                    role: 1, // Default role
                    is_active: true,
                    last_login_at: None,
                })
            },
            assigned_to: {
                // Simplified assigned agent info - don't fetch from database to avoid RLS issues
                if let Some(agent_id) = ticket.assigned_agent_id {
                    Some(crate::smartticket_v1::User {
                        id: agent_id.to_string(),
                        tenant_id: ticket.tenant_id.to_string(),
                        email: "agent@example.com".to_string(),
                        username: "agent".to_string(),
                        full_name: "Assigned Agent".to_string(),
                        role: 1, // Default role
                        is_active: true,
                        last_login_at: None,
                    })
                } else {
                    None
                }
            },
            created_by: {
                // Simplified creator info - don't fetch from database to avoid RLS issues
                if let Some(created_by_str) = &ticket.created_by {
                    if let Ok(created_by_uuid) = Uuid::parse_str(created_by_str) {
                        Some(crate::smartticket_v1::User {
                            id: created_by_uuid.to_string(),
                            tenant_id: ticket.tenant_id.to_string(),
                            email: "creator@example.com".to_string(),
                            username: "creator".to_string(),
                            full_name: "Ticket Creator".to_string(),
                            role: 1, // Default role
                            is_active: true,
                            last_login_at: None,
                        })
                    } else {
                        None
                    }
                } else {
                    None
                }
            },
            category: {
                // Simplified category info - don't fetch from database to avoid RLS issues
                if let Some(category_id) = ticket.category_id {
                    Some(crate::smartticket_v1::TicketCategory {
                        id: category_id.to_string(),
                        tenant_id: ticket.tenant_id.to_string(),
                        name: "Ticket Category".to_string(),
                        description: "Category description".to_string(),
                        parent_id: "".to_string(),
                        color: "#007bff".to_string(),
                        created_at: Some(prost_types::Timestamp {
                            seconds: 0,
                            nanos: 0,
                        }),
                        updated_at: Some(prost_types::Timestamp {
                            seconds: 0,
                            nanos: 0,
                        }),
                    })
                } else {
                    None
                }
            },
        }
    }
}

#[tonic::async_trait]
impl TicketService for TicketGrpcService {
    #[instrument(skip(self))]
    async fn create_ticket(
        &self,
        request: Request<CreateTicketRequest>,
    ) -> StdResult<Response<CreateTicketResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("ticket:create") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Creating ticket with title: {} by user: {}",
            req.title, auth_user.email
        );

        // Validate required fields
        if req.title.trim().is_empty() {
            let response = CreateTicketResponse {
                response: Some(Self::create_error_response(
                    "Title is required",
                    &request_id,
                )),
                ticket: None,
            };
            return Ok(Response::new(response));
        }

        // For customers, use their own ID as contact_id
        let contact_id = if req.contact_id.trim().is_empty() {
            if context.user_role.contains("Customer") {
                auth_user.id
            } else {
                let response = CreateTicketResponse {
                    response: Some(Self::create_error_response(
                        "Contact ID is required for non-customer users",
                        &request_id,
                    )),
                    ticket: None,
                };
                return Ok(Response::new(response));
            }
        } else {
            match Uuid::from_str(&req.contact_id) {
                Ok(id) => id,
                Err(_) => {
                    let response = CreateTicketResponse {
                        response: Some(Self::create_error_response(
                            "Invalid contact ID format",
                            &request_id,
                        )),
                        ticket: None,
                    };
                    return Ok(Response::new(response));
                }
            }
        };

        let priority = Self::grpc_priority_to_core(req.priority).unwrap_or(TicketPriority::Normal);

        let core_request = CoreCreateTicketRequest {
            tenant_id: context.tenant_id,
            customer_id: contact_id,
            title: req.title,
            description: req.description,
            priority,
            ticket_type: TicketType::Incident, // Default to Incident type
            category_id: if req.category_id.is_empty() {
                None
            } else {
                Uuid::from_str(&req.category_id).ok()
            },
            tags: req.tags,
            team_id: None,            // TODO: Extract from request if needed
            due_date: None,           // TODO: Convert from request if needed
            external_reference: None, // TODO: Extract from request if needed
        };

        // Create ticket using core service
        match self
            .core_service
            .create_ticket(core_request, &context, context.user_id.to_string())
            .await
        {
            Ok(ticket) => {
                info!("Successfully created ticket: {}", ticket.id);
                let grpc_ticket = self.core_ticket_to_grpc(ticket).await;
                let response = CreateTicketResponse {
                    response: Some(Self::create_success_response(
                        "Ticket created successfully",
                        &request_id,
                    )),
                    ticket: Some(grpc_ticket),
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to create ticket: {}", e);
                let response = CreateTicketResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    ticket: None,
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn get_ticket(
        &self,
        request: Request<GetTicketRequest>,
    ) -> StdResult<Response<GetTicketResponse>, Status> {
        // Check authentication and authorization
        let permission = if request.auth_user().map(|u| u.email).is_ok() {
            let auth_user = request.auth_user()?;
            let context = Self::create_tenant_context(&auth_user);
            if context.can_view_all_tickets() {
                "ticket:view"
            } else {
                "ticket:view_own"
            }
        } else {
            return Err(Status::unauthenticated("User not authenticated"));
        };

        if let Err(e) = request.check_permission(permission) {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Getting ticket: {} by user: {}",
            req.ticket_id, auth_user.email
        );

        // Parse ticket ID
        let ticket_id = match Uuid::from_str(&req.ticket_id) {
            Ok(id) => id,
            Err(_) => {
                let response = GetTicketResponse {
                    response: Some(Self::create_error_response(
                        "Invalid ticket ID format",
                        &request_id,
                    )),
                    ticket: None,
                    comments: vec![],
                };
                return Ok(Response::new(response));
            }
        };

        // Get ticket using core service
        match self.core_service.get_ticket(ticket_id, &context).await {
            Ok(ticket) => {
                info!("Successfully retrieved ticket: {}", ticket.id);
                let grpc_ticket = self.core_ticket_to_grpc(ticket).await;
                let response = GetTicketResponse {
                    response: Some(Self::create_success_response(
                        "Ticket retrieved successfully",
                        &request_id,
                    )),
                    ticket: Some(grpc_ticket),
                    comments: vec![], // TODO: Implement comments retrieval if requested
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to get ticket: {}", e);
                let response = GetTicketResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    ticket: None,
                    comments: vec![],
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn list_tickets(
        &self,
        request: Request<ListTicketsRequest>,
    ) -> StdResult<Response<ListTicketsResponse>, Status> {
        // Check authentication and authorization
        let permission = if request.auth_user().map(|u| u.email).is_ok() {
            let auth_user = request.auth_user()?;
            let context = Self::create_tenant_context(&auth_user);
            if context.can_view_all_tickets() {
                "ticket:view"
            } else {
                "ticket:view_own"
            }
        } else {
            return Err(Status::unauthenticated("User not authenticated"));
        };

        if let Err(e) = request.check_permission(permission) {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Listing tickets by user: {}", auth_user.email);

        // Parse pagination parameters
        let page_size = req.pagination.as_ref().map_or(20, |p| p.page_size);
        let page_token = req.pagination.as_ref().and_then(|p| {
            if p.page_token.is_empty() {
                None
            } else {
                Some(p.page_token.clone())
            }
        });

        let offset = page_token
            .as_ref()
            .and_then(|token| token.parse::<usize>().ok())
            .unwrap_or(0);

        // Build filter parameters
        let customer_id = if req.contact_id.is_empty() {
            None
        } else {
            Uuid::from_str(&req.contact_id).ok()
        };
        let assigned_agent_id = if req.assigned_to_id.is_empty() {
            None
        } else {
            Uuid::from_str(&req.assigned_to_id).ok()
        };
        let status = if req.statuses.is_empty() {
            None
        } else {
            req.statuses
                .first()
                .and_then(|s| Self::grpc_status_to_core(s.parse::<i32>().unwrap_or(0)).ok())
        };
        let priority = if req.priorities.is_empty() {
            None
        } else {
            req.priorities
                .first()
                .and_then(|p| Self::grpc_priority_to_core(p.parse::<i32>().unwrap_or(0)).ok())
        };

        // Build search filters
        let search_filters = smartticket_core::models::ticket::TicketSearchFilters {
            tenant_id: context.tenant_id,
            customer_id,
            assigned_agent_id,
            team_id: None,
            status,
            priority,
            ticket_type: None,
            category_id: None,
            created_after: None,
            created_before: None,
            updated_after: None,
            updated_before: None,
            search_query: None,
            tags: vec![],
            page_size: Some(page_size),
            page_token: page_token,
            order_by: Some("created_at".to_string()),
            order_desc: Some(true),
        };

        // Get tickets using core service
        match self
            .core_service
            .list_tickets(search_filters, &context)
            .await
        {
            Ok(ticket_response) => {
                info!(
                    "Successfully retrieved {} tickets",
                    ticket_response.tickets.len()
                );
                let mut grpc_tickets = Vec::new();
                for ticket in ticket_response.tickets {
                    grpc_tickets.push(self.core_ticket_to_grpc(ticket).await);
                }

                let response = ListTicketsResponse {
                    response: Some(Self::create_success_response(
                        "Tickets listed successfully",
                        &request_id,
                    )),
                    tickets: grpc_tickets,
                    pagination: Some(PaginationResponse {
                        total_count: ticket_response.total_count as i32,
                        page_size: req.pagination.as_ref().map_or(20, |p| p.page_size),
                        next_page_token: ticket_response.next_page_token.unwrap_or_default(),
                        prev_page_token: if offset > 0 {
                            (offset - page_size as usize).to_string()
                        } else {
                            String::new()
                        },
                    }),
                };

                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to list tickets: {}", e);
                let response = ListTicketsResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    tickets: vec![],
                    pagination: Some(PaginationResponse {
                        total_count: 0,
                        page_size: req.pagination.as_ref().map_or(20, |p| p.page_size),
                        next_page_token: String::new(),
                        prev_page_token: String::new(),
                    }),
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn update_ticket(
        &self,
        request: Request<UpdateTicketRequest>,
    ) -> StdResult<Response<UpdateTicketResponse>, Status> {
        // Check authentication and authorization
        let permission = if request.auth_user().map(|u| u.email).is_ok() {
            let auth_user = request.auth_user()?;
            let context = Self::create_tenant_context(&auth_user);
            if context.can_view_all_tickets() {
                "ticket:update"
            } else {
                "ticket:update_own"
            }
        } else {
            return Err(Status::unauthenticated("User not authenticated"));
        };

        if let Err(e) = request.check_permission(permission) {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Updating ticket: {} by user: {}",
            req.ticket_id, auth_user.email
        );

        // Parse ticket ID
        let ticket_id = match Uuid::from_str(&req.ticket_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateTicketResponse {
                    response: Some(Self::create_error_response(
                        "Invalid ticket ID format",
                        &request_id,
                    )),
                    ticket: None,
                };
                return Ok(Response::new(response));
            }
        };

        // For non-admin users, check if they can update this ticket
        if !context.can_view_all_tickets() {
            // TODO: Check if user owns this ticket or is assigned to it
            // For now, allow all authenticated users to update
        }

        // Build update request
        let update_request = CoreUpdateTicketRequest {
            id: ticket_id,
            tenant_id: context.tenant_id,
            title: if req.title.is_empty() {
                None
            } else {
                Some(req.title)
            },
            description: if req.description.is_empty() {
                None
            } else {
                Some(req.description)
            },
            priority: if req.priority == 0 {
                None
            } else {
                Self::grpc_priority_to_core(req.priority).ok()
            },
            category_id: if req.category_id.is_empty() {
                None
            } else {
                Uuid::from_str(&req.category_id).ok()
            },
            tags: None,               // Not available in UpdateTicketRequest
            due_date: None,           // Not available in UpdateTicketRequest
            external_reference: None, // Not available in UpdateTicketRequest
        };

        // Update ticket using core service
        match self
            .core_service
            .update_ticket(update_request, &context, context.user_id.to_string())
            .await
        {
            Ok(ticket) => {
                info!("Successfully updated ticket: {}", ticket.id);
                let grpc_ticket = self.core_ticket_to_grpc(ticket).await;
                let response = UpdateTicketResponse {
                    response: Some(Self::create_success_response(
                        "Ticket updated successfully",
                        &request_id,
                    )),
                    ticket: Some(grpc_ticket),
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to update ticket: {}", e);
                let response = UpdateTicketResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    ticket: None,
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn delete_ticket(
        &self,
        request: Request<DeleteTicketRequest>,
    ) -> StdResult<Response<DeleteTicketResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("ticket:delete") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Deleting ticket: {} by user: {}",
            req.ticket_id, auth_user.email
        );

        // Parse ticket ID
        let ticket_id = match Uuid::from_str(&req.ticket_id) {
            Ok(id) => id,
            Err(_) => {
                let response = DeleteTicketResponse {
                    response: Some(Self::create_error_response(
                        "Invalid ticket ID format",
                        &request_id,
                    )),
                };
                return Ok(Response::new(response));
            }
        };

        // Delete ticket using core service (soft delete)
        match self
            .core_service
            .delete_ticket(ticket_id, &context, context.user_id.to_string())
            .await
        {
            Ok(()) => {
                info!("Successfully deleted ticket: {}", ticket_id);
                let response = DeleteTicketResponse {
                    response: Some(Self::create_success_response(
                        "Ticket deleted successfully",
                        &request_id,
                    )),
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to delete ticket: {}", e);
                let response = DeleteTicketResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn update_ticket_status(
        &self,
        request: Request<UpdateTicketStatusRequest>,
    ) -> StdResult<Response<UpdateTicketStatusResponse>, Status> {
        // Check authentication and authorization
        let permission = if request.auth_user().map(|u| u.email).is_ok() {
            let auth_user = request.auth_user()?;
            let context = Self::create_tenant_context(&auth_user);
            if context.can_view_all_tickets() {
                "ticket:update"
            } else {
                "ticket:update_own"
            }
        } else {
            return Err(Status::unauthenticated("User not authenticated"));
        };

        if let Err(e) = request.check_permission(permission) {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Updating ticket status: {} -> {} by user: {}",
            req.ticket_id, req.status, auth_user.email
        );

        // Parse ticket ID
        let ticket_id = match Uuid::from_str(&req.ticket_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateTicketStatusResponse {
                    response: Some(Self::create_error_response(
                        "Invalid ticket ID format",
                        &request_id,
                    )),
                    ticket: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Convert gRPC status to core status
        let new_status = Self::grpc_status_to_core(req.status).map_err(|e| {
            error!("Failed to convert status: {}", e);
            Status::invalid_argument(format!("Invalid status: {}", e))
        })?;

        // Update ticket status using core service
        match self
            .core_service
            .change_ticket_status(
                ticket_id,
                new_status,
                if req.comment.is_empty() {
                    None
                } else {
                    Some(req.comment)
                },
                &context,
                context.user_id.to_string(),
            )
            .await
        {
            Ok(ticket) => {
                info!("Successfully updated ticket status: {}", ticket.id);
                let grpc_ticket = self.core_ticket_to_grpc(ticket).await;
                let response = UpdateTicketStatusResponse {
                    response: Some(Self::create_success_response(
                        "Ticket status updated successfully",
                        &request_id,
                    )),
                    ticket: Some(grpc_ticket),
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to update ticket status: {}", e);
                let response = UpdateTicketStatusResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    ticket: None,
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn assign_ticket(
        &self,
        request: Request<AssignTicketRequest>,
    ) -> StdResult<Response<AssignTicketResponse>, Status> {
        // Check authentication and authorization - only support staff can assign tickets
        if let Err(e) = request.check_permission("ticket:assign") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Assigning ticket: {} to {} by user: {}",
            req.ticket_id, req.assigned_to_id, auth_user.email
        );

        // Parse ticket ID
        let ticket_id = match Uuid::from_str(&req.ticket_id) {
            Ok(id) => id,
            Err(_) => {
                let response = AssignTicketResponse {
                    response: Some(Self::create_error_response(
                        "Invalid ticket ID format",
                        &request_id,
                    )),
                    ticket: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Parse assigned agent ID
        let agent_id = match Uuid::from_str(&req.assigned_to_id) {
            Ok(id) => id,
            Err(_) => {
                let response = AssignTicketResponse {
                    response: Some(Self::create_error_response(
                        "Invalid agent ID format",
                        &request_id,
                    )),
                    ticket: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Assign ticket using core service
        match self
            .core_service
            .assign_ticket(
                ticket_id,
                agent_id,
                None, // team_id - TODO: Extract from request if needed
                &context,
                context.user_id.to_string(),
            )
            .await
        {
            Ok(ticket) => {
                info!("Successfully assigned ticket: {}", ticket.id);
                let grpc_ticket = self.core_ticket_to_grpc(ticket).await;
                let response = AssignTicketResponse {
                    response: Some(Self::create_success_response(
                        "Ticket assigned successfully",
                        &request_id,
                    )),
                    ticket: Some(grpc_ticket),
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to assign ticket: {}", e);
                let response = AssignTicketResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    ticket: None,
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn add_comment(
        &self,
        request: Request<AddCommentRequest>,
    ) -> StdResult<Response<AddCommentResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("ticket:comment") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Adding comment to ticket: {} by user: {}",
            req.ticket_id, auth_user.email
        );

        // Parse ticket ID
        let ticket_id = match Uuid::from_str(&req.ticket_id) {
            Ok(id) => id,
            Err(_) => {
                let response = AddCommentResponse {
                    response: Some(Self::create_error_response(
                        "Invalid ticket ID format",
                        &request_id,
                    )),
                    comment: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Check if user can comment on this ticket
        let can_comment = match (auth_user.role, context.user_role.as_str()) {
            (smartticket_shared_database::UserRole::CustomerUser, _)
            | (smartticket_shared_database::UserRole::Sales, _) => {
                // Customers can comment on their own tickets
                let user_id = auth_user.id.to_string();
                let ticket_ids =
                    if let Ok(ticket) = self.core_service.get_ticket(ticket_id, &context).await {
                        vec![ticket.id.to_string()]
                    } else {
                        vec![]
                    };
                ticket_ids.contains(&user_id)
            }
            (smartticket_shared_database::UserRole::SupportEngineer, _)
            | (smartticket_shared_database::UserRole::TenantAdmin, _)
            | (smartticket_shared_database::UserRole::SuperAdmin, _) => true,
        };

        if !can_comment {
            let response = AddCommentResponse {
                response: Some(Self::create_error_response(
                    "Insufficient permissions to add comment",
                    &request_id,
                )),
                comment: None,
            };
            return Ok(Response::new(response));
        }

        // Parse author name and email (from auth user for now)
        let author_name = auth_user.full_name.clone();
        let author_email = auth_user.email.clone();
        let author_id = auth_user.id.to_string();

        // Use the is_internal flag directly from the request
        let is_internal = req.is_internal;
        let comment_type = if is_internal {
            smartticket_core::models::ticket::CommentType::Internal
        } else {
            smartticket_core::models::ticket::CommentType::Public
        };

        // Add comment using core service
        match self
            .core_service
            .add_comment(
                ticket_id,
                author_id,
                author_name,
                author_email,
                req.content,
                comment_type,
                is_internal,
                &context,
                auth_user.id.to_string(),
            )
            .await
        {
            Ok(comment) => {
                info!("Successfully added comment to ticket: {}", ticket_id);
                let grpc_comment = crate::smartticket_v1::TicketComment {
                    id: comment.id.to_string(),
                    ticket_id: comment.ticket_id.to_string(),
                    user_id: comment.author_id.to_string(),
                    content: comment.content,
                    is_internal,
                    created_at: Some(prost_types::Timestamp {
                        seconds: comment.created_at.timestamp(),
                        nanos: comment.created_at.timestamp_subsec_nanos() as i32,
                    }),
                    updated_at: Some(prost_types::Timestamp {
                        seconds: comment.created_at.timestamp(),
                        nanos: comment.created_at.timestamp_subsec_nanos() as i32,
                    }),
                    author: Some(crate::smartticket_v1::User {
                        id: comment.author_id.to_string(),
                        tenant_id: context.tenant_id.to_string(),
                        email: comment.author_email.clone(),
                        username: comment.author_email.clone(),
                        full_name: comment.author_name.clone(),
                        role: 1, // Default role
                        is_active: true,
                        last_login_at: None,
                    }),
                };

                let response = AddCommentResponse {
                    response: Some(Self::create_success_response(
                        "Comment added successfully",
                        &request_id,
                    )),
                    comment: Some(grpc_comment),
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to add comment: {}", e);
                let response = AddCommentResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    comment: None,
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn get_comments(
        &self,
        request: Request<GetCommentsRequest>,
    ) -> StdResult<Response<GetCommentsResponse>, Status> {
        // Check authentication and authorization
        let permission = if request.auth_user().map(|u| u.email).is_ok() {
            let auth_user = request.auth_user()?;
            let context = Self::create_tenant_context(&auth_user);
            if context.can_view_all_tickets() {
                "ticket:view"
            } else {
                "ticket:view_own"
            }
        } else {
            return Err(Status::unauthenticated("User not authenticated"));
        };

        if let Err(e) = request.check_permission(permission) {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Getting comments for ticket: {} by user: {}",
            req.ticket_id, auth_user.email
        );

        // Parse ticket ID
        let ticket_id = match Uuid::from_str(&req.ticket_id) {
            Ok(id) => id,
            Err(_) => {
                let response = GetCommentsResponse {
                    response: Some(Self::create_error_response(
                        "Invalid ticket ID format",
                        &request_id,
                    )),
                    comments: vec![],
                    pagination: Some(PaginationResponse {
                        total_count: 0,
                        page_size: 10,
                        next_page_token: "".to_string(),
                        prev_page_token: "".to_string(),
                    }),
                };
                return Ok(Response::new(response));
            }
        };

        // Check if user can view comments on this ticket
        let can_view_comments = match (auth_user.role, context.user_role.as_str()) {
            (smartticket_shared_database::UserRole::CustomerUser, _)
            | (smartticket_shared_database::UserRole::Sales, _) => {
                // Customers can view comments on their own tickets
                let user_id = auth_user.id.to_string();
                let ticket_ids =
                    if let Ok(ticket) = self.core_service.get_ticket(ticket_id, &context).await {
                        vec![ticket.id.to_string()]
                    } else {
                        vec![]
                    };
                ticket_ids.contains(&user_id)
            }
            (smartticket_shared_database::UserRole::SupportEngineer, _)
            | (smartticket_shared_database::UserRole::TenantAdmin, _)
            | (smartticket_shared_database::UserRole::SuperAdmin, _) => true,
        };

        if !can_view_comments {
            let response = GetCommentsResponse {
                response: Some(Self::create_error_response(
                    "Insufficient permissions to view comments",
                    &request_id,
                )),
                comments: vec![],
                pagination: Some(PaginationResponse {
                    total_count: 0,
                    page_size: 10,
                    next_page_token: "".to_string(),
                    prev_page_token: "".to_string(),
                }),
            };
            return Ok(Response::new(response));
        }

        // Parse pagination parameters
        let page_size = req.pagination.as_ref().map_or(20, |p| p.page_size);
        let page_token = req.pagination.as_ref().and_then(|p| {
            if p.page_token.is_empty() {
                None
            } else {
                Some(p.page_token.clone())
            }
        });

        let offset = page_token
            .as_ref()
            .and_then(|token| token.parse::<usize>().ok())
            .unwrap_or(0);

        // Get comments using core service
        match self
            .core_service
            .get_comments(ticket_id, &context, Some(page_size as i32), page_token)
            .await
        {
            Ok(comments) => {
                info!(
                    "Successfully retrieved {} comments for ticket: {}",
                    comments.len(),
                    ticket_id
                );
                let comments_count = comments.len();
                let grpc_comments: Vec<crate::smartticket_v1::TicketComment> = comments
                    .into_iter()
                    .map(|comment| {
                        crate::smartticket_v1::TicketComment {
                            id: comment.id.to_string(),
                            ticket_id: comment.ticket_id.to_string(),
                            user_id: comment.author_id.to_string(),
                            content: comment.content,
                            is_internal: comment.is_internal,
                            created_at: Some(prost_types::Timestamp {
                                seconds: comment.created_at.timestamp(),
                                nanos: comment.created_at.timestamp_subsec_nanos() as i32,
                            }),
                            updated_at: Some(prost_types::Timestamp {
                                seconds: comment.created_at.timestamp(),
                                nanos: comment.created_at.timestamp_subsec_nanos() as i32,
                            }),
                            author: Some(crate::smartticket_v1::User {
                                id: comment.author_id.to_string(),
                                tenant_id: context.tenant_id.to_string(),
                                email: comment.author_email.clone(),
                                username: comment.author_email.clone(),
                                full_name: comment.author_name.clone(),
                                role: 1, // Default role
                                is_active: true,
                                last_login_at: None,
                            }),
                        }
                    })
                    .collect();

                let next_page_token = if (offset + grpc_comments.len()) >= page_size as usize {
                    (offset + page_size as usize).to_string()
                } else {
                    String::new()
                };

                let response = GetCommentsResponse {
                    response: Some(Self::create_success_response(
                        "Comments retrieved successfully",
                        &request_id,
                    )),
                    comments: grpc_comments,
                    pagination: Some(PaginationResponse {
                        total_count: comments_count as i32,
                        page_size: req.pagination.as_ref().map_or(20, |p| p.page_size),
                        next_page_token,
                        prev_page_token: if offset > 0 {
                            (offset - page_size as usize).to_string()
                        } else {
                            String::new()
                        },
                    }),
                };

                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to get comments: {}", e);
                let response = GetCommentsResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    comments: vec![],
                    pagination: Some(PaginationResponse {
                        total_count: 0,
                        page_size: req.pagination.as_ref().map_or(20, |p| p.page_size),
                        next_page_token: String::new(),
                        prev_page_token: String::new(),
                    }),
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn search_tickets(
        &self,
        request: Request<SearchTicketsRequest>,
    ) -> StdResult<Response<SearchTicketsResponse>, Status> {
        // Check authentication and authorization
        let permission = if request.auth_user().map(|u| u.email).is_ok() {
            let auth_user = request.auth_user()?;
            let context = Self::create_tenant_context(&auth_user);
            if context.can_view_all_tickets() {
                "ticket:view"
            } else {
                "ticket:view_own"
            }
        } else {
            return Err(Status::unauthenticated("User not authenticated"));
        };

        if let Err(e) = request.check_permission(permission) {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Searching tickets by user: {} with query: {}",
            auth_user.email, req.query
        );

        // Parse pagination parameters
        let page_size = req.pagination.as_ref().map_or(20, |p| p.page_size);
        let page_token = req.pagination.as_ref().and_then(|p| {
            if p.page_token.is_empty() {
                None
            } else {
                Some(p.page_token.clone())
            }
        });

        let offset = page_token
            .as_ref()
            .and_then(|token| token.parse::<usize>().ok())
            .unwrap_or(0);

        // Build search filters
        let mut filters = Vec::new();
        // Status and category_id are now in the filters field
        for filter in &req.filters {
            // Check if filter has a field and values (meaningful filter)
            if !filter.field.is_empty() && !filter.values.is_empty() {
                filters.push(filter.clone());
            }
        }

        // Convert FilterRequest to Vec<String> for core service
        let filter_strings: Vec<String> = filters.iter().map(|f| f.field.clone()).collect();

        // Search tickets using core service
        match self
            .core_service
            .search_tickets(
                context.tenant_id,
                &req.query,
                filter_strings,
                Some(page_size as i32),
                page_token,
                Some("created_at".to_string()), // default sort by created_at
                Some(true),                     // default descending order
            )
            .await
        {
            Ok(tickets) => {
                info!(
                    "Successfully found {} tickets matching query",
                    tickets.len()
                );
                let total_count = tickets.len() as i32;
                let mut grpc_tickets = Vec::new();
                for ticket in tickets {
                    grpc_tickets.push(self.core_ticket_to_grpc(ticket).await);
                }

                let next_page_token = (offset + grpc_tickets.len()) >= page_size as usize;

                let response = SearchTicketsResponse {
                    response: Some(Self::create_success_response(
                        "Tickets searched successfully",
                        &request_id,
                    )),
                    tickets: grpc_tickets,
                    pagination: Some(PaginationResponse {
                        total_count: total_count,
                        page_size: req.pagination.as_ref().map_or(20, |p| p.page_size),
                        next_page_token: if next_page_token {
                            (offset + page_size as usize).to_string()
                        } else {
                            String::new()
                        },
                        prev_page_token: if offset > 0 {
                            (offset - page_size as usize).to_string()
                        } else {
                            String::new()
                        },
                    }),
                    total_matches: total_count,
                };

                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to search tickets: {}", e);
                let response = SearchTicketsResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    tickets: vec![],
                    pagination: Some(PaginationResponse {
                        total_count: 0,
                        page_size: req.pagination.as_ref().map_or(20, |p| p.page_size),
                        next_page_token: String::new(),
                        prev_page_token: String::new(),
                    }),
                    total_matches: 0,
                };
                Ok(Response::new(response))
            }
        }
    }

    #[instrument(skip(self))]
    async fn get_ticket_statistics(
        &self,
        request: Request<crate::smartticket_v1::GetTicketStatisticsRequest>,
    ) -> StdResult<Response<crate::smartticket_v1::GetTicketStatisticsResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("ticket:view") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let context = Self::create_tenant_context(&auth_user);
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Getting ticket statistics by user: {}", auth_user.email);

        // Get ticket statistics using core service
        match self
            .core_service
            .get_ticket_stats(&context)
            .await
        {
            Ok(stats) => {
                info!("Successfully retrieved ticket statistics");
                let statistics = crate::smartticket_v1::TicketStatistics {
                    total_tickets: stats.total_tickets as i32,
                    open_tickets: stats.open_tickets as i32,
                    in_progress_tickets: stats.in_progress_tickets as i32,
                    resolved_tickets: stats.resolved_tickets as i32,
                    closed_tickets: stats.closed_tickets as i32,
                    overdue_tickets: stats.overdue_tickets as i32,
                    average_resolution_time_hours: stats.average_resolution_time_minutes.unwrap_or(0.0) / 60.0,
                    average_response_time_hours: stats.average_response_time_minutes.unwrap_or(0.0) / 60.0,
                };
                let response = crate::smartticket_v1::GetTicketStatisticsResponse {
                    response: Some(Self::create_success_response(
                        "Ticket statistics retrieved successfully",
                        &request_id,
                    )),
                    statistics: Some(statistics),
                };
                Ok(Response::new(response))
            }
            Err(e) => {
                error!("Failed to get ticket statistics: {}", e);
                let statistics = crate::smartticket_v1::TicketStatistics {
                    total_tickets: 0,
                    open_tickets: 0,
                    in_progress_tickets: 0,
                    resolved_tickets: 0,
                    closed_tickets: 0,
                    overdue_tickets: 0,
                    average_resolution_time_hours: 0.0,
                    average_response_time_hours: 0.0,
                };
                let response = crate::smartticket_v1::GetTicketStatisticsResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    statistics: Some(statistics),
                };
                Ok(Response::new(response))
            }
        }
    }
}
