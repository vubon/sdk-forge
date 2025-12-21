# SDK Forge - User Manual

**Version**: 0.3.0  
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

**4. Generate SDKs for all languages**

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
| `--lang` | `-l` | Target language (`python`, `go`, `all`) | `--lang python` |
| `--name` | `-n` | Name for the generated SDK | `--name my-sdk` |
| `--output` | `-o` | Output directory for generated SDK | `--output ./sdks` |

### Optional Flags

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--http-lib` | HTTP library to use | Language default | `--http-lib httpx` |
| `--go-version` | Go version (1.24, 1.25) | 1.24 | `--go-version 1.25` |
| `--python-version` | Python version (3.11-3.14) | 3.11 | `--python-version 3.14` |
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

- **Options**: `python`, `go`, `all`
- **Future**: `php`, `js`, `ts`, `ruby`
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
└── go/
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
    all

? Enter SDK name: my-api-sdk

? Enter output directory: ./sdks

? Go version (1.24, 1.25) [1.24]: 

? Python version (3.11, 3.12, 3.13, 3.14) [3.11]: 

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
- Single language: `--lang python` or `--lang go`
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

---

## Language-Specific Guides

### Python SDK Guide

#### Installation

```bash
cd ./sdks/python/my-api-sdk
pip install -e .
```

#### Basic Usage

```python
from my_api_sdk import MyApiSdk

# Initialize client
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    apiKey="your-api-key"  # If using API key auth
)

# Make API calls
response = client.list_users()
users = response.json()
```

#### Authentication

**API Key:**
```python
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    apiKey="your-api-key"
)
```

**Bearer Token:**
```python
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    bearer_token="your-token"
)
```

**Basic Auth:**
```python
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    username="user",
    password="pass"
)
```

**OAuth2:**
```python
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    oauth2_token="your-oauth-token"
)
```

**OpenID Connect:**
```python
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    openIdConnect_token="your-openid-token"
)
```

**Digest Authentication:**
```python
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    username="user",
    password="pass"
)
# Digest auth is automatically used when both username and password are provided
# and the OpenAPI schema specifies digest authentication
```

**Mutual TLS (mTLS):**
```python
client = MyApiSdk(
    base_url="https://api.example.com/v1",
    # mTLS is configured via client certificates
    # Requires additional SSL/TLS configuration
)
```

#### Running Tests

```bash
cd ./sdks/python/my-api-sdk
pip install -e ".[dev]"
pytest tests/
```

### Go SDK Guide

#### Installation

```bash
cd ./sdks/go/my-api-client
go mod download
```

#### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/example/my-api-client"
)

func main() {
    // Initialize client
    client := myapiclient.NewMyApiClient("https://api.example.com/v1")
    
    // Set authentication
    client.ApiKey = "your-api-key"
    
    // Make API calls
    data, err := client.ListUsers()
    if err != nil {
        panic(err)
    }
    
    fmt.Println(string(data))
}
```

#### Authentication

**API Key:**
```go
client := myapiclient.NewMyApiClient("https://api.example.com/v1")
client.ApiKey = "your-api-key"
```

**Bearer Token:**
```go
client := myapiclient.NewMyApiClient("https://api.example.com/v1")
client.BearerToken = "your-token"
```

**Basic Auth:**
```go
client := myapiclient.NewMyApiClient("https://api.example.com/v1")
client.Username = "user"
client.Password = "pass"
```

**OAuth2:**
```go
client := myapiclient.NewMyApiClient("https://api.example.com/v1")
client.OAuth2Token = "your-oauth-token"
```

**OpenID Connect:**
```go
client := myapiclient.NewMyApiClient("https://api.example.com/v1")
client.OpenIdConnectToken = "your-openid-token"
```

**Digest Authentication:**
```go
client := myapiclient.NewMyApiClient("https://api.example.com/v1")
client.Username = "user"
client.Password = "pass"
// Digest auth is automatically used when both username and password are provided
// and the OpenAPI schema specifies digest authentication
```

**Mutual TLS (mTLS):**
```go
client := myapiclient.NewMyApiClient("https://api.example.com/v1")
// mTLS is configured via client certificates
// Requires additional SSL/TLS configuration
```

#### Running Tests

```bash
cd ./sdks/go/my-api-client
go test ./...
```

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
- Check supported languages: `python`, `go`, `all`
- Future languages (php, js, ts) coming soon

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

