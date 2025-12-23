# SDK Forge - User Manual

**Version**: 0.6.0  
**Last Updated**: December 2025

A comprehensive guide to using SDK Forge to generate production-ready SDKs from OpenAPI schemas.

---

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Command Reference](#command-reference)
4. [Interactive Mode](#interactive-mode)
5. [Generating SDKs](#generating-sdks)
6. [Language-Specific Guides](#language-specific-guides)
7. [Advanced Usage](#advanced-usage)
   - [Retry Configuration](#retry-configuration)
8. [Troubleshooting](#troubleshooting)
9. [Best Practices](#best-practices)
10. [Examples](#examples)

---

## Installation

### Prerequisites

- **Go 1.24+** (for building SDK Forge)
- **Python 3.11+** (if generating Python SDKs)
- **Node.js 18+** (if generating TypeScript/JavaScript SDKs)
- **OpenAPI 3.0.x or 3.1.x** schema file

### Install from Source

```bash
# Clone the repository
git clone https://github.com/vubon/sdk-forge.git
cd sdk-forge

# Build the binary
make build

# Or install directly to $GOPATH/bin
make install
```

### Manual Build

```bash
go build -o sdk-forge ./cmd/cli
```

### Verify Installation

```bash
sdk-forge --version
# Should output: sdk-forge version 0.3.0-alpha.2
```

---

## Quick Start

### Generate Your First SDK

**1. Prepare your OpenAPI schema**

Ensure you have an OpenAPI 3.x schema file (YAML or JSON). Example:

```yaml
openapi: 3.0.0
info:
  title: My API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /users:
    get:
      summary: List users
      operationId: listUsers
      responses:
        '200':
          description: Success
```

**2. Generate a Python SDK**

```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --name my-api-sdk \
  --output ./sdks
```

**3. Generate a Go SDK**

```bash
sdk-forge generate \
  --schema api.yaml \
  --lang go \
  --name my-api-client \
  --output ./sdks
```

**4. Generate a TypeScript/JavaScript SDK**

```bash
sdk-forge generate \
  --schema api.yaml \
  --lang typescript \
  --name my-api-sdk \
  --output ./sdks
```

**5. Generate SDKs for all languages**

```bash
sdk-forge generate \
  --schema api.yaml \
  --lang all \
  --name my-api-sdk \
  --output ./sdks
```

---

## Command Reference

### Basic Syntax

```bash
sdk-forge generate [flags]
```

### Required Flags

| Flag | Short | Description | Example |
|------|-------|-------------|---------|
| `--schema` | `-s` | Path or URL to OpenAPI schema (YAML/JSON) | `--schema api.yaml` |
| `--lang` | `-l` | Target language (`python`, `go`, `php`, `typescript`, `all`) | `--lang python` |
| `--name` | `-n` | Name for the generated SDK | `--name my-sdk` |
| `--output` | `-o` | Output directory for generated SDK | `--output ./sdks` |

### Optional Flags

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--http-lib` | HTTP library to use | Language default | `--http-lib httpx` |
| `--go-version` | Go version (1.24, 1.25) | 1.24 | `--go-version 1.25` |
| `--python-version` | Python version (3.11-3.14) | 3.11 | `--python-version 3.14` |
| `--php-version` | PHP version (8.0-8.3) | 8.1 | `--php-version 8.3` |
| `--typescript-version` | TypeScript version (4.9-5.5) | 5.0 | `--typescript-version 5.5` |
| `--sdk-version` | SDK version | OpenAPI schema or 1.0.0 | `--sdk-version 2.0.0` |
| `--skip-tests` | Skip test generation | Tests generated | `--skip-tests` |
| `--ignore-minor-issues` | Ignore minor validation issues | false | `--ignore-minor-issues` |
| `--force` | `-f` | Overwrite existing SDK | false | `--force` |
| `--retry-enabled` | Enable retry logic for HTTP requests | false | `--retry-enabled` |
| `--retry-max-attempts` | Maximum retry attempts | 3 | `--retry-max-attempts 5` |
| `--retry-strategy` | Retry strategy (exponential, linear, fixed) | exponential | `--retry-strategy linear` |
| `--retry-status-codes` | Comma-separated status codes to retry | 429,500,502,503,504 | `--retry-status-codes "429,500"` |

### Flag Details

#### `--schema` / `-s`

Path to your OpenAPI schema file or URL.

- **Local file**: `--schema ./api.yaml` or `--schema /path/to/api.json`
- **Remote URL**: `--schema https://api.example.com/openapi.yaml`
- **Formats**: YAML (`.yaml`, `.yml`) or JSON (`.json`)

#### `--lang` / `--language` / `-l`

Target programming language for SDK generation.

- **Options**: `python`, `go`, `php`, `typescript`, `all`
- **Future**: `js`, `ts`, `ruby`
- **Special**: `all` generates SDKs for all available languages

#### `--name` / `-n`

Name for your generated SDK. This will be used for:
- Package/module names
- Client class names
- Directory names

**Naming Guidelines:**
- Use lowercase with hyphens: `my-api-sdk`
- Avoid special characters
- Keep it descriptive and short

#### `--output` / `-o`

Output directory where SDKs will be generated.

**Output Structure:**
```
output/
├── python/
│   └── sdk-name/
├── go/
│   └── sdk-name/
├── php/
│   └── sdk-name/
│       └── PascalCaseName/
└── typescript/
    └── sdk-name/
        └── sdk-name/
```

#### `--http-lib`

HTTP library to use for the generated SDK.

**Python Options:**
- `requests` (default)
- `httpx`
- `aiohttp`
- `urllib3`

**Go Options:**
- `net/http` (default, standard library)

**PHP Options:**
- `guzzle` (default)

**TypeScript/JavaScript Options:**
- `axios` (default)
- `fetch`
- `node-fetch`
- `ky`

#### `--go-version`

Target Go version for the generated SDK.

- **Options**: `1.24`, `1.25`
- **Default**: `1.24`
- Affects generated code features and compatibility

#### `--python-version`

Target Python version for the generated SDK.

- **Options**: `3.11`, `3.12`, `3.13`, `3.14`
- **Default**: `3.11`
- Affects type hints and language features

#### `--php-version`

Target PHP version for the generated SDK.

- **Options**: `8.0`, `8.1`, `8.2`, `8.3`
- **Default**: `8.1`

#### `--typescript-version`

Target TypeScript version for the generated SDK.

- **Options**: `4.9`, `5.0`, `5.1`, `5.2`, `5.3`, `5.4`, `5.5`
- **Default**: `5.0`
- Affects type hints and language features

#### `--sdk-version`

Version for the generated SDK.

**Priority Order:**
1. OpenAPI schema `info.version` (if present)
2. `--sdk-version` flag value
3. Default: `1.0.0`

**Format**: Semantic Versioning (SemVer 2.0.0)
- Examples: `1.0.0`, `2.1.3`, `1.0.0-alpha.1`

#### `--skip-tests`

Skip automatic test generation.

- **Default**: Tests are generated
- Use when you want to generate code only
- Tests include: client, models, API methods, authentication, error handling

#### `--ignore-minor-issues`

Ignore minor OpenAPI validation issues.

- **Default**: All issues block generation
- Use when schema has minor issues that don't affect SDK generation
- Major issues (missing paths, invalid schemas) still block generation

#### `--force` / `-f`

Overwrite existing SDK directory.

- **Default**: Fails if directory exists
- Use to regenerate SDKs without manual cleanup
- **Warning**: This will delete existing files in the output directory

#### Retry Configuration Flags

Configure retry logic for HTTP requests in generated SDKs. Retry is **disabled by default** for backward compatibility.

**Enable Retry:**
```bash
--retry-enabled
```

**Retry Options:**
- `--retry-max-attempts`: Maximum retry attempts (default: `3`)
- `--retry-initial-delay`: Initial retry delay in seconds (default: `1.0`)
- `--retry-max-delay`: Maximum retry delay in seconds (default: `60.0`)
- `--retry-backoff-multiplier`: Exponential backoff multiplier (default: `2.0`)
- `--retry-strategy`: Retry strategy - `exponential`, `linear`, or `fixed` (default: `exponential`)
- `--retry-status-codes`: Comma-separated HTTP status codes to retry (default: `429,500,502,503,504`)

**Example:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --name my-sdk \
  --retry-enabled \
  --retry-max-attempts 5 \
  --retry-strategy exponential \
  --retry-status-codes "429,500,502,503,504"
```

For detailed retry configuration documentation, see the [Retry Configuration](#retry-configuration) section.

---

## Interactive Mode

SDK Forge automatically enters interactive mode when required flags are missing.

### When Interactive Mode Activates

Interactive mode activates when any of these are missing:
- `--schema`
- `--lang` / `--language`
- `--name`
- `--output`

### Interactive Mode Flow

```bash
$ sdk-forge generate --schema api.yaml

? Select language: 
  ▸ python
    go
    php
    typescript
    all

? Enter SDK name: my-api-sdk

? Enter output directory: ./sdks

? Go version (1.24, 1.25) [1.24]: 

? Python version (3.11, 3.12, 3.13, 3.14) [3.11]: 

? TypeScript version (4.9, 5.0, 5.1, 5.2, 5.3, 5.4, 5.5) [5.0]: 

? SDK version (leave empty to use OpenAPI schema version): 

? HTTP library (leave empty for default): 

? Skip test generation? (y/N): 

? Force overwrite existing directory? (y/N): 

Generating SDK...
✓ SDK generated successfully!
```

### Interactive Mode Tips

- Use arrow keys to navigate options
- Press Enter to use default values
- Type answers for text prompts
- Use `Ctrl+C` to cancel

---

## Generating SDKs

### Step-by-Step Process

**1. Prepare Your OpenAPI Schema**

Ensure your OpenAPI schema is valid:

```bash
# Validate schema (optional, SDK Forge will validate)
# Use online tools like Swagger Editor or Redoc
```

**2. Choose Your Language**

Decide which language(s) to generate:
- Single language: `--lang python`, `--lang go`, or `--lang php`
- All languages: `--lang all`

**3. Set Output Directory**

Choose where to generate SDKs:

```bash
--output ./sdks          # Current directory
--output ~/projects/sdks # Home directory
--output /tmp/sdks       # Temporary directory
```

**4. Run Generation**

```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --name my-api \
  --output ./sdks
```

**5. Verify Generated SDK**

Check the output directory:

```bash
ls -la ./sdks/python/my-api/
```

### Generation Output

After successful generation, you'll see:

```
✓ SDK generated successfully!
Location: ./sdks/python/my-api/
```

### What Gets Generated

**For Python:**
- Package structure with `__init__.py`
- Client class with authentication
- Data models from schemas
- API methods organized by tags
- `setup.py` and `requirements.txt`
- Test suite (if not skipped)
- Usage examples

**For Go:**
- Module structure with `go.mod`
- Client struct and methods
- Data models from schemas
- API methods organized by tags
- Test files (if not skipped)
- Usage examples

**For PHP:**
- PSR-4 autoloading structure with `composer.json`
- Client class with authentication
- Data models from schemas
- API classes organized by tags
- PHPUnit test suite (if not skipped)
- Usage examples
- Code quality configuration (PHP-CS-Fixer, PHPStan, PHP_CodeSniffer)

---

## Language-Specific Guides

For detailed, language-specific documentation, see the dedicated guides:

- **[Python SDK Guide](languages/python.md)** - Complete Python SDK documentation
  - Installation and setup
  - Basic usage and examples
  - Authentication methods
  - HTTP libraries (requests, httpx, aiohttp, urllib3)
  - Retry mechanism
  - Testing with pytest
  - Advanced usage patterns
  - Troubleshooting

- **[Go SDK Guide](languages/go.md)** - Complete Go SDK documentation
  - Installation and setup
  - Basic usage and examples
  - Authentication methods
  - HTTP libraries (net/http, resty, gentleman)
  - Retry mechanism
  - Testing with go test
  - Advanced usage patterns
  - Troubleshooting

- **[PHP SDK Guide](languages/php.md)** - Complete PHP SDK documentation
  - Installation with Composer
  - Basic usage and examples
  - Authentication methods
  - HTTP libraries (Guzzle)
  - Retry mechanism
  - Testing with PHPUnit
  - Advanced usage patterns
  - Troubleshooting

- **[TypeScript/JavaScript SDK Guide](languages/typescript.md)** - Complete TypeScript/JavaScript SDK documentation
  - Installation with npm/yarn
  - Basic usage and examples (ESM/CommonJS)
  - Authentication methods
  - HTTP libraries (axios, fetch, node-fetch, ky)
  - Retry mechanism
  - Testing with Jest
  - TypeScript type safety
  - Advanced usage patterns
  - Troubleshooting

Each guide includes comprehensive examples, best practices, and troubleshooting tips specific to that language.

---

## Advanced Usage

### Generating Multiple SDKs

**Generate for all languages:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang all \
  --name my-api \
  --output ./sdks
```

**Generate with different versions:**
```bash
# Python 3.14
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --python-version 3.14 \
  --name my-api \
  --output ./sdks

# Go 1.25
sdk-forge generate \
  --schema api.yaml \
  --lang go \
  --go-version 1.25 \
  --name my-api \
  --output ./sdks

# PHP 8.3
sdk-forge generate \
  --schema api.yaml \
  --lang php \
  --php-version 8.3 \
  --name my-api \
  --output ./sdks

# TypeScript 5.5
sdk-forge generate \
  --schema api.yaml \
  --lang typescript \
  --typescript-version 5.5 \
  --name my-api \
  --output ./sdks
```

### Custom HTTP Libraries

**Python with httpx:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --http-lib httpx \
  --name my-api \
  --output ./sdks
```

### Version Management

**Override SDK version:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --sdk-version 2.0.0 \
  --name my-api \
  --output ./sdks
```

**Use OpenAPI schema version:**
```yaml
# In your OpenAPI schema
info:
  version: 1.2.3
```

SDK Forge will automatically use `1.2.3` as the SDK version.

### Regenerating SDKs

**Force overwrite:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --name my-api \
  --output ./sdks \
  --force
```

### Skipping Tests

**Generate code only:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --name my-api \
  --output ./sdks \
  --skip-tests
```

### Retry Configuration

SDK Forge can generate SDKs with automatic retry logic for handling transient network errors and rate limits. Retry is **disabled by default** but can be enabled via CLI flags or OpenAPI extensions.

#### Overview

When retry is enabled, generated SDKs will automatically retry failed HTTP requests based on:
- **HTTP Status Codes**: Configurable status codes that trigger retries (default: 429, 500, 502, 503, 504)
- **Network Errors**: Connection timeouts, DNS errors, and other network-related failures
- **Retry Strategies**: Exponential backoff, linear backoff, or fixed delay

#### Retry Strategies

**1. Exponential Backoff (Default)**
- Delay increases exponentially: 1s, 2s, 4s, 8s, 16s...
- Formula: `initialDelay * (multiplier ^ attempt)`
- Best for: APIs with rate limits or high load

**2. Linear Backoff**
- Delay increases linearly: 1s, 2s, 3s, 4s, 5s...
- Formula: `initialDelay * (attempt + 1)`
- Best for: Predictable retry patterns

**3. Fixed Delay**
- Constant delay between retries: 1s, 1s, 1s, 1s...
- Formula: Always `initialDelay`
- Best for: Simple retry scenarios

#### Configuration via CLI Flags

**Basic Example:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --name my-sdk \
  --retry-enabled \
  --retry-max-attempts 3 \
  --retry-strategy exponential
```

**Advanced Example:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang go \
  --name my-sdk \
  --retry-enabled \
  --retry-max-attempts 5 \
  --retry-initial-delay 2.0 \
  --retry-max-delay 120.0 \
  --retry-backoff-multiplier 2.5 \
  --retry-strategy exponential \
  --retry-status-codes "429,500,502,503,504,408"
```

#### Configuration via OpenAPI Extension

You can define retry configuration directly in your OpenAPI schema using the `x-sdk-forge-retry` extension:

```yaml
openapi: 3.0.0
info:
  title: My API
  version: 1.0.0

x-sdk-forge-retry:
  enabled: true
  maxAttempts: 3
  initialDelay: 1
  maxDelay: 60
  strategy: exponential
  backoffMultiplier: 2.0
  retryableStatusCodes: [429, 500, 502, 503, 504]
  retryOnNetworkErrors: true

paths:
  /users:
    get:
      summary: List users
      responses:
        '200':
          description: Success
```

**Configuration Priority:**
1. CLI flags (highest priority - override OpenAPI extension)
2. OpenAPI extension (`x-sdk-forge-retry`)
3. Default values (if neither is provided)

#### Using Retry in Generated SDKs

**Python SDK Example:**
```python
from my_sdk import MySdkClient

# Client is initialized with retry configuration
client = MySdkClient(
    base_url="https://api.example.com/v1",
    api_key="your-key"
)

# Retry happens automatically on retryable errors
try:
    users = client.list_users()
except Exception as e:
    # Retry exhausted - handle final error
    print(f"Request failed after retries: {e}")
```

**Go SDK Example:**
```go
package main

import (
    "fmt"
    "github.com/example/my-sdk"
)

func main() {
    // Client is initialized with retry configuration
    client := mysdk.NewMySdkClient("https://api.example.com/v1")
    client.ApiKey = "your-key"
    
    // Retry happens automatically on retryable errors
    data, err := client.ListUsers()
    if err != nil {
        // Retry exhausted - handle final error
        fmt.Printf("Request failed after retries: %v\n", err)
        return
    }
    
    fmt.Println(string(data))
}
```

#### Retry Behavior

**When Retries Occur:**
- HTTP status codes in `retryableStatusCodes` list (default: 429, 500, 502, 503, 504)
- Network errors (connection refused, timeouts, DNS errors) if `retryOnNetworkErrors` is `true`
- Transient failures that may succeed on retry

**When Retries Don't Occur:**
- HTTP 4xx errors (except 429) - client errors that won't succeed on retry
- HTTP 2xx, 3xx - successful responses
- Non-retryable network errors (if `retryOnNetworkErrors` is `false`)
- After `maxAttempts` is reached

**Retry Flow:**
1. Make HTTP request
2. If error/retryable status → wait for calculated delay
3. Retry request (up to `maxAttempts` times)
4. If all retries fail → return final error

#### Best Practices

1. **Choose Appropriate Strategy:**
   - Use **exponential backoff** for rate-limited APIs (429 errors)
   - Use **linear backoff** for predictable retry patterns
   - Use **fixed delay** for simple scenarios

2. **Set Reasonable Limits:**
   - `maxAttempts`: 3-5 attempts is usually sufficient
   - `maxDelay`: Cap delays to avoid long waits (60-120 seconds)
   - `initialDelay`: Start with 1-2 seconds

3. **Configure Status Codes:**
   - Include `429` (Too Many Requests) for rate-limited APIs
   - Include `5xx` codes (500, 502, 503, 504) for server errors
   - Exclude `4xx` codes (except 429) - client errors won't succeed

4. **Test Retry Behavior:**
   - Test with mock servers that simulate failures
   - Verify retry attempts and delays
   - Ensure final errors are properly handled

#### Troubleshooting Retry

**Retry Not Working:**
- Ensure `--retry-enabled` flag is set
- Check that status codes are in `retryableStatusCodes`
- Verify `retryOnNetworkErrors` is `true` for network errors

**Too Many Retries:**
- Reduce `maxAttempts`
- Check if non-retryable errors are being retried
- Review retryable status codes list

**Retries Taking Too Long:**
- Reduce `maxDelay`
- Use `linear` or `fixed` strategy instead of `exponential`
- Reduce `initialDelay` and `backoffMultiplier`

### Remote Schema URLs

**Generate from remote OpenAPI schema:**
```bash
sdk-forge generate \
  --schema https://api.example.com/openapi.yaml \
  --lang python \
  --name my-api \
  --output ./sdks
```

---

## Troubleshooting

### macOS Security Warning (Quarantine)

If macOS shows a security warning when running the downloaded binary (e.g., "sdk-forge cannot be opened because it is from an unidentified developer"), you need to remove the quarantine attribute:

```bash
# Check if file has quarantine attribute
xattr -l ~/bin/sdk-forge
# Or if the file is in a different location:
xattr -l /path/to/sdk-forge-0.3.0-darwin-arm64

# If you see "com.apple.quarantine", remove it:
xattr -d com.apple.quarantine ~/bin/sdk-forge
# Or:
xattr -d com.apple.quarantine /path/to/sdk-forge-0.3.0-darwin-arm64
```

**Alternative method:**
1. Right-click the binary file
2. Select **"Open"** (not double-click)
3. Click **"Open"** in the security dialog (first time only)
4. After this, you can run it normally from the terminal

**Note:** This is a macOS security feature (Gatekeeper) that applies to all unsigned binaries downloaded from the internet. The binary is safe - it's just not signed with an Apple Developer certificate.

### Common Issues

#### 1. "Schema file not found"

**Problem:**
```
Error: open api.yaml: no such file or directory
```

**Solution:**
- Check the file path is correct
- Use absolute path: `--schema /full/path/to/api.yaml`
- Verify file exists: `ls -la api.yaml`

#### 2. "Invalid OpenAPI schema"

**Problem:**
```
Error: failed to parse OpenAPI schema: invalid schema
```

**Solution:**
- Validate your OpenAPI schema using [Swagger Editor](https://editor.swagger.io/)
- Check schema version (must be 3.0.x or 3.1.x)
- Ensure required fields are present (`openapi`, `info`, `paths`)

#### 3. "Output directory already exists"

**Problem:**
```
Error: output directory already exists: ./sdks/python/my-api
```

**Solution:**
- Use `--force` to overwrite: `--force`
- Or delete the directory first: `rm -rf ./sdks/python/my-api`

#### 4. "Language not supported"

**Problem:**
```
Error: unsupported language: php
```

**Solution:**
- Check supported languages: `python`, `go`, `php`, `all`
- Future languages (js, ts) coming soon

#### 5. "Test generation failed"

**Problem:**
```
Warning: failed to format test file
```

**Solution:**
- This is usually non-fatal (tests are still generated)
- Install formatters: `pip install black` (Python) or ensure `gofmt` is available (Go)
- Check file permissions

#### 6. "Authentication not working"

**Problem:**
Generated SDK doesn't send authentication headers.

**Solution:**
- Verify OpenAPI schema defines security schemes correctly
- Check security requirements on operations
- Ensure you're setting authentication on the client correctly

### Getting Help

**Check logs:**
- SDK Forge provides detailed error messages
- Check the error output for specific issues

**Validate OpenAPI schema:**
- Use [Swagger Editor](https://editor.swagger.io/)
- Use [Redoc](https://redocly.com/docs/cli/)

**Report issues:**
- GitHub Issues: https://github.com/vubon/sdk-forge/issues
- Include: OpenAPI schema (if possible), error message, SDK Forge version

---

## Best Practices

### OpenAPI Schema

1. **Use descriptive operation IDs**
   ```yaml
   operationId: listUsers  # Good
   operationId: get1      # Bad
   ```

2. **Include descriptions**
   ```yaml
   paths:
     /users:
       get:
         summary: List all users
         description: Retrieves a paginated list of users
   ```

3. **Define schemas properly**
   ```yaml
   components:
     schemas:
       User:
         type: object
         properties:
           id:
             type: integer
           name:
             type: string
   ```

4. **Use tags for organization**
   ```yaml
   tags:
     - name: users
       description: User management operations
   ```

5. **Include examples**
   ```yaml
   responses:
     '200':
       content:
         application/json:
           example:
             id: 1
             name: "John Doe"
   ```

### SDK Generation

1. **Use semantic versioning**
   - Set `info.version` in OpenAPI schema
   - Or use `--sdk-version` flag

2. **Generate tests**
   - Keep `--skip-tests` off (default)
   - Tests help verify SDK functionality

3. **Organize output**
   - Use consistent output directory structure
   - Example: `./sdks/{language}/{sdk-name}/`

4. **Version control**
   - Commit generated SDKs to version control
   - Or generate on-demand in CI/CD

5. **Documentation**
   - Review generated README files
   - Add custom documentation if needed

### Using Generated SDKs

1. **Install properly**
   - Python: Use `pip install -e .` for development
   - Go: Use `go mod` for dependency management

2. **Handle errors**
   - Always check error returns (Go)
   - Use try/except (Python)

3. **Authentication**
   - Store credentials securely
   - Use environment variables
   - Don't commit credentials

4. **Testing**
   - Run generated tests
   - Add integration tests
   - Test error scenarios

---

## Examples

### Example 1: Simple API SDK

**OpenAPI Schema (`simple-api.yaml`):**
```yaml
openapi: 3.0.0
info:
  title: Simple API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /health:
    get:
      operationId: checkHealth
      responses:
        '200':
          description: OK
```

**Generate SDK:**
```bash
sdk-forge generate \
  --schema simple-api.yaml \
  --lang python \
  --name simple-api \
  --output ./sdks
```

**Use SDK:**
```python
from simple_api import SimpleApi

client = SimpleApi(base_url="https://api.example.com/v1")
response = client.check_health()
print(response.json())
```

### Example 2: API with Authentication

**OpenAPI Schema (`auth-api.yaml`):**
```yaml
openapi: 3.0.0
info:
  title: Auth API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
paths:
  /users:
    get:
      operationId: listUsers
      security:
        - apiKey: []
      responses:
        '200':
          description: OK
```

**Generate SDK:**
```bash
sdk-forge generate \
  --schema auth-api.yaml \
  --lang python \
  --name auth-api \
  --output ./sdks
```

**Use SDK:**
```python
from auth_api import AuthApi

client = AuthApi(
    base_url="https://api.example.com/v1",
    apiKey="your-api-key"
)
users = client.list_users()
```

### Example 3: Multi-Language Generation

**Generate for all languages:**
```bash
sdk-forge generate \
  --schema api.yaml \
  --lang all \
  --name my-api \
  --output ./sdks \
  --sdk-version 2.0.0
```

**Result:**
```
./sdks/
├── python/
│   └── my-api/
└── go/
    └── my-api/
```

### Example 4: Custom Versions

**Generate with specific versions:**
```bash
# Python 3.14 SDK
sdk-forge generate \
  --schema api.yaml \
  --lang python \
  --python-version 3.14 \
  --name my-api \
  --output ./sdks

# Go 1.25 SDK
sdk-forge generate \
  --schema api.yaml \
  --lang go \
  --go-version 1.25 \
  --name my-api \
  --output ./sdks
```

---

## Additional Resources

- **Main README**: See `README.md` in the root directory
- **Feature List**: See `docs/README.md`
- **GitHub Repository**: https://github.com/vubon/sdk-forge
- **OpenAPI Specification**: https://swagger.io/specification/
- **Examples**: See `examples/` directory in the repository

---

## Support

For questions, issues, or contributions:

- **GitHub Issues**: https://github.com/vubon/sdk-forge/issues
- **Documentation**: See `docs/` directory
- **Examples**: See `examples/` directory

---

**Happy SDK Generating! 🚀**

