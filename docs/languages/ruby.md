# Ruby SDK Generation Guide

This guide covers Ruby SDK generation with SDK Forge.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Ruby Versions](#ruby-versions)
- [HTTP Libraries](#http-libraries)
- [Authentication](#authentication)
- [Retry Mechanism](#retry-mechanism)
- [Generated Structure](#generated-structure)
- [Usage Examples](#usage-examples)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Best Practices](#best-practices)

## Overview

SDK Forge generates production-ready Ruby SDKs from OpenAPI 3.x schemas with:

- ✅ Ruby 3.0+ support
- ✅ Multiple HTTP client libraries (Faraday, Net::HTTP, HTTP.rb)
- ✅ Full authentication support
- ✅ Automatic retry mechanism
- ✅ RSpec test generation
- ✅ RuboCop and YARD integration
- ✅ Bundler/Gem packaging

## Prerequisites

- **Ruby**: 3.0 or later
- **Bundler**: For dependency management
- **OpenAPI Schema**: 3.0.x or 3.1.x

## Quick Start

Generate a Ruby SDK:

```bash
sdk-forge generate \
  --schema openapi.yaml \
  --lang ruby \
  --name my_api_sdk \
  --output ./sdks
```

The generated SDK will be at: `./sdks/ruby/my_api_sdk/`

## Ruby Versions

SDK Forge supports Ruby versions 3.0 through 3.3:

### Available Versions

- Ruby 3.0 (default)
- Ruby 3.1
- Ruby 3.2
- Ruby 3.3

### Specifying Ruby Version

```bash
sdk-forge generate \
  --schema openapi.yaml \
  --lang ruby \
  --ruby-version 3.2 \
  --name my_api_sdk \
  --output ./sdks
```

Or use the short alias:

```bash
sdk-forge generate \
  --schema openapi.yaml \
  --lang ruby \
  --rb-version 3.2 \
  --name my_api_sdk \
  --output ./sdks
```

### Version-Specific Features

All supported Ruby versions include:

- Keyword arguments
- Pattern matching (3.0+)
- Endless methods (3.0+)
- Hash shorthand syntax (3.1+)
- Data class support (3.2+)

## HTTP Libraries

SDK Forge supports three HTTP client libraries for Ruby:

### Faraday (Default)

Modern, middleware-based HTTP client:

```bash
sdk-forge generate \
  --schema openapi.yaml \
  --lang ruby \
  --http-lib faraday \
  --name my_api_sdk \
  --output ./sdks
```

**Features:**
- Middleware support
- Adapter pattern
- Thread-safe
- Widely used

**Gemfile dependency:**
```ruby
gem 'faraday', '~> 2.0'
```

### Net::HTTP (Standard Library)

Built-in Ruby HTTP client (no external dependencies):

```bash
sdk-forge generate \
  --schema openapi.yaml \
  --lang ruby \
  --http-lib net-http \
  --name my_api_sdk \
  --output ./sdks
```

**Features:**
- No external dependencies
- Standard library
- Lightweight
- Suitable for simple use cases

### HTTP.rb

Clean, fast HTTP client:

```bash
sdk-forge generate \
  --schema openapi.yaml \
  --lang ruby \
  --http-lib httprb \
  --name my_api_sdk \
  --output ./sdks
```

**Features:**
- Chainable API
- Fast performance
- Clean syntax
- Modern design

**Gemfile dependency:**
```ruby
gem 'http', '~> 5.0'
```

## Authentication

SDK Forge automatically generates authentication support based on your OpenAPI schema's `securitySchemes`.

### API Key Authentication

**Header-based:**

```ruby
require 'my_api_sdk'

client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  api_key: 'your-api-key-here'
)
```

**Query parameter:**

The SDK automatically adds API keys to query parameters if defined in the schema.

### Bearer Token Authentication

```ruby
require 'my_api_sdk'

client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  bearer_token: 'your-jwt-token-here'
)
```

### Basic Authentication

```ruby
require 'my_api_sdk'

client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  username: 'your-username',
  password: 'your-password'
)
```

### OAuth2 Authentication

For OAuth2, obtain a token first and use bearer authentication:

```ruby
# Obtain OAuth2 token (using your OAuth2 flow)
access_token = get_oauth2_token(
  client_id: 'your-client-id',
  client_secret: 'your-client-secret'
)

# Use the token with the SDK
client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  bearer_token: access_token
)
```

## Retry Mechanism

SDK Forge generates automatic retry logic for transient failures.

### Enable Retry

```bash
sdk-forge generate \
  --schema openapi.yaml \
  --lang ruby \
  --name my_api_sdk \
  --retry-enabled \
  --retry-max-attempts 3 \
  --retry-strategy exponential \
  --output ./sdks
```

### Retry Strategies

**Exponential Backoff (Default):**

```ruby
client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  retry_enabled: true,
  retry_max_attempts: 3,
  retry_strategy: :exponential,
  retry_initial_delay: 1.0,
  retry_backoff_multiplier: 2.0
)
```

Delays: 1s, 2s, 4s, 8s...

**Linear Backoff:**

```ruby
client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  retry_strategy: :linear,
  retry_initial_delay: 2.0
)
```

Delays: 2s, 4s, 6s, 8s...

**Fixed Delay:**

```ruby
client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  retry_strategy: :fixed,
  retry_initial_delay: 5.0
)
```

Delays: 5s, 5s, 5s...

### Retryable Status Codes

Configure which HTTP status codes trigger retries:

```ruby
client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  retry_enabled: true,
  retry_status_codes: [429, 500, 502, 503, 504]
)
```

## Generated Structure

```
my_api_sdk/
├── my_api_sdk.gemspec          # Gem specification
├── Gemfile                      # Bundler configuration
├── README.md                    # SDK documentation
├── .rubocop.yml                 # RuboCop configuration
├── .yardopts                    # YARD documentation options
├── lib/
│   ├── my_api_sdk.rb           # Main entry point
│   └── my_api_sdk/
│       ├── version.rb          # SDK version
│       ├── client.rb           # HTTP client
│       ├── models/             # Data models
│       │   ├── user.rb
│       │   ├── pet.rb
│       │   └── ...
│       ├── api/                # API methods by tag
│       │   ├── users_api.rb
│       │   ├── pets_api.rb
│       │   └── ...
│       └── exceptions/         # Custom exceptions
│           └── api_exception.rb
├── spec/                       # RSpec tests
│   ├── spec_helper.rb
│   ├── client_spec.rb
│   ├── models/
│   │   └── *_spec.rb
│   └── api/
│       └── *_api_spec.rb
└── examples/                   # Usage examples
    └── basic_usage.rb
```

## Usage Examples

### Basic API Call

```ruby
require 'my_api_sdk'

# Initialize client
client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  bearer_token: 'your-token'
)

# Make API call
response = MyApiSdk::API::PetsApi.list_pets(client, limit: 10)
puts response
```

### Create a Resource

```ruby
require 'my_api_sdk'

client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  api_key: 'your-api-key'
)

# Create a new pet
pet_data = {
  name: 'Fluffy',
  species: 'cat',
  age: 3
}

new_pet = MyApiSdk::API::PetsApi.create_pet(client, body: pet_data)
puts "Created pet with ID: #{new_pet['id']}"
```

### Using Models

```ruby
require 'my_api_sdk'

# Create model instance
pet = MyApiSdk::Models::Pet.new(
  name: 'Buddy',
  species: 'dog',
  age: 5
)

# Convert to hash
pet_hash = pet.to_h
# => { name: "Buddy", species: "dog", age: 5 }

# Convert to JSON
pet_json = pet.to_json
# => '{"name":"Buddy","species":"dog","age":5}'

# Create from hash
pet_from_hash = MyApiSdk::Models::Pet.from_hash(pet_hash)

# Create from JSON
pet_from_json = MyApiSdk::Models::Pet.from_json(pet_json)
```

### Error Handling

```ruby
require 'my_api_sdk'

client = MyApiSdk::Client.new(base_url: 'https://api.example.com')

begin
  response = MyApiSdk::API::UsersApi.get_user(client, id: 123)
rescue MyApiSdk::Exceptions::ApiException => e
  puts "API Error: #{e.message}"
  puts "Status Code: #{e.status_code}"
  puts "Response Body: #{e.response_body}"
end
```

### With Retry Configuration

```ruby
require 'my_api_sdk'

client = MyApiSdk::Client.new(
  base_url: 'https://api.example.com',
  bearer_token: 'your-token',
  retry_enabled: true,
  retry_max_attempts: 5,
  retry_strategy: :exponential,
  retry_status_codes: [429, 500, 502, 503]
)

# API calls will automatically retry on transient failures
response = MyApiSdk::API::PetsApi.list_pets(client)
```

## Testing

### Running Tests

```bash
cd my_api_sdk

# Install dependencies
bundle install

# Run all tests
bundle exec rspec

# Run with coverage
bundle exec rspec --format documentation

# Run specific test file
bundle exec rspec spec/models/pet_spec.rb
```

### Test Structure

The generated SDK includes comprehensive RSpec tests:

- **Client tests**: Authentication, request handling
- **Model tests**: Serialization, deserialization, validation
- **API tests**: Method calls with mocked responses
- **Error handling tests**: Exception scenarios

### WebMock Integration

Tests use WebMock for HTTP mocking:

```ruby
# spec/api/pets_api_spec.rb
RSpec.describe MyApiSdk::API::PetsApi do
  let(:client) { MyApiSdk::Client.new(base_url: 'https://api.example.com') }

  it 'lists pets' do
    stub_request(:get, 'https://api.example.com/pets')
      .to_return(
        status: 200,
        body: '[{"id":1,"name":"Fluffy"}]',
        headers: { 'Content-Type' => 'application/json' }
      )

    response = described_class.list_pets(client)
    expect(response).to be_a(Array)
  end
end
```

## Code Quality

### RuboCop

Run RuboCop for code style checks:

```bash
cd my_api_sdk

# Check code style
bundle exec rubocop

# Auto-fix issues
bundle exec rubocop --auto-correct

# Generate TODO list
bundle exec rubocop --auto-gen-config
```

### YARD Documentation

Generate and view documentation:

```bash
cd my_api_sdk

# Generate documentation
bundle exec yard doc

# Start documentation server
bundle exec yard server

# Open http://localhost:8808 in browser
```

### Building the Gem

```bash
cd my_api_sdk

# Build gem
gem build my_api_sdk.gemspec

# Install locally
gem install my_api_sdk-1.0.0.gem

# Publish to RubyGems (optional)
gem push my_api_sdk-1.0.0.gem
```

## Best Practices

### 1. Use Keyword Arguments

The SDK uses keyword arguments throughout:

```ruby
# Good
MyApiSdk::API::PetsApi.list_pets(client, limit: 10, offset: 0)

# Avoid positional arguments
```

### 2. Handle Errors Gracefully

```ruby
begin
  response = MyApiSdk::API::UsersApi.get_user(client, id: user_id)
rescue MyApiSdk::Exceptions::ApiException => e
  case e.status_code
  when 404
    puts "User not found"
  when 401
    puts "Authentication failed"
  else
    puts "Error: #{e.message}"
  end
end
```

### 3. Reuse Client Instances

```ruby
# Create once
@client = MyApiSdk::Client.new(
  base_url: ENV['API_BASE_URL'],
  bearer_token: ENV['API_TOKEN']
)

# Reuse for multiple calls
pets = MyApiSdk::API::PetsApi.list_pets(@client)
users = MyApiSdk::API::UsersApi.list_users(@client)
```

### 4. Use Environment Variables

```ruby
# .env file
API_BASE_URL=https://api.example.com
API_KEY=your-api-key-here

# In your code
require 'dotenv/load'
require 'my_api_sdk'

client = MyApiSdk::Client.new(
  base_url: ENV['API_BASE_URL'],
  api_key: ENV['API_KEY']
)
```

### 5. Enable Retry for Production

```ruby
client = MyApiSdk::Client.new(
  base_url: ENV['API_BASE_URL'],
  bearer_token: ENV['API_TOKEN'],
  retry_enabled: true,
  retry_max_attempts: 3,
  retry_strategy: :exponential
)
```

### 6. Log API Calls

```ruby
require 'logger'

logger = Logger.new(STDOUT)

begin
  logger.info "Calling API: list_pets"
  response = MyApiSdk::API::PetsApi.list_pets(client, limit: 10)
  logger.info "Response received: #{response.length} pets"
rescue MyApiSdk::Exceptions::ApiException => e
  logger.error "API Error: #{e.message} (#{e.status_code})"
  raise
end
```

### 7. Use Frozen String Literals

All generated files use frozen string literals:

```ruby
# frozen_string_literal: true

# Your application code should too
```

## Troubleshooting

### Gem Installation Issues

```bash
# Update bundler
gem update bundler

# Install with verbose output
bundle install --verbose

# Clear gem cache
rm -rf vendor/bundle
bundle install
```

### HTTP Client Issues

**Faraday:**
```bash
gem install faraday -v '~> 2.0'
```

**HTTP.rb:**
```bash
gem install http -v '~> 5.0'
```

### Test Failures

```bash
# Clean and reinstall
rm -rf vendor/bundle
bundle install
bundle exec rspec
```

### Documentation Generation Issues

```bash
# Install YARD
gem install yard

# Generate docs
bundle exec yard doc --verbose
```

## Additional Resources

- [Ruby Documentation](https://docs.ruby-lang.org/)
- [Bundler Guide](https://bundler.io/guides/creating_gem.html)
- [RSpec Documentation](https://rspec.info/)
- [RuboCop Documentation](https://docs.rubocop.org/)
- [YARD Documentation](https://yardoc.org/)
- [Faraday Documentation](https://lostisland.github.io/faraday/)
- [HTTP.rb Documentation](https://github.com/httprb/http)

## Support

For issues, questions, or contributions:

- GitHub Issues: [sdk-forge/issues](https://github.com/vubon/sdk-forge/issues)
- Documentation: [docs/](../../README.md)
- Examples: [examples/](../../examples/)

