# Changelog

All notable changes to SDK Forge will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2025-12-23

### Added
- **TypeScript/JavaScript SDK Generation**: Full support for TypeScript/JavaScript SDK generation
  - TypeScript versions: 4.9, 5.0, 5.1, 5.2, 5.3, 5.4, 5.5 (Default: 5.0)
  - HTTP Libraries: axios (default), fetch, node-fetch, ky
  - Dual package exports: ESM (ES Modules) and CommonJS support
  - Full TypeScript type safety with comprehensive type definitions
  - Complete authentication support (API Key, Bearer, Basic, OAuth2, OpenID Connect, Mutual TLS)
  - Retry mechanism support with all strategies (exponential, linear, fixed)
  - Automatic test generation with Jest and ts-jest
  - Code quality tools: ESLint and Prettier configuration files
  - TypeScript compiler configuration (tsconfig.json) with strict type checking
  - Package.json generation with proper dependencies and scripts
  - Comprehensive README and examples generation
  - Manual integration test support with Prism mock server
  - HTTP library compatibility testing for all supported clients
- **TypeScript Version CLI Flag**: `--typescript-version` flag for specifying target TypeScript version
- **TypeScript Interactive Mode**: Interactive prompts for TypeScript version selection
- **TypeScript Language Validation**: Added typescript/javascript to supported languages
- **TypeScript Documentation**: Complete TypeScript/JavaScript SDK guide in docs/languages/typescript.md
- **TypeScript Implementation Details**:
  - ESM module resolution with `.js` extensions for Node.js compatibility
  - Jest ESM configuration with jest.config.cjs and ts-jest
  - Retry mechanism with separate `executeRequest` method to prevent recursion
  - Proper TypeScript interface and class field separation
  - Re-export pattern for default exports
  - NetworkException with error chaining support
  - Complete RetryConfig interface with backoffMultiplier
  - Object parameter syntax for API method calls (e.g., `getPetById({ id: 1 })`)

### Changed
- **Version**: Updated from 0.5.0 to 0.6.0
- **README**: Updated to reflect TypeScript/JavaScript SDK support and current project structure
- **Documentation**: Updated MANUAL.md with TypeScript examples, CLI flags, and language-specific guide links
- **Project Structure**: Updated documentation to reflect new directory structure with language-specific generator folders

## [0.5.0] - 2025-12-22

### Added
- **PHP SDK Generation**: Full support for PHP SDK generation
  - PHP versions: 8.0, 8.1, 8.2, 8.3 (Default: 8.1)
  - HTTP Library: Guzzle (default)
  - PSR-4 autoloading support with Composer
  - Full authentication support (API Key, Bearer, Basic, OAuth2, OpenID Connect)
  - Retry mechanism support with all strategies (exponential, linear, fixed)
  - Automatic test generation with PHPUnit
  - Code quality tools: PHP-CS-Fixer, PHPStan, PHP_CodeSniffer
  - Comprehensive README and examples generation
  - Manual integration test support

### Fixed
- **PHP PSR-4 Autoloading**: Fixed client class filename to match class name (e.g., `Petstore.php` instead of `PetstoreClient.php`)
- **PHP ApiException Namespace**: Fixed hardcoded namespace placeholder to use actual SDK name
- **PHP Composer.json**: Fixed namespace escaping for proper JSON formatting with PSR-4 autoloading

## [0.4.0] - 2025-12-22

### Added
- **Retry Mechanism**: Configurable automatic retry logic for generated SDKs
  - Support for exponential, linear, and fixed backoff strategies
  - Configurable retry attempts, delays, and retryable status codes
  - Network error retry support
  - CLI flags: `--retry-enabled`, `--retry-max-attempts`, `--retry-strategy`, `--retry-initial-delay`, `--retry-max-delay`, `--retry-backoff-multiplier`, `--retry-status-codes`
  - OpenAPI extension support via `x-sdk-forge-retry` for schema-level configuration
  - Full support for Python SDKs (requests, httpx, aiohttp, urllib3)
  - Full support for Go SDKs (nethttp, resty, gentleman)
  - Comprehensive retry configuration structure with sensible defaults
  - Retry delay calculation methods for all strategies
  - Retryable status code checking
  - Integration with existing HTTP request flow in generated SDKs
- Comprehensive unit tests for retry mechanism
- CLI integration tests for retry configuration parsing
- Manual test scripts for retry mechanism verification (Python and Go)
- Documentation for retry configuration in User Manual

### Changed
- Default retry configuration: disabled by default (backward compatible)
- When enabled, defaults to exponential backoff with 3 max attempts
- Retryable status codes default: 429, 500, 502, 503, 504
- Network error retries enabled by default when retry is enabled

## [0.3.0] - 2025-12-19

### Added
- Complete infrastructure setup and project documentation
  - CI/CD pipeline (GitHub Actions) for pull requests targeting main branch
  - Automated release workflow for building and publishing releases
  - CONTRIBUTING.md with comprehensive development workflow and guidelines
  - CODE_OF_CONDUCT.md following Contributor Covenant 2.1
  - GitHub issue templates (bug report, feature request, question)
  - Pull request template with comprehensive checklist
  - Dockerfile with multi-stage build for containerized usage
  - docker-compose.yml for easy Docker usage
  - .dockerignore to optimize Docker builds
- Complete authentication test coverage for all 7 authentication schemes
- Common utilities refactoring (`common.go`) for shared functionality across language generators
- Support for Digest and Mutual TLS authentication test generation

### Fixed
- Go SDK path parameter formatting issues
- Go SDK body parameter handling for methods without request bodies
- Python SDK dataclass field ordering (required fields before optional fields)
- Python SDK setup.py description escaping for multi-line descriptions
- Go SDK import management (conditional imports only when needed)
- Go SDK comment formatting for multi-line descriptions

### Changed
- Extracted common utilities (`getClientClassName`, `groupOperationsByTag`, `determineSDKVersion`) to shared `common.go` file
- Improved code reusability for future language support
- Updated README.md with Docker installation instructions

## [0.3.0-alpha.2] - 2025-12-19

### Added
- Complete authentication test coverage for all 7 authentication schemes
- Common utilities refactoring (`common.go`) for shared functionality across language generators
- Support for Digest and Mutual TLS authentication test generation

### Fixed
- Go SDK path parameter formatting issues
- Go SDK body parameter handling for methods without request bodies
- Python SDK dataclass field ordering (required fields before optional fields)
- Python SDK setup.py description escaping for multi-line descriptions
- Go SDK import management (conditional imports only when needed)
- Go SDK comment formatting for multi-line descriptions

### Changed
- Extracted common utilities (`getClientClassName`, `groupOperationsByTag`, `determineSDKVersion`) to shared `common.go` file
- Improved code reusability for future language support

## [0.3.0-alpha.1] - 2025-12-18

### Added
- Automatic test generation for Python and Go SDKs
- Complete test suite generation:
  - Client initialization tests
  - Model tests (instantiation, serialization, validation)
  - API method tests with mocked HTTP responses
  - Authentication tests for all security schemes
  - Error handling tests (4xx, 5xx responses)
  - Test fixtures automatically extracted from OpenAPI examples
- Python pytest-based test generation
- Go standard library testing package test generation
- Test data extraction from OpenAPI schema examples

### Changed
- Tests are now generated by default (use `--skip-tests` to disable)
- Enhanced test coverage for generated SDKs

## [0.2.0-alpha.1] - 2025-12-17

### Added
- Initial Go SDK generation
- Python SDK generation
- Template-based code generation system
- OpenAPI 3.0.x and 3.1.x schema support
- Authentication support:
  - API Key (query, header, cookie)
  - HTTP Basic authentication
  - HTTP Bearer token authentication
  - OAuth2 (all flows)
  - OpenID Connect
  - Mutual TLS (mTLS)
- Language version configuration (`--go-version`, `--python-version`)
- SDK version management with priority: OpenAPI schema → CLI → Default
- Code formatting integration (gofmt, black/autopep8)
- Interactive CLI mode
- HTTP library selection per language
- Batch generation for all languages (`--lang all`)

### Changed
- Improved error handling and validation
- Enhanced OpenAPI schema parsing

---

## Version Format

- **MAJOR.MINOR.PATCH-PRERELEASE**
- **0.x.x**: Initial development (API may change)
- **-alpha.x**: Alpha releases (unstable, for testing)
- **-beta.x**: Beta releases (feature complete, testing)
- **-rc.x**: Release candidates (stable, final testing)
- **x.y.z**: Stable releases (production ready)

[0.6.0]: https://github.com/vubon/sdk-forge/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/vubon/sdk-forge/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/vubon/sdk-forge/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/vubon/sdk-forge/compare/v0.3.0-alpha.2...v0.3.0
[0.3.0-alpha.2]: https://github.com/vubon/sdk-forge/compare/v0.3.0-alpha.1...v0.3.0-alpha.2
[0.3.0-alpha.1]: https://github.com/vubon/sdk-forge/compare/v0.2.0-alpha.1...v0.3.0-alpha.1
[0.2.0-alpha.1]: https://github.com/vubon/sdk-forge/releases/tag/v0.2.0-alpha.1

