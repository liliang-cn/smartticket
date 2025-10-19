use axum::{
    extract::State,
    http::StatusCode,
    response::{Html, Json},
};
use std::sync::Arc;
use serde_json::Value;
use crate::gateway::{GatewayConfig, openapi::OpenApiGenerator};

/// 简化的Swagger UI处理器
pub async fn swagger_ui_handler() -> Result<Html<String>, StatusCode> {
    let html = r#"
<!DOCTYPE html>
<html>
<head>
    <title>SmartTicket API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin: 0;
            background: #fafafa;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: true
            });
        };
    </script>
</body>
</html>
    "#;

    Ok(Html(html.to_string()))
}

/// OpenAPI JSON处理器
pub async fn openapi_json_handler(
    State((config, _gateway)): State<(GatewayConfig, Arc<crate::gateway::HttpToGrpcGateway>)>
) -> Result<Json<Value>, StatusCode> {
    let generator = OpenApiGenerator::new(config);
    let spec = generator.generate_spec();
    Ok(Json(serde_json::to_value(spec).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?))
}

/// OpenAPI YAML处理器
pub async fn openapi_yaml_handler(
    State((config, _gateway)): State<(GatewayConfig, Arc<crate::gateway::HttpToGrpcGateway>)>
) -> Result<String, StatusCode> {
    let generator = OpenApiGenerator::new(config);
    generator
        .export_yaml()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)
}