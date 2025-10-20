# Contributing to SmartTicket

Thank you for your interest in contributing to SmartTicket! This guide will help you get started with contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)
- [Code Review](#code-review)
- [Community](#community)

## Getting Started

### Prerequisites

- Go 1.25+ installed
- Docker and Docker Compose (optional)
- Git configured
- Development environment set up (see [Development Setup Guide](docs/development-setup.md))

### First-Time Setup

1. **Fork the repository**
   ```bash
   # Fork the repository on GitHub, then clone your fork
   git clone https://github.com/your-username/smartticket.git
   cd smartticket
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/company/smartticket.git
   ```

3. **Set up development environment**
   ```bash
   make env-setup
   cp .env.example .env
   # Edit .env as needed
   ```

4. **Verify setup**
   ```bash
   make test
   make lint
   make build
   ```

## Development Workflow

### 1. Create a Branch

```bash
# Sync with main branch
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Changes

- Write code following the [Code Standards](#code-standards)
- Add tests for new functionality
- Ensure all tests pass: `make test`
- Run linting: `make lint`
- Format code: `make fmt`

### 3. Test Your Changes

```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-e2e

# Run with coverage
make test-cover

# Run pre-commit checks
make pre-commit
```

### 4. Commit Your Changes

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```bash
# Feature
git commit -m "feat: add user authentication with JWT tokens"

# Bug fix
git commit -m "fix: resolve memory leak in ticket processing"

# Documentation
git commit -m "docs: update API documentation for v2.0"

# Refactoring
git commit -m "refactor: simplify database connection handling"

# Performance
git commit -m "perf: improve query performance with indexing"

# Tests
git commit -m "test: add integration tests for ticket lifecycle"
```

### 5. Push and Create Pull Request

```bash
# Push to your fork
git push origin feature/your-feature-name

# Create pull request on GitHub
# Use descriptive title and fill out the template
```

## Code Standards

### Go Style Guidelines

1. **Formatting**: Use `gofmt` and `goimports`
   ```bash
   make fmt
   ```

2. **Naming**: Follow Go conventions
   - Use `camelCase` for variables and functions
   - Use `PascalCase` for exported types and functions
   - Use `ALL_CAPS` for constants
   - Be descriptive and concise

3. **Comments**: Document public code
   ```go
   // NewUserService creates a new user service with the given repository
   func NewUserService(repo UserRepository) *UserService {
       // ...
   }
   ```

4. **Error Handling**: Always handle errors explicitly
   ```go
   result, err := someFunction()
   if err != nil {
       return nil, fmt.Errorf("failed to do something: %w", err)
   }
   ```

### Architecture Guidelines

1. **Clean Architecture**: Follow the established layers
   - **Domain**: Business entities and rules
   - **Application**: Use cases and services
   - **Infrastructure**: External dependencies
   - **Interface**: HTTP handlers and middleware

2. **Dependency Injection**: Use constructor injection
   ```go
   type TicketService struct {
       repo   TicketRepository
       logger Logger
   }

   func NewTicketService(repo TicketRepository, logger Logger) *TicketService {
       return &TicketService{
           repo:   repo,
           logger: logger,
       }
   }
   ```

3. **Interface Segregation**: Keep interfaces small and focused
   ```go
   type UserRepository interface {
       Create(ctx context.Context, user *User) error
       GetByID(ctx context.Context, id string) (*User, error)
       // Don't add unrelated methods
   }
   ```

### Security Guidelines

1. **Input Validation**: Always validate user input
2. **SQL Injection**: Use parameterized queries (GORM handles this)
3. **Authentication**: Use JWT tokens with proper expiration
4. **Authorization**: Check permissions for all operations
5. **Sensitive Data**: Never log passwords, tokens, or PII

### Performance Guidelines

1. **Database**: Use appropriate indexes and query optimization
2. **Memory**: Be mindful of memory allocation in loops
3. **Concurrency**: Use goroutines and channels appropriately
4. **Caching**: Implement caching for frequently accessed data

## Testing Guidelines

### Test Structure

```go
func TestUserService_Create(t *testing.T) {
    // Setup
    tests := []struct {
        name    string
        user    *User
        wantErr bool
        errType error
    }{
        {
            name: "valid user",
            user: &User{
                Name:  "John Doe",
                Email: "john@example.com",
            },
            wantErr: false,
        },
        {
            name: "invalid email",
            user: &User{
                Name:  "John Doe",
                Email: "invalid-email",
            },
            wantErr: true,
            errType: ErrInvalidEmail,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            repo := &MockUserRepository{}
            service := NewUserService(repo, logger)

            // Act
            err := service.Create(context.Background(), tt.user)

            // Assert
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errType != nil {
                    assert.ErrorIs(t, err, tt.errType)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods
   - Target: 80%+ code coverage
   - Use mocks for external dependencies
   - Test edge cases and error conditions

2. **Integration Tests**: Test component interactions
   - Test with real database (test database)
   - Test API endpoints
   - Test service interactions

3. **End-to-End Tests**: Test complete user workflows
   - Test through HTTP interface
   - Use test fixtures
   - Test critical user journeys

### Test Data Management

Use the test utilities in `tests/testutils/`:

```go
func TestTicketAPI(t *testing.T) {
    // Use test server
    ts := testutils.NewTestServer(t)
    defer ts.Close()

    // Use test fixtures
    tenant := fixtures.NewTestTenant()
    user := fixtures.NewTestUser(tenant.ID)

    // Test API endpoint
    client := testutils.NewHTTPClient(ts)
    resp, err := client.Post("/api/v1/tickets", "application/json", ticketData)
    assert.NoError(t, err)
    assert.Equal(t, 201, resp.StatusCode)
}
```

## Documentation

### Code Documentation

- Document all public functions, types, and constants
- Use godoc format
- Include examples for complex functions
- Document configuration options

### API Documentation

- Update OpenAPI specification for API changes
- Include example requests and responses
- Document error responses
- Provide curl examples

### README Updates

- Update README.md for new features
- Update installation instructions
- Update configuration examples
- Update contributing guidelines

## Pull Request Process

### Before Creating PR

1. **Complete all requirements**
   - [ ] Code follows style guidelines
   - [ ] Tests pass (`make test`)
   - [ ] Linting passes (`make lint`)
   - [ ] Documentation is updated
   - [ ] CHANGELOG.md is updated (if applicable)

2. **Test thoroughly**
   ```bash
   make pre-commit
   make test-cover
   ```

3. **Commit messages follow Conventional Commits**
   - Use proper types (feat, fix, docs, etc.)
   - Include scope when appropriate
   - Describe what and why, not how

### Creating PR

1. **Use descriptive title**
   - `feat: add user authentication with JWT tokens`
   - `fix: resolve database connection leak in ticket service`

2. **Fill out PR template**
   - Description of changes
   - Testing done
   - Breaking changes (if any)
   - Related issues

3. **Link to issues**
   - Use `Closes #123` for issues that this PR resolves
   - Use `Related to #123` for related issues

### PR Review Process

1. **Automatic checks** (CI/CD)
   - Tests must pass
   - Linting must pass
   - Build must succeed

2. **Code review**
   - At least one approval required
   - Address all review comments
   - Update PR based on feedback

3. **Merge**
   - Squash and merge for feature branches
   - Maintain clean commit history
   - Delete feature branch after merge

## Code Review

### Review Guidelines

When reviewing code, check for:

1. **Correctness**
   - Does the code work as intended?
   - Are there bugs or logic errors?
   - Are edge cases handled?

2. **Design**
   - Is the code well-structured?
   - Does it follow project architecture?
   - Is it maintainable and extensible?

3. **Performance**
   - Are there performance bottlenecks?
   - Is memory usage appropriate?
   - Are database queries optimized?

4. **Security**
   - Are there security vulnerabilities?
   - Is input validation proper?
   - Are sensitive data handled correctly?

5. **Testing**
   - Are tests comprehensive?
   - Do tests cover edge cases?
   - Are tests maintainable?

### Review Etiquette

1. **Be constructive and respectful**
2. **Explain reasoning for suggestions**
3. **Ask questions if something is unclear**
4. **Acknowledge good work and improvements**
5. **Focus on the code, not the author**

## Community

### Getting Help

- **Issues**: Use GitHub Issues for bugs and feature requests
- **Discussions**: Use GitHub Discussions for questions and ideas
- **Discord/Slack**: Join our community chat (link in README)

### Communication

- **Be respectful**: Treat all community members with respect
- **Be inclusive**: Welcome contributors of all backgrounds and skill levels
- **Be patient**: Remember that everyone is volunteering their time

### Recognition

- Contributors are recognized in releases and documentation
- Top contributors are highlighted in project README
- Active contributors may be invited to become maintainers

## Release Process

### Versioning

SmartTicket follows [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. **Code quality**
   - [ ] All tests pass
   - [ ] Code coverage meets requirements
   - [ ] Documentation is updated
   - [ ] CHANGELOG.md is updated

2. **Release preparation**
   - [ ] Version number is updated
   - [ ] Release notes are prepared
   - [ ] Tag is created
   - [ ] Release is published

3. **Post-release**
   - [ ] Update documentation
   - [ ] Announce release
   - [ ] Monitor for issues

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Docker Documentation](https://docs.docker.com/)
- [SmartTicket Development Setup](docs/development-setup.md)

Thank you for contributing to SmartTicket! 🎉