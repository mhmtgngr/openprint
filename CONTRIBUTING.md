# Contributing to OpenPrint Cloud

Thank you for your interest in contributing to OpenPrint Cloud! This document provides guidelines and standards for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Release Process](#release-process)

## Code of Conduct

### Our Pledge

We the maintainers and contributors, pledge to make participation in our project a harassment-free experience for everyone. We are committed to providing a welcoming and inspiring community for all.

### Our Standards

Examples of behavior that contributes to creating a positive environment include:

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

Examples of unacceptable behavior include:

- The use of sexualized language or imagery
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information without explicit permission
- Other conduct which could reasonably be considered inappropriate

## Development Setup

### Prerequisites

- **Go 1.24+**: Install from [golang.org](https://golang.org/)
- **Node.js 18+**: Install from [nodejs.org](https://nodejs.org/)
- **Docker**: Install from [docker.com](https://docker.com/)
- **PostgreSQL 16**: Install or use Docker
- **Redis 7**: Install or use Docker

### Quick Start

```bash
# Clone the repository
git clone https://github.com/openprint/openprint.git
cd openprint

# Install dependencies
go mod download
cd web/dashboard && npm install && cd ../..

# Set up environment
cp .env.example .env
# Edit .env with your settings

# Start services with Docker
cd deployments/docker
docker-compose up -d
cd ../..

# Run tests
make test-unit

# Start developing
make run
```

### Environment Setup

1. Copy `.env.example` to `.env`
2. Set `JWT_SECRET` to a random 32+ character string
3. Configure database URLs if needed
4. Start services: `docker-compose up -d` or run locally

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check:

1. **Existing Issues**: Search existing issues to avoid duplicates
2. **Latest Version**: Verify the bug exists in the latest version
3. **Debug Logs**: Include relevant logs and error messages

When filing a bug report, include:

- **Title**: Clear, descriptive title
- **Description**: Steps to reproduce
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**: OS, Go version, Node version
- **Logs**: Relevant log snippets

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:

1. **Check Existing Issues**: Ensure it's not already suggested
2. **Write a Detailed Proposal**: Include use cases, benefits, implementation ideas
3. **Mark as Discussion**: Label the issue as `[Discussion]`

### Pull Requests

1. **Fork the Repository**: Create your fork
2. **Create a Branch**: Use descriptive name (e.g., `feature/add-mfa`, `fix/auth-timeout`)
3. **Make Your Changes**: Follow coding standards
4. **Test Thoroughly**: Ensure all tests pass
5. **Update Documentation**: Update relevant docs
6. **Submit PR**: Fill in the PR template completely

## Coding Standards

### Go Code

#### Formatting
- Run `gofmt -s -w .` before committing
- Use `goimports` to manage imports
- Maximum line length: 120 characters
- Use tabs for indentation

#### Best Practices
- **Error Handling**: Always check and handle errors explicitly
- **Context**: Pass `context.Context` through all operations
- **Logging**: Use structured logging with context
- **Validation**: Validate all inputs at API boundaries
- **Testing**: Write tests for new code

#### Project Structure
```
services/
├── my-service/
    ├── main.go           # Entry point, HTTP server setup
    ├── handler/          # HTTP handlers
    │   └── my_handler.go
    ├── repository/       # Database operations
    │   └── my_repo.go
    └── my_service_test.go # Tests
```

#### Example Handler
```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 1. Parse and validate input
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid JSON")
        return
    }
    
    if req.Email == "" {
        respondError(w, http.StatusBadRequest, "email required")
        return
    }
    
    // 2. Business logic
    user, err := h.repo.CreateUser(ctx, &req)
    if err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    // 3. Return response
    respondJSON(w, http.StatusCreated, user)
}
```

### TypeScript/React

#### Formatting
- Run `npm run lint` before committing
- Run `npm run type-check` to verify types
- Use Prettier for code formatting
- Maximum line length: 100 characters

#### Best Practices
- **Functional Components**: Use functional components with hooks
- **TypeScript**: Strong typing, no `any` without justification
- **Testing**: Write unit tests for components
- **State Management**: Use Zustand for client state
- **Server State**: Use TanStack Query

### Database Migrations

#### Naming Convention
- Format: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`
- Example: `000029_add_user_preferences.up.sql`

#### Migration Content
- **Up Migration**: Create tables, indexes, constraints
- **Down Migration**: Drop objects in reverse order
- **Idempotent**: Migrations should be re-runnable
- **Transactional**: Wrap in transactions when needed

#### Example Migration
```sql
-- +build Up

CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    value JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_user_preference UNIQUE (user_id, key)
);

CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

-- +build Down

DROP TABLE IF EXISTS user_preferences;
```

### Commit Messages

#### Format
```
<type>(<scope>): <subject>

<body>
```

#### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `test`: Adding/updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks
- `perf`: Performance improvement

#### Examples
```
feat(auth): add multi-factor authentication

- Add TOTP-based MFA support
- Implement backup codes
- Add QR code generation
- Update user settings UI

Closes #123
```

```
fix(job-service): resolve race condition in job assignment

- Add mutex lock around job assignment logic
- Ensure only one agent can claim a job at a time
- Add integration test for concurrent assignments

Fixes #456
```

## Testing Requirements

### Unit Tests
- **Coverage**: Minimum 60% for services, 80% for internal packages
- **Speed**: Unit tests should run in milliseconds
- **Isolation**: Mock external dependencies
- **Naming**: `Test<FunctionName>` or `Test<FunctionName>_<scenario>`

### Integration Tests
- **Build Tag**: Use `// +build integration` tag
- **Docker**: Requires Docker for test containers
- **Cleanup**: Clean up test data after each test
- **Parallel**: Can run in parallel with proper isolation

### Running Tests

```bash
# Unit tests only
make test-unit

# Integration tests (requires Docker)
make test-integration

# All tests
make test

# Coverage report
make test-cover

# Specific package
go test ./services/auth-service/...

# With verbose output
go test -v ./services/job-service/...
```

### Test Examples

#### Unit Test
```go
func TestHashPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"valid password", "SecurePass123!", false},
        {"empty password", "", true},
        {"too short", "a", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            hashed, err := password.Hash(tt.password)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.True(t, password.Verify(tt.password, hashed))
        })
    }
}
```

#### Integration Test
```go
//go:build integration
// +build integration

func TestUserRegistration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    ctx := context.Background()
    db := testutil.SetupTestDB(t)
    defer testutil.CleanupTestDB(t, db)
    
    repo := repository.NewUserRepository(db)
    
    user, err := repo.CreateUser(ctx, &repository.CreateUserRequest{
        Email:    "test@example.com",
        Password: "SecurePass123!",
    })
    
    require.NoError(t, err)
    assert.NotEmpty(t, user.ID)
    assert.Equal(t, "test@example.com", user.Email)
}
```

## Pull Request Process

### Before Submitting

1. **Update from Main**: Ensure your branch is up-to-date
   ```bash
   git checkout main
   git pull origin main
   git checkout your-branch
   git rebase main
   ```

2. **Run All Tests**: Ensure everything passes
   ```bash
   make test
   make lint
   cd web/dashboard && npm run lint && npm run type-check
   ```

3. **Update Documentation**: Update relevant docs

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] All tests passing

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings
- [ ] Tests added and passing

## Related Issues
Fixes #(issue number)
```

### After Submitting

1. **Respond to Reviews**: Address all review comments
2. **Keep Updated**: Rebase if main branch changes
3. **Be Patient**: Maintainers will review as time allows

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/). Version numbers are `MAJOR.MINOR.PATCH`:

- **MAJOR**: Incompatible API changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Release Checklist

- [ ] All tests passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version bumped in code
- [ ] Tag created
- [ ] Release notes written

## Questions or Problems?

- **GitHub Issues**: For bug reports and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Email**: security@openprint.ai for security concerns (DO NOT create public issues for security problems)

## License

By contributing to OpenPrint Cloud, you agree that your contributions will be licensed under the Apache License 2.0.

## Recognition

Contributors are recognized in:
- GitHub's contributors graph
- Release notes for significant contributions
- Our README.md file

Thank you for contributing to OpenPrint Cloud! 🎉
