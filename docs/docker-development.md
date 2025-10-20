# Docker Development Guide

This guide covers Docker development setup for SmartTicket.

## Quick Start

### Development Environment
```bash
# Start development environment with hot reload
docker-compose -f deployments/docker-compose.dev.yml up -d

# View logs
docker-compose -f deployments/docker-compose.dev.yml logs -f smartticket-dev

# Stop development environment
docker-compose -f deployments/docker-compose.dev.yml down
```

### Production Environment
```bash
# Build and start production environment
docker-compose -f deployments/docker-compose.yml up -d

# View logs
docker-compose -f deployments/docker-compose.yml logs -f

# Stop production environment
docker-compose -f deployments/docker-compose.yml down
```

### Testing Environment
```bash
# Run tests in Docker
docker-compose -f deployments/docker-compose.test.yml up --build

# View test results
docker-compose -f deployments/docker-compose.test.yml logs smartticket-test

# Clean up test environment
docker-compose -f deployments/docker-compose.test.yml down -v
```

## Available Services

### Development Environment
- **smartticket-dev**: Main application on port 6533 with hot reload
- **redis-dev**: Redis cache on port 6379
- **adminer**: Database admin tool on port 8080
- **mailhog**: Email testing tool on ports 1025 (SMTP) and 8025 (Web UI)

### Production Environment
- **smartticket**: Main application on port 6533
- **redis**: Redis cache on port 6379
- **nginx**: Reverse proxy on ports 80 and 443

## Configuration

### Environment Variables
Create a `.env` file in the project root:

```bash
# Application
PORT=6533
ENVIRONMENT=development
LOG_LEVEL=debug

# Database
DB_PATH=/app/data/smartticket.db

# JWT
JWT_SECRET=your-super-secret-jwt-key-here

# Frontend URL (for CORS)
FRONTEND_URL=http://localhost:3000

# Email (optional)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
FROM_EMAIL=noreply@smartticket.local
```

### Production Environment Variables
For production, use stronger secrets and configuration:

```bash
# Production settings
ENVIRONMENT=production
LOG_LEVEL=info
GIN_MODE=release

# Strong JWT secret
JWT_SECRET=your-super-strong-production-jwt-secret-key

# Production database path
DB_PATH=/app/data/smartticket.db

# CORS settings
FRONTEND_URL=https://yourdomain.com

# SSL certificates (place in deployments/nginx/ssl/)
SSL_CERT_PATH=/etc/nginx/ssl/cert.pem
SSL_KEY_PATH=/etc/nginx/ssl/key.pem
```

## Development Workflow

### 1. Hot Reload Development
The development environment uses `air` for hot reload. Any changes to Go files will automatically rebuild and restart the application.

### 2. Database Management
- Development database: `./data/smartticket_dev.db`
- Test database: `./data/smartticket_test.db`
- Production database: `/app/data/smartticket.db` (in container)

### 3. Redis Development
- Development Redis: `localhost:6379`
- Data persists in Docker volume `redis-dev-data`

### 4. Email Testing
Use MailHog for email testing during development:
- Web interface: http://localhost:8025
- SMTP server: localhost:1025

## Building Images

### Build Production Image
```bash
# Build production image
docker build -t smartticket:latest .

# Build with custom tag
docker build -t smartticket:v1.0.0 .

# Build for specific platform
docker build --platform linux/amd64 -t smartticket:latest .
```

### Build Development Image
```bash
# Build development image
docker build -f Dockerfile.dev -t smartticket:dev .
```

## Docker Compose Files

### `docker-compose.yml`
Production environment with:
- Optimized multi-stage build
- Non-root user
- Health checks
- Nginx reverse proxy
- Redis cache

### `docker-compose.dev.yml`
Development environment with:
- Hot reload support
- Debug enabled
- Development tools
- Additional services (Adminer, MailHog)

### `docker-compose.test.yml`
Testing environment with:
- Minimal dependencies
- Test database
- CI/CD optimized

## Performance Optimization

### Production Optimizations
- Multi-stage builds reduce image size
- Non-root user for security
- Health checks for monitoring
- Nginx reverse proxy with gzip
- Redis for caching

### Development Optimizations
- Hot reload for faster development
- Air for automatic rebuilds
- Debug logging enabled
- Development tools included

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 6533, 6379, 8080, 8025 are available
2. **Permission issues**: Make sure `data/` directory is writable
3. **Docker build failures**: Check network connection and registry availability
4. **Hot reload not working**: Verify `.air.toml` configuration

### Debug Commands

```bash
# Check container status
docker-compose ps

# View container logs
docker-compose logs smartticket-dev

# Execute commands in container
docker-compose exec smartticket-dev sh

# Check resource usage
docker stats

# Clean up unused containers and images
docker system prune -f
```

## Security Considerations

1. **Don't use development configuration in production**
2. **Use strong JWT secrets in production**
3. **Enable HTTPS in production**
4. **Regularly update base images**
5. **Use non-root users in production containers**
6. **Limit container resource usage**

## Deployment

### Local Production Deployment
```bash
# Create production environment file
cp .env.example .env.prod

# Edit .env.prod with production values

# Start production services
docker-compose -f deployments/docker-compose.yml --env-file .env.prod up -d
```

### Backup and Restore
```bash
# Backup data directory
docker run --rm -v smartticket-data:/data -v $(pwd)/backups:/backup alpine tar czf /backup/smartticket-backup-$(date +%Y%m%d).tar.gz -C /data .

# Restore from backup
docker run --rm -v smartticket-data:/data -v $(pwd)/backups:/backup alpine tar xzf /backup/smartticket-backup-20241201.tar.gz -C /data
```