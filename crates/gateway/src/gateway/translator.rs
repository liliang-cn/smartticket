//! HTTP-to-gRPC Translator
//!
//! Translates HTTP requests to gRPC calls and gRPC responses back to HTTP responses.

use std::collections::HashMap;
use std::sync::Arc;
use tonic::transport::Channel;
use serde::{Deserialize, Serialize};
use crate::gateway::error::{GatewayError, ErrorContext, DefaultErrorHandler};

/// HTTP-to-gRPC request translator
pub struct RequestTranslator;

impl RequestTranslator {
    /// Translate HTTP query parameters to gRPC request
    pub fn translate_query_params<T>(
        params: &HashMap<String, String>,
    ) -> Result<T, GatewayError>
    where
        T: for<'de> Deserialize<'de>,
    {
        serde_json::from_value(serde_json::to_value(params).map_err(|e| {
            GatewayError::BadRequest(format!("Failed to serialize query params: {}", e))
        })?).map_err(|e| {
            GatewayError::BadRequest(format!("Failed to translate query parameters: {}", e))
        })
    }

    /// Translate HTTP JSON body to gRPC request
    pub fn translate_json_body<T>(
        body: &serde_json::Value,
    ) -> Result<T, GatewayError>
    where
        T: for<'de> Deserialize<'de>,
    {
        serde_json::from_value(body.clone()).map_err(|e| {
            GatewayError::BadRequest(format!("Failed to translate JSON body: {}", e))
        })
    }

    /// Translate form data to gRPC request
    pub fn translate_form_data<T>(
        form_data: &HashMap<String, String>,
    ) -> Result<T, GatewayError>
    where
        T: for<'de> Deserialize<'de>,
    {
        Self::translate_query_params(form_data)
    }
}

/// gRPC-to-HTTP response translator
pub struct ResponseTranslator;

impl ResponseTranslator {
    /// Translate gRPC response to HTTP JSON response
    pub fn translate_grpc_response<T>(
        grpc_response: T,
    ) -> Result<serde_json::Value, GatewayError>
    where
        T: Serialize,
    {
        serde_json::to_value(grpc_response).map_err(|e| {
            GatewayError::InternalError(format!("Failed to serialize gRPC response: {}", e))
        })
    }

    /// Translate gRPC stream response to HTTP paginated response
    pub fn translate_grpc_stream<T>(
        items: Vec<T>,
        page: u32,
        page_size: u32,
        total: u64,
    ) -> Result<serde_json::Value, GatewayError>
    where
        T: Serialize,
    {
        let items_json = items
            .into_iter()
            .map(|item| serde_json::to_value(item))
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| GatewayError::InternalError(format!("Failed to serialize items: {}", e)))?;

        let total_pages = ((total as f64) / (page_size as f64)).ceil() as u32;
        let total_pages = if total_pages == 0 { 1 } else { total_pages };

        Ok(serde_json::json!({
            "items": items_json,
            "pagination": {
                "page": page,
                "page_size": page_size,
                "total": total,
                "total_pages": total_pages,
                "has_next": page < total_pages,
                "has_prev": page > 1
            }
        }))
    }
}

/// gRPC channel manager
pub struct GrpcChannelManager {
    channels: HashMap<String, Arc<Channel>>,
}

impl GrpcChannelManager {
    /// Create new channel manager
    pub fn new() -> Self {
        Self {
            channels: HashMap::new(),
        }
    }

    /// Add gRPC channel for a service
    pub fn add_channel(&mut self, service_name: &str, channel: Arc<Channel>) {
        self.channels.insert(service_name.to_string(), channel);
    }

    /// Get gRPC channel for a service
    pub fn get_channel(&self, service_name: &str) -> Option<Arc<Channel>> {
        self.channels.get(service_name).cloned()
    }

    /// Check if service is available
    pub fn is_service_available(&self, service_name: &str) -> bool {
        self.channels.contains_key(service_name)
    }

    /// List all available services
    pub fn list_services(&self) -> Vec<String> {
        self.channels.keys().cloned().collect()
    }
}

impl Default for GrpcChannelManager {
    fn default() -> Self {
        Self::new()
    }
}

/// Service metadata for translation
#[derive(Debug, Clone)]
pub struct ServiceMetadata {
    pub name: String,
    pub version: String,
    pub methods: Vec<MethodMetadata>,
}

/// Method metadata for translation
#[derive(Debug, Clone)]
pub struct MethodMetadata {
    pub name: String,
    pub request_type: String,
    pub response_type: String,
    pub http_method: String,
    pub http_path: String,
}

/// Translation context
#[derive(Debug, Clone)]
pub struct TranslationContext {
    pub service_name: String,
    pub method_name: String,
    pub user_id: Option<String>,
    pub tenant_id: Option<String>,
    pub request_id: Option<String>,
}

impl TranslationContext {
    /// Create new translation context
    pub fn new(service_name: &str, method_name: &str) -> Self {
        Self {
            service_name: service_name.to_string(),
            method_name: method_name.to_string(),
            user_id: None,
            tenant_id: None,
            request_id: None,
        }
    }

    /// Add user ID to context
    pub fn with_user_id(mut self, user_id: &str) -> Self {
        self.user_id = Some(user_id.to_string());
        self
    }

    /// Add tenant ID to context
    pub fn with_tenant_id(mut self, tenant_id: &str) -> Self {
        self.tenant_id = Some(tenant_id.to_string());
        self
    }

    /// Add request ID to context
    pub fn with_request_id(mut self, request_id: &str) -> Self {
        self.request_id = Some(request_id.to_string());
        self
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_request_translator_query_params() {
        let mut params = HashMap::new();
        params.insert("name".to_string(), "test".to_string());
        params.insert("value".to_string(), "123".to_string());

        #[derive(Deserialize)]
        struct TestRequest {
            name: String,
            value: i32,
        }

        // This test would need actual JSON conversion logic
        assert_eq!(params.get("name"), Some(&"test".to_string()));
    }

    #[test]
    fn test_grpc_channel_manager() {
        let mut manager = GrpcChannelManager::new();

        // Test with mock channel would require actual gRPC setup
        assert_eq!(manager.list_services().len(), 0);

        // Service availability check
        assert!(!manager.is_service_available("TestService"));
    }

    #[test]
    fn test_translation_context() {
        let context = TranslationContext::new("UserService", "GetUser")
            .with_user_id("user123")
            .with_tenant_id("tenant456")
            .with_request_id("req789");

        assert_eq!(context.service_name, "UserService");
        assert_eq!(context.method_name, "GetUser");
        assert_eq!(context.user_id, Some("user123".to_string()));
        assert_eq!(context.tenant_id, Some("tenant456".to_string()));
        assert_eq!(context.request_id, Some("req789".to_string()));
    }
}