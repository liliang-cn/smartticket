# Git Hooks Guide

This guide covers the Git hooks used in SmartTicket to maintain code quality, consistency, and development standards.

## Overview

SmartTicket uses custom Git hooks to automate quality checks and enforce development standards. The hooks are designed to catch issues early and maintain consistency across the codebase.

## Installed Hooks

### Pre-commit Hook (`pre-commit`)

**Purpose**: Runs checks before allowing a commit to be created.

**Triggers**: When you run `git commit`

**Checks performed**:
1. **Go formatting** - Ensures code is properly formatted with `gofmt`
2. **Go vet** - Runs static analysis to find potential issues
3. **Import optimization** - Checks imports with `goimports`
4. **Linting** - Runs `golangci-lint` for comprehensive code analysis
5. **Unit tests** - Runs tests for changed packages
6. **TODO/FIXME check** - Warns about temporary comments
7. **Security scan** - Runs `gosec` for security vulnerabilities
8. **License headers** - Ensures proper license headers
9. **Sensitive data check** - Prevents committing secrets/passwords
10. **Build check** - Verifies code builds successfully

**Example output**:
```bash
[PRE-COMMIT] Running pre-commit checks...
[PRE-COMMIT] Found staged Go files:
internal/ticket/service.go
internal/ticket/service_test.go
[PRE-COMMIT] ✓ Go code formatting check passed
[PRE-COMMIT] ✓ go vet check passed
[PRE-COMMIT] ✓ Go imports check passed
[PRE-COMMIT] ✓ golangci-lint check passed
[PRE-COMMIT] ✓ Unit tests passed
[PRE-COMMIT] ✓ License header check passed
[PRE-COMMIT] ✓ Sensitive data check passed
[PRE-COMMIT] ✓ Build check passed
[PRE-COMMIT] ✓ All pre-commit checks passed! ✓
```

### Pre-push Hook (`pre-push`)

**Purpose**: Runs comprehensive checks before allowing a push to remote.

**Triggers**: When you run `git push`

**Checks performed**:
1. **Full test suite** - Runs all tests with coverage
2. **Race condition tests** - Runs tests with race detector (main branch)
3. **Coverage threshold** - Ensures 80%+ test coverage
4. **Comprehensive linting** - Full `golangci-lint` analysis
5. **Security scan** - Comprehensive security analysis
6. **Vulnerability check** - Checks for known vulnerabilities
7. **Multi-platform build** - Builds for all platforms (main branch)
8. **Integration tests** - Runs integration test suite
9. **Documentation check** - Validates README structure
10. **Configuration validation** - Validates YAML configuration files
11. **Uncommitted changes check** - Warns about uncommitted changes
12. **Commit message validation** - Ensures conventional commit format
13. **Performance benchmarks** - Runs benchmark tests (main branch)

**Example output**:
```bash
[PRE-PUSH] Pushing to branch 'feature/user-auth' - running standard checks...
[PRE-PUSH] Running pre-push checks...
[PRE-PUSH] ✓ Full test suite passed
[PRE-PUSH] ✓ Test coverage is 85.3% (meets 80% threshold)
[PRE-PUSH] ✓ Linting passed
[PRE-PUSH] ✓ Security scan passed
[PRE-PUSH] ✓ Local build successful
[PRE-PUSH] ✓ Integration tests passed
[PRE-PUSH] ✓ README.md documentation check passed
[PRE-PUSH] ✓ Configuration validation passed
[PRE-PUSH] ✓ All pre-push checks passed! ✓
```

### Commit Message Hook (`commit-msg`)

**Purpose**: Validates commit messages to ensure they follow conventional commit format.

**Triggers**: When you finish editing a commit message

**Validation rules**:
1. **Conventional format** - Must match `type(scope): description`
2. **Subject length** - 10-72 characters
3. **No trailing period** - Subject should not end with a period
4. **Valid types** - `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `ci`, `build`, `revert`, `bump`
5. **Imperative mood** - Use "add feature" not "added feature"
6. **Breaking changes** - Properly indicated when present
7. **Body formatting** - Blank line between subject and body, wrapped at 72 characters

**Example valid commit messages**:
```bash
feat(auth): add JWT token validation
fix(api): resolve memory leak in ticket processing
docs(readme): update installation instructions
test(tickets): add integration tests for ticket lifecycle
refactor(database): simplify connection pool management
```

### Prepare Commit Message Hook (`prepare-commit-msg`)

**Purpose**: Helps create standardized commit messages by providing templates and suggestions.

**Triggers**: When you start writing a commit message

**Features**:
1. **Branch analysis** - Extracts information from branch name
2. **Ticket number detection** - Extracts ticket numbers (e.g., TICKET-123)
3. **Type suggestion** - Suggests commit type based on branch prefix
4. **Template generation** - Provides a commit message template
5. **Guidelines inclusion** - Includes helpful guidelines in the template

**Example template**:
```bash
feat(TICKET-123): add user authentication

# Commit message guidelines:
# - Keep subject line under 72 characters
# - Use imperative mood ("add feature" not "added feature")
# - Reference issues with "fixes #123" or "closes #123"
#
# Types:
#   feat:     New feature
#   fix:      Bug fix
#   docs:     Documentation changes
#   ...
```

## Installation

### Automatic Installation

The Git hooks are installed automatically when you run:

```bash
make env-setup
```

### Manual Installation

To install the hooks manually:

```bash
./scripts/install-git-hooks.sh
```

### Verification

Verify hooks are installed:

```bash
ls -la .git/hooks/ | grep -E "(pre-commit|pre-push|commit-msg|prepare-commit-msg)"
```

Expected output:
```bash
lrwxr-xr-x  1 user  staff   65 Dec 19 10:30 commit-msg -> ../../scripts/git-hooks/commit-msg
lrwxr-xr-x  1 user  staff   68 Dec 19 10:30 pre-commit -> ../../scripts/git-hooks/pre-commit
lrwxr-xr-x  1 user  staff   66 Dec 19 10:30 pre-push -> ../../scripts/git-hooks/pre-push
lrwxr-xr-x  1 user  staff   78 Dec 19 10:30 prepare-commit-msg -> ../../scripts/git-hooks/prepare-commit-msg
```

## Usage

### Normal Workflow

The hooks run automatically during normal Git operations:

```bash
# 1. Make changes to code
echo "package main" > main.go

# 2. Stage changes
git add main.go

# 3. Commit - pre-commit hook runs
git commit -m "feat: add main package"
# Output: [PRE-COMMIT] ✓ All pre-commit checks passed! ✓

# 4. Push - pre-push hook runs
git push origin feature/new-feature
# Output: [PRE-PUSH] ✓ All pre-push checks passed! ✓
```

### Bypassing Hooks

**Warning**: Only bypass hooks when absolutely necessary!

```bash
# Bypass pre-commit hook
git commit --no-verify -m "WIP commit"

# Bypass pre-push hook
git push --no-verify origin feature/new-feature
```

### Hook Output and Logs

Hooks provide detailed output to help you understand what's being checked:

```bash
[PRE-COMMIT] Running pre-commit checks...
[PRE-COMMIT] Found staged Go files:
main.go
[PRE-COMMIT] Checking Go code formatting...
[PRE-COMMIT] ✓ Go code formatting check passed
[PRE-COMMIT] Running go vet...
[PRE-COMMIT] ✓ go vet check passed
...
```

## Configuration

### Environment Variables

Some hook behavior can be configured with environment variables:

```bash
# Skip certain checks (use with caution)
export SMARTTICKET_SKIP_LINT=true
export SMARTTICKET_SKIP_TESTS=true

# Enable verbose output
export SMARTTICKET_HOOKS_VERBOSE=true
```

### Customization

To customize hook behavior, edit the hook files in `scripts/git-hooks/`:

```bash
# Edit pre-commit hook
nano scripts/git-hooks/pre-commit

# Edit commit message hook
nano scripts/git-hooks/commit-msg
```

After editing, the changes take effect immediately since hooks are symlinked.

## Troubleshooting

### Common Issues

#### Hook not executed

**Symptoms**: Commit or push succeeds without hook output

**Solutions**:
1. Check if hooks are installed:
   ```bash
   ls -la .git/hooks/
   ```
2. Check if hooks are executable:
   ```bash
   ls -la .git/hooks/pre-commit
   ```
3. Reinstall hooks:
   ```bash
   ./scripts/install-git-hooks.sh
   ```

#### Hook fails unexpectedly

**Symptoms**: Hook reports an error but you believe the code is correct

**Solutions**:
1. Run the failing check manually:
   ```bash
   # For formatting issues
   gofmt -l .

   # For linting issues
   golangci-lint run

   # For test failures
   go test ./...
   ```

2. Check hook permissions:
   ```bash
   chmod +x scripts/git-hooks/pre-commit
   ```

3. Run hook manually for debugging:
   ```bash
   .git/hooks/pre-commit
   ```

#### Performance issues

**Symptoms**: Hooks take too long to run

**Solutions**:
1. **Optimize test runs** - Run tests only for changed packages
2. **Parallel execution** - Some checks can run in parallel
3. **Cache results** - Cache expensive operations
4. **Skip optional checks** - Use environment variables to skip expensive checks

```bash
# Skip integration tests for faster commits
export SMARTTICKET_SKIP_INTEGRATION_TESTS=true
```

#### False positives

**Symptoms**: Hook reports issues that aren't actually problems

**Solutions**:
1. **Update hook rules** - Modify the hook to be less strict
2. **Add exceptions** - Add specific exceptions for known cases
3. **Report the issue** - Create an issue to improve the hook

### Debug Mode

Enable debug output for troubleshooting:

```bash
export SMARTTICKET_HOOKS_DEBUG=true
git commit -m "test commit"
```

### Hook Logs

Hooks create temporary logs that can help with debugging:

```bash
# Check for recent hook logs
find /tmp -name "smartticket-hook-*" -mtime -1

# Hook logs are automatically cleaned up after 24 hours
```

## Best Practices

### Writing Commit Messages

1. **Use conventional format**: `type(scope): description`
2. **Be specific**: Describe what changed and why
3. **Use imperative mood**: "Add feature" not "Added feature"
4. **Reference issues**: `fixes #123` or `closes #123`
5. **Keep subject short**: Under 72 characters

### Working with Hooks

1. **Run checks locally**: Fix issues before pushing
2. **Use the feedback**: Hook output helps improve code quality
3. **Don't bypass unnecessarily**: Only bypass hooks when absolutely needed
4. **Contribute improvements**: Help improve the hooks for everyone

### Team Collaboration

1. **Consistent environment**: Ensure all team members have the same hooks
2. **Hook updates**: Regularly update hooks when improving the process
3. **Documentation**: Keep this guide updated with changes
4. **Training**: Help new team members understand hook usage

## Advanced Usage

### Custom Hooks

You can add custom hooks for additional checks:

```bash
# Create a custom hook
cat > scripts/git-hooks/post-checkout << 'EOF'
#!/bin/bash
# Custom post-checkout hook
echo "Post-checkout: Running custom checks..."
# Add your custom logic here
EOF

chmod +x scripts/git-hooks/post-checkout

# Install the custom hook
ln -s scripts/git-hooks/post-checkout .git/hooks/post-checkout
```

### Integration with CI/CD

The hooks complement CI/CD pipelines:

1. **Local feedback**: Get immediate feedback before CI
2. **Reduce CI load**: Fewer CI failures due to pre-checks
3. **Faster development**: Quick iteration with quality gates
4. **Consistent standards**: Same checks locally and in CI

### Hook Chaining

You can chain multiple tools in a single hook:

```bash
# Example in pre-commit hook
print_status "Running security and quality checks..."

# Security scan
gosec ./...

# Quality check
golangci-lint run

# Custom tool
my-custom-tool check

print_success "All checks passed"
```

## Maintenance

### Updating Hooks

When the project adds new hook features:

```bash
# Update hooks from repository
git pull origin main

# Reinstall hooks to get latest changes
./scripts/install-git-hooks.sh
```

### Hook Versioning

Hooks are versioned with the project:

- Check `scripts/git-hooks/` directory for hook versions
- Update hooks when updating the project
- Test hooks after updates

### Performance Monitoring

Monitor hook performance:

```bash
# Time hook execution
time .git/hooks/pre-commit

# Profile slow hooks
time golangci-lint run
```

## Support

### Getting Help

If you encounter issues with Git hooks:

1. **Check this guide** for common solutions
2. **Search existing issues** on GitHub
3. **Create a new issue** with details about the problem
4. **Include hook output** and error messages

### Contributing

Help improve the Git hooks:

1. **Report issues** when hooks don't work as expected
2. **Suggest improvements** for better performance or coverage
3. **Submit pull requests** with hook enhancements
4. **Update documentation** when adding new features