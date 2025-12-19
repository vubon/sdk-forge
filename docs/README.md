# SDK Forge - Usage Guide

**Version**: 0.3.0  
**Last Updated**: December 2025

Quick reference guide for using SDK Forge. For detailed documentation, see the [User Manual](MANUAL.md).

---

## Quick Start

### 1. Install SDK Forge

```bash
# Clone and build
git clone https://github.com/vubon/sdk-forge.git
cd sdk-forge
make build
```

### 2. Generate Your First SDK

```bash
# Generate a Python SDK
sdk-forge generate \
  --schema examples/petstore.yaml \
  --lang python \
  --name my-api-sdk \
  --output ./sdks

# Generate a Go SDK
sdk-forge generate \
  --schema examples/petstore.yaml \
  --lang go \
  --name my-api-client \
  --output ./sdks
```

### 3. Use Your Generated SDK

**Python:**
```bash
cd ./sdks/python/my-api-sdk
pip install -e .
```

**Go:**
```bash
cd ./sdks/go/my-api-client
go mod download
```

---

## Common Tasks

### Generate SDK for a Specific Language

See [Language-Specific Guides](MANUAL.md#language-specific-guides) in the manual.

- **[Python SDK Guide](MANUAL.md#python-sdk-guide)** - Installation, usage, authentication
- **[Go SDK Guide](MANUAL.md#go-sdk-guide)** - Installation, usage, authentication

### Configure Authentication

See [Authentication](MANUAL.md#authentication) section in the manual for:
- API Key authentication
- Bearer token authentication
- Basic authentication
- Digest authentication
- OAuth2 authentication
- OpenID Connect authentication
- Mutual TLS (mTLS) authentication

### Use Interactive Mode

See [Interactive Mode](MANUAL.md#interactive-mode) section in the manual for step-by-step prompts.

### Advanced Options

See [Advanced Usage](MANUAL.md#advanced-usage) section in the manual for:
- Custom HTTP libraries
- Version management
- Batch generation
- Remote schema URLs

---

## Command Reference

### Basic Command

```bash
sdk-forge generate [flags]
```

### Required Flags

- `--schema` / `-s`: Path or URL to OpenAPI schema (YAML/JSON)
- `--lang` / `--language` / `-l`: Target language (`python`, `go`, `all`)
- `--name` / `-n`: Name for the generated SDK
- `--output` / `-o`: Output directory for generated SDK

### Common Optional Flags

- `--go-version`: Go version (1.24, 1.25). Default: 1.24
- `--python-version`: Python version (3.11-3.14). Default: 3.11
- `--sdk-version`: SDK version. Default: from OpenAPI schema or 1.0.0
- `--http-lib`: HTTP library to use (defaults to language-specific default)
- `--skip-tests`: Skip test generation
- `--force` / `-f`: Overwrite existing SDK directory

For complete flag documentation, see [Command Reference](MANUAL.md#command-reference) in the manual.

---

## Documentation

- **[User Manual](MANUAL.md)** - Complete guide with detailed instructions
  - Installation
  - Command reference
  - Language-specific guides
  - Advanced usage
  - Troubleshooting
  - Best practices
  - Examples

- **[Main README](../README.md)** - Project overview and features

- **[CHANGELOG](../CHANGELOG.md)** - Version history and changes

---

## Getting Help

### Common Issues

See [Troubleshooting](MANUAL.md#troubleshooting) section in the manual for:
- Schema validation errors
- Output directory issues
- Authentication problems
- Test generation failures

### Examples

See [Examples](MANUAL.md#examples) section in the manual for:
- Simple API SDK generation
- API with authentication
- Multi-language generation
- Custom versions

---

## Next Steps

1. **Read the [User Manual](MANUAL.md)** for comprehensive documentation
2. **Check [Examples](MANUAL.md#examples)** for real-world usage
3. **Review [Best Practices](MANUAL.md#best-practices)** for optimal results
4. **See [Troubleshooting](MANUAL.md#troubleshooting)** if you encounter issues

---

**Need more help?** See the [User Manual](MANUAL.md) for detailed documentation.
