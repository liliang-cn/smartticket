# Database Seed Data

This directory contains tools and data for seeding the SmartTicket database with sample data for development and testing.

## Overview

The seed system provides:
- **Sample data generation** for development environments
- **Realistic test data** that covers all major features
- **Multi-tenant support** with proper data isolation
- **Extensible structure** for adding new seed data types

## File Structure

```
scripts/seed/
├── README.md                    # This documentation
├── seed_data.go                # Seed data generation and management
├── cmd/
│   └── seed/
│       └── main.go             # Seed command-line tool
└── data/
    ├── development.json        # Pre-generated development data
    ├── testing.json            # Pre-generated testing data
    └── fixtures.json           # Test fixtures for automated tests
```

## Seed Data Types

### Core Entities

1. **Tenants** - Multi-tenant organizations
   - Acme Corporation
   - Globex Inc
   - Stark Industries

2. **Users** - User accounts with different roles
   - Admin (System Administrator)
   - Engineers (John Smith, Sarah Jones)
   - Support Engineer (Mike Wilson)
   - Customer (Jane Doe)

3. **Tickets** - Sample support tickets
   - Bug reports
   - Feature requests
   - Technical issues
   - General inquiries

4. **Knowledge Articles** - Documentation and guides
   - How-to guides
   - Troubleshooting articles
   - Best practices

5. **Configuration** - System settings and LLM providers
   - Default settings
   - LLM provider configurations
   - Ticket categories and statuses

## Usage

### Command Line Tool

The seed command provides flexible options for managing seed data:

```bash
# Build the seed tool
go build -o seed ./scripts/seed/cmd/seed

# Generate seed data to file
./seed -output seed_data.json

# Seed database with generated data
./seed -config configs/config.dev.yaml

# Seed database from file
./seed -config configs/config.dev.yaml -load seed_data.json

# Clear and reseed database
./seed -config configs/config.dev.yaml -clear -force

# Use custom database path
./seed -config configs/config.dev.yaml -db ./data/test.db

# Verbose output
./seed -config configs/config.dev.yaml -verbose
```

### Makefile Integration

```bash
# Generate seed data file
make seed-generate

# Seed development database
make seed

# Reseed database (clear and seed)
make seed-reseed

# Seed test database
make seed-test
```

## Seed Data Examples

### Tenant Example

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme Corporation",
  "domain": "acme.example.com",
  "status": "active",
  "settings": "{\"timezone\": \"America/New_York\", \"language\": \"en\"}",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### User Example

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "admin@example.com",
  "name": "System Administrator",
  "role": "admin",
  "status": "active",
  "password": "$2a$10$...",  // bcrypt hash of "password123"
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Ticket Example

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440002",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "number": "TICKET-001",
  "title": "Unable to login to system",
  "description": "I've been trying to login for the past hour...",
  "status": "Open",
  "priority": "high",
  "severity": "medium",
  "category": "Technical Issue",
  "created_by": "550e8400-e29b-41d4-a716-446655440001",
  "created_at": "2024-01-15T08:30:00Z",
  "updated_at": "2024-01-15T08:30:00Z"
}
```

## Default Credentials

All seed users use the same default password for convenience in development:

```
Email: admin@example.com
Password: password123
```

## Customization

### Adding New Seed Data

1. **Add to data structures** in `seed_data.go`
   ```go
   type NewEntity struct {
       ID        string    `json:"id" gorm:"primaryKey"`
       Name      string    `json:"name" gorm:"not null"`
       CreatedAt time.Time `json:"created_at"`
       UpdatedAt time.Time `json:"updated_at"`
   }
   ```

2. **Add to SeedData struct**:
   ```go
   type SeedData struct {
       // ... existing fields
       NewEntities []NewEntity `json:"new_entities"`
   }
   ```

3. **Create generation function**:
   ```go
   func generateNewEntities(tenantID string) []NewEntity {
       // Generate sample data
   }
   ```

4. **Add to main generator**:
   ```go
   func GenerateSeedData() *SeedData {
       // ... existing code
       data.NewEntities = generateNewEntities(tenant.ID)
   }
   ```

5. **Add seeding function**:
   ```go
   func seedNewEntities(ctx context.Context, db *gorm.DB, entities []NewEntity) error {
       for _, entity := range entities {
           if err := db.Create(&entity).Error; err != nil {
               return err
           }
       }
       return nil
   }
   ```

### Modifying Sample Data

Edit the generation functions in `seed_data.go`:

```go
func generateTenants() []Tenant {
    return []Tenant{
        {
            Name: "Your Company",
            Domain: "yourcompany.example.com",
            // ... other fields
        },
        // Add more tenants
    }
}
```

### Environment-Specific Data

Create different seed files for different environments:

```bash
# Development data with realistic amounts
./seed -config configs/config.dev.yaml -output data/development.json

# Minimal testing data
./seed -config configs/config.test.yaml -output data/testing.json

# Production-like staging data
./seed -config configs/config.staging.yaml -output data/staging.json
```

## Best Practices

### Development

1. **Use consistent data** across environments
2. **Keep passwords simple** for development (change in production)
3. **Include edge cases** in test data
4. **Maintain referential integrity** in relationships

### Security

1. **Never use production credentials** in seed data
2. **Hash passwords** properly (bcrypt)
3. **Don't include sensitive information** in sample data
4. **Use different data** for production seeding

### Performance

1. **Batch insert** when possible
2. **Index foreign keys** properly
3. **Use transactions** for data consistency
4. **Clear old data** before reseeding

## Troubleshooting

### Common Issues

#### Database Connection Errors
```bash
# Check configuration file
./seed -config configs/config.dev.yaml -verbose

# Use custom database path
./seed -config configs/config.dev.yaml -db ./data/dev.db
```

#### Foreign Key Constraints
```bash
# Clear and reseed database
./seed -config configs/config.dev.yaml -clear -force
```

#### Permission Errors
```bash
# Check database file permissions
ls -la data/
chmod 644 data/*.db
```

### Debug Mode

Enable verbose logging to see detailed operations:

```bash
./seed -config configs/config.dev.yaml -verbose
```

### Validation

Validate seed data after generation:

```bash
# Check generated file
cat seed_data.json | jq '.tenants | length'

# Verify database seeding
sqlite3 data/smartticket_dev.db "SELECT COUNT(*) FROM tenants;"
```

## Integration with Tests

The seed data integrates with the testing framework:

```go
func TestTicketService(t *testing.T) {
    // Load test fixtures
    fixtures := fixtures.GetSampleTenants()

    // Use seed data in tests
    testDB := testutils.NewTestDatabase(t)
    defer testDB.Close()

    // Seed test database
    seedData := generateSeedData()
    err := SeedDatabase(testDB.GetDB(), seedData)
    require.NoError(t, err)
}
```

## Maintenance

### Updating Seed Data

When the data model changes:

1. **Update struct definitions** in `seed_data.go`
2. **Regenerate seed files** with updated tool
3. **Test seeding** in development environment
4. **Update documentation** with new fields

### Versioning

Keep track of seed data versions:

```bash
# Tag seed data versions
./seed -version 1.0.0 -output data/v1.0.0.json

# Load specific version
./seed -config configs/config.dev.yaml -load data/v1.0.0.json
```

### Cleanup

Clean up old seed data files:

```bash
# Remove old seed files
rm -f data/*.json.old

# Clean database
make clean-db
```

## Automation

### CI/CD Integration

Add seed data generation to CI pipeline:

```yaml
# .github/workflows/seed.yml
- name: Generate Seed Data
  run: |
    go build -o seed ./scripts/seed/cmd/seed
    ./seed -output data/ci-seed.json

- name: Seed Test Database
  run: |
    ./seed -config configs/config.test.yaml -load data/ci-seed.json
```

### Docker Integration

Include seed data in Docker image:

```dockerfile
COPY scripts/seed/ /app/scripts/seed/
RUN go build -o /app/seed ./scripts/seed/cmd/seed

# Generate seed data at container startup
CMD ["/app/seed", "-config", "configs/config.prod.yaml"]
```

This comprehensive seed system provides a solid foundation for development, testing, and deployment of the SmartTicket application.