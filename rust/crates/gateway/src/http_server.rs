use axum::{
    http::{header, StatusCode},
    response::{Html, IntoResponse},
    routing::get,
    Router,
};
use tower::ServiceBuilder;
use tower_http::{cors::CorsLayer, trace::TraceLayer};
use tracing::info;

pub struct HttpServer;

impl HttpServer {
    pub fn create_app() -> Router {
        Router::new()
            .route("/docs", get(swagger_ui_handler))
            .route("/openapi.yaml", get(openapi_yaml_handler))
            .route("/swagger-ui.html", get(swagger_ui_handler))
            .route("/", get(root_handler))
            .layer(
                ServiceBuilder::new()
                    .layer(TraceLayer::new_for_http())
                    .layer(CorsLayer::permissive()),
            )
    }

    pub async fn start(port: u16) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let app = Self::create_app();

        let addr = format!("0.0.0.0:{}", port);
        info!("🌐 HTTP Server starting on http://{}", addr);
        info!("📚 API Documentation available at http://{}/docs", addr);

        let listener = tokio::net::TcpListener::bind(addr).await?;
        axum::serve(listener, app).await?;

        Ok(())
    }
}

async fn swagger_ui_handler() -> impl IntoResponse {
    let swagger_ui_html = include_str!("../static/swagger-ui.html");
    Html(swagger_ui_html)
}

async fn openapi_yaml_handler() -> impl IntoResponse {
    let openapi_yaml = include_str!("../static/openapi.yaml");
    (
        StatusCode::OK,
        [(header::CONTENT_TYPE, "application/x-yaml")],
        openapi_yaml,
    )
}

async fn root_handler() -> impl IntoResponse {
    let html = r#"
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SmartTicket API Gateway</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .container { text-align: center; }
        .links { margin: 30px 0; }
        .links a { display: inline-block; margin: 10px; padding: 15px 25px; background: #007acc; color: white; text-decoration: none; border-radius: 5px; }
        .links a:hover { background: #005a9e; }
        .status { color: #28a745; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 SmartTicket API Gateway</h1>
        <p class="status">✅ Service is running</p>

        <div class="links">
            <a href="/docs">📚 API Documentation</a>
            <a href="/openapi.yaml">📄 OpenAPI Specification</a>
        </div>

        <h2>Available Services</h2>
        <ul style="text-align: left; display: inline-block;">
            <li><strong>gRPC Services</strong>: localhost:50051</li>
            <li><strong>TicketService</strong>: Ticket lifecycle management</li>
            <li><strong>UserService</strong>: User management and authentication</li>
            <li><strong>KnowledgeService</strong>: Knowledge base management</li>
            <li><strong>TenantService</strong>: Multi-tenant management</li>
            <li><strong>RolePermissionService</strong>: Role and permissions</li>
        </ul>
    </div>
</body>
</html>
    "#;

    Html(html)
}
