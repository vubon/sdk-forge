# Contributing to SDK Forge

Thank you for your interest in contributing to SDK Forge! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Project Structure](#project-structure)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Make (for using the Makefile)
- Git

### Setting Up Your Development Environment

1. **Fork the repository** on GitHub

2. **Clone your fork**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/sdk-forge.git
   cd sdk-forge
   ```

3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/vubon/sdk-forge.git
   ```

4. **Install dependencies**:
   ```bash
   make deps
   ```

5. **Build the project**:
   ```bash
   make build
   ```

6. **Run tests** to verify everything works:
   ```bash
   make test
   ```

## Development Workflow

### 1. Branch Management

**ALWAYS** create a new branch before making any code changes.

- **Branch naming convention**: `feature/<topic>`, `fix/<issue>`, or `docs/<topic>`
- Use descriptive branch names that reflect the work being done
- Examples:
  - `feature/add-php-sdk`
  - `fix/python-version-handling`
  - `docs/update-readme`

**Before starting work**:
```bash
# Check current branch
git branch

# Create and switch to a new branch
git checkout -b feature/your-feature-name

# Verify clean working tree
git status
```

### 2. Making Changes

1. Make your code changes
2. Write or update tests for your changes
3. Ensure your code follows the project's code standards
4. Run all quality checks before committing

### 3. Code Quality Checks

After completing any feature or fix, **MUST** run:

```bash
make check    # Formatting + linting
make test     # Run all tests
```

**DO NOT** commit code that fails:
- Linting checks
- Tests
- Formatting checks

### 4. Before Committing

Follow this checklist before every commit:

- [ ] Created feature branch
- [ ] Code changes completed
- [ ] Tests added/updated
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make check` passes
- [ ] Code reviewed (self-review with `git diff`)
- [ ] Commit message follows [Conventional Commits](#commit-messages)

**Commands to run**:
```bash
# Check formatting
make fmt-check

# Run linter
make lint

# Run tests
make test

# Run all checks
make check

# Review your changes
git diff
```

## Code Standards

### Go Code Style

- Follow Go best practices and conventions
- Use `gofmt` for formatting (enforced via `make fmt`)
- Follow linting rules (golangci-lint)
- Write clear, self-documenting code
- Add comments for complex logic
- Use meaningful variable and function names

### Code Formatting

The project uses `gofmt` for code formatting. Always format your code before committing:

```bash
make fmt        # Format code
make fmt-check  # Check if code is formatted
```

### Linting

We use `golangci-lint` for code linting. The linter will automatically install if not present:

```bash
make lint
```

## Testing Requirements

### Test Coverage

- **MUST** add test cases when adding new code or functionality
- **MUST** update existing tests when modifying existing functionality
- Test coverage should be maintained or improved
- All tests must pass before committing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests in short mode (skip long-running tests)
make test-short
```

### Writing Tests

- Place test files next to the code they test (e.g., `file.go` → `file_test.go`)
- Use table-driven tests when appropriate
- Test both success and error cases
- Use descriptive test names that explain what is being tested

## Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

### Format

```
<type>: <description>

[optional body]

[optional footer]
```

### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `test`: Adding or updating tests
- `refactor`: Code changes that neither fix a bug nor add a feature
- `style`: Changes that do not affect the meaning of the code (formatting, etc.)
- `chore`: Changes to build process or auxiliary tools

### Examples

```bash
feat: add PHP SDK generation support
fix: handle missing OpenAPI version field
docs: update installation instructions
test: add integration tests for Go SDK generation
refactor: extract common template functions
```

## Pull Request Process

### Creating a Pull Request

1. **Push your branch**:
   ```bash
   git push -u origin feature/your-feature-name
   ```

2. **Create a Pull Request** on GitHub:
   - Use a descriptive title
   - Provide a clear description of changes
   - Reference any related issues
   - Wait for CI/CD checks to pass

3. **PR Requirements**:
   - All CI/CD checks must pass
   - Code must be reviewed and approved
   - Tests must pass
   - Code must follow project standards

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactoring

## Related Issues
Closes #123

## Testing
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Manual testing completed (if applicable)

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated (if needed)
- [ ] No new warnings generated
- [ ] Tests added/updated
```

### CI/CD

- Pull requests targeting `main` will automatically run CI/CD checks
- The CI pipeline runs:
  - Code formatting checks
  - Linting
  - Tests
  - Build verification
- All checks must pass before merging

## Version Management

- Version is managed in the `VERSION` file (SemVer 2.0.0)
- **Do not manually update version** unless explicitly requested
- Version follows format: `MAJOR.MINOR.PATCH-PRERELEASE`
- Version priority: OpenAPI schema → Command line → Default

## Project Structure

```
sdk-forge/
├── cmd/
│   └── cli/              # CLI application entry point
├── internal/
│   ├── generator/        # SDK generation logic
│   ├── parser/           # OpenAPI schema parsing
│   └── validator/        # Validation logic
├── pkg/
│   └── languages/        # Language-specific utilities
├── examples/             # Example OpenAPI schemas
├── docs/                 # Documentation
└── .github/
    └── workflows/        # CI/CD workflows
```

## Exceptions

- **Documentation-only changes**: May skip tests (but still run linting)
- **Emergency hotfixes**: Should still follow workflow but can be expedited
- Always get explicit approval before skipping any step

## Getting Help

- Open an issue for bug reports or feature requests
- Check existing issues and discussions
- Review the [documentation](docs/README.md)

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to SDK Forge! 🚀

