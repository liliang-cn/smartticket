# SmartTicket API Documentation

This directory contains the automatically generated OpenAPI/Swagger documentation for the SmartTicket platform.

## 📁 Files Overview

- `openapi.yaml` - Complete OpenAPI 3.0 specification with all API endpoints
- `smartticket.v1.openapi.json` - Generated JSON OpenAPI specification from proto files
- `swagger-ui.html` - Interactive Swagger UI for API exploration
- `generate-openapi.sh` - Script to regenerate API documentation from proto files

## 🚀 Quick Start

### 1. View API Documentation

Open `swagger-ui.html` in your browser:
```bash
# Method 1: Open directly in browser
open api/swagger-ui.html

# Method 2: Start a simple HTTP server
cd api
python3 -m http.server 8080
# Then visit http://localhost:8080/swagger-ui.html
```

### 2. Interactive API Testing

The Swagger UI provides:
- ✅ Interactive API documentation
- ✅ "Try it out" functionality for testing endpoints
- ✅ Request/response examples
- ✅ Schema definitions
- ✅ Authentication support

### 3. Authentication Setup

For testing protected endpoints:

1. **Login first** using `/auth/login` endpoint:
   ```json
   {
     "email": "admin@test.smartticket.com",
     "password": "securePassword123",
     "tenant_domain": "test.smartticket.com",
     "remember_me": false
   }
   ```

2. **Tokens are automatically stored** in browser localStorage:
   - `access_token` - JWT access token
   - `user_id` - Current user ID
   - `tenant_id` - Current tenant ID

3. **All subsequent requests** will include proper authentication headers

## 📝 API Documentation Structure

### Authentication Endpoints
- `POST /auth/login` - User login
- `POST /auth/refresh` - Refresh access token

### Tenant Management
- `GET /tenants` - List tenants
- `POST /tenants` - Create new tenant
- `GET /tenants/{tenant_id}` - Get tenant details

### User Management
- `GET /users` - List users
- `POST /users` - Create new user
- `GET /users/{user_id}` - Get user details

## 🔧 Regenerating Documentation

To regenerate the API documentation from proto files:

```bash
# Make the script executable
chmod +x scripts/generate-openapi.sh

# Run the generation script
./scripts/generate-openapi.sh
```

### Manual Generation

If you prefer to generate manually:

```bash
# Generate OpenAPI from proto files
protoc \
    --openapiv2_out=api \
    --openapiv2_opt=logtostderr=true,json_names_for_fields=true,use_go_templates=true \
    -I. \
    proto/smartticket/api.proto
```

## 🏗️ From Proto to API

### How it works:

1. **Proto Definitions**: Service definitions in `proto/smartticket/`
2. **OpenAPI Generation**: `protoc-gen-openapiv2` converts proto to OpenAPI
3. **HTTP Routes**: RESTful paths added based on service methods
4. **Swagger UI**: Interactive documentation generated from OpenAPI spec

### Adding New APIs:

1. **Add to proto file**:
   ```protobuf
   rpc NewFeature(NewFeatureRequest) returns (NewFeatureResponse) {
     option (google.api.http) = {
       post: "/v1/new-feature"
       body: "*"
     };
   }
   ```

2. **Regenerate docs**:
   ```bash
   ./scripts/generate-openapi.sh
   ```

3. **Update OpenAPI spec** if needed for additional customization

## 🔒 Security Considerations

- **Authentication**: JWT Bearer tokens required for most endpoints
- **Multi-tenancy**: `X-Tenant-ID` header required for tenant isolation
- **HTTPS**: Production APIs should use HTTPS
- **Rate Limiting**: Implement appropriate rate limiting per tenant

## 📊 API Usage Examples

### Create a New User
```bash
curl -X POST "http://localhost:3001/v1/users" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "username": "newuser",
    "full_name": "New User",
    "password": "securePassword123",
    "role": "USER_ROLE_CUSTOMER_USER"
  }'
```

### List Tenants
```bash
curl -X GET "http://localhost:3001/v1/tenants?page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID"
```

## 🐛 Troubleshooting

### Common Issues:

1. **CORS Errors**: Ensure your HTTP gateway has proper CORS configuration
2. **Authentication Failures**: Check JWT token validity and expiration
3. **Missing Headers**: Ensure `X-Tenant-ID` and `Authorization` headers are included
4. **Proto Compilation**: Verify all proto imports and syntax are correct

### Debug Mode:

Enable debug output in the generation script:
```bash
export PROTOC_DEBUG=1
./scripts/generate-openapi.sh
```

## 📚 Additional Resources

- [OpenAPI Specification](https://swagger.io/specification/)
- [Swagger UI Documentation](https://swagger.io/tools/swagger-ui/)
- [gRPC-Gateway Documentation](https://grpc-ecosystem.github.io/grpc-gateway/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)

---

*This documentation is automatically generated from protobuf definitions. Keep your proto files up to date to ensure documentation accuracy.*