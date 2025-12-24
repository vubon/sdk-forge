package ruby

import (
	"fmt"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateRubyClient generates the Ruby client class
func generateRubyClient(data common.TemplateData, _ /* version */ common.LanguageVersion) (string, error) {
	sanitizedName := data.SDKName
	moduleName := common.ToPascalCase(sanitizedName)
	clientClassName := common.GetClientClassName(data.SDKName)

	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	if !ok {
		extractedData = nil
	}

	var buf strings.Builder

	// File header
	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString(fmt.Sprintf("require '%s'\n", data.HTTPLibImport))
	buf.WriteString("require 'json'\n\n")

	// Module and class definition
	buf.WriteString(fmt.Sprintf("module %s\n", moduleName))
	buf.WriteString("  # Main API client for SDK\n")
	buf.WriteString(fmt.Sprintf("  class %s\n", clientClassName))
	buf.WriteString("    attr_reader :base_url, :http_client\n")
	buf.WriteString("    attr_accessor :api_key, :bearer_token, :username, :password\n\n")

	// Retry configuration attributes
	if data.RetryConfig.Enabled {
		buf.WriteString("    attr_accessor :retry_enabled, :retry_max_attempts, :retry_strategy\n")
		buf.WriteString("    attr_accessor :retry_initial_delay, :retry_max_delay, :retry_backoff_multiplier\n")
		buf.WriteString("    attr_accessor :retry_status_codes\n\n")
	}

	// Initialize method
	buf.WriteString("    # Initialize the API client\n")
	buf.WriteString("    #\n")
	buf.WriteString("    # @param base_url [String] The base URL for the API\n")
	buf.WriteString("    # @param options [Hash] Additional options\n")
	buf.WriteString("    # @option options [String] :api_key API key for authentication\n")
	buf.WriteString("    # @option options [String] :bearer_token Bearer token for authentication\n")
	buf.WriteString("    # @option options [String] :username Username for basic authentication\n")
	buf.WriteString("    # @option options [String] :password Password for basic authentication\n")
	if data.RetryConfig.Enabled {
		buf.WriteString("    # @option options [Boolean] :retry_enabled Enable retry mechanism (default: true)\n")
		buf.WriteString(fmt.Sprintf("    # @option options [Integer] :retry_max_attempts Maximum retry attempts (default: %d)\n", data.RetryConfig.MaxAttempts))
		buf.WriteString(fmt.Sprintf("    # @option options [Symbol] :retry_strategy Retry strategy (:exponential, :linear, :fixed) (default: :%s)\n", data.RetryConfig.Strategy))
	}
	buf.WriteString("    def initialize(base_url:, **options)\n")
	buf.WriteString("      @base_url = base_url.chomp('/')\n")

	// HTTP client initialization based on library
	switch data.HTTPLib {
	case "faraday":
		buf.WriteString("      @http_client = Faraday.new(url: @base_url) do |f|\n")
		buf.WriteString("        f.adapter Faraday.default_adapter\n")
		buf.WriteString("      end\n")
	case "net-http":
		buf.WriteString("      @http_client = Net::HTTP\n")
	case "httprb":
		buf.WriteString("      @http_client = HTTP\n")
	default:
		buf.WriteString("      @http_client = Faraday.new(url: @base_url)\n")
	}

	// Authentication setup
	buf.WriteString("\n      # Authentication\n")
	buf.WriteString("      @api_key = options[:api_key]\n")
	buf.WriteString("      @bearer_token = options[:bearer_token]\n")
	buf.WriteString("      @username = options[:username]\n")
	buf.WriteString("      @password = options[:password]\n")

	// Retry configuration setup
	if data.RetryConfig.Enabled {
		buf.WriteString("\n      # Retry configuration\n")
		buf.WriteString(fmt.Sprintf("      @retry_enabled = options.fetch(:retry_enabled, %t)\n", data.RetryConfig.Enabled))
		buf.WriteString(fmt.Sprintf("      @retry_max_attempts = options.fetch(:retry_max_attempts, %d)\n", data.RetryConfig.MaxAttempts))
		buf.WriteString(fmt.Sprintf("      @retry_strategy = options.fetch(:retry_strategy, :%s)\n", data.RetryConfig.Strategy))
		buf.WriteString(fmt.Sprintf("      @retry_initial_delay = options.fetch(:retry_initial_delay, %.1f)\n", data.RetryConfig.InitialDelay.Seconds()))
		buf.WriteString(fmt.Sprintf("      @retry_max_delay = options.fetch(:retry_max_delay, %.1f)\n", data.RetryConfig.MaxDelay.Seconds()))
		buf.WriteString(fmt.Sprintf("      @retry_backoff_multiplier = options.fetch(:retry_backoff_multiplier, %.1f)\n", data.RetryConfig.BackoffMultiplier))

		// Retry status codes
		buf.WriteString("      @retry_status_codes = options.fetch(:retry_status_codes, [")
		for i, code := range data.RetryConfig.RetryableStatusCodes {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%d", code))
		}
		buf.WriteString("])\n")
	}

	buf.WriteString("    end\n\n")

	// Request method
	buf.WriteString("    # Make an HTTP request\n")
	buf.WriteString("    #\n")
	buf.WriteString("    # @param method [Symbol] HTTP method (:get, :post, :put, :delete, etc.)\n")
	buf.WriteString("    # @param path [String] Request path\n")
	buf.WriteString("    # @param params [Hash] Query parameters\n")
	buf.WriteString("    # @param body [Hash, nil] Request body\n")
	buf.WriteString("    # @param headers [Hash] Additional headers\n")
	buf.WriteString("    # @return [Hash] Parsed response body\n")
	buf.WriteString("    # @raise [Exceptions::ApiException] If request fails\n")
	buf.WriteString("    def request(method, path, params: {}, body: nil, headers: {})\n")

	if data.RetryConfig.Enabled {
		buf.WriteString("      if @retry_enabled\n")
		buf.WriteString("        request_with_retry(method, path, params: params, body: body, headers: headers)\n")
		buf.WriteString("      else\n")
		buf.WriteString("        execute_request(method, path, params: params, body: body, headers: headers)\n")
		buf.WriteString("      end\n")
	} else {
		buf.WriteString("      execute_request(method, path, params: params, body: body, headers: headers)\n")
	}

	buf.WriteString("    end\n\n")

	// Execute request method
	buf.WriteString("    private\n\n")
	buf.WriteString("    def execute_request(method, path, params: {}, body: nil, headers: {})\n")
	buf.WriteString("      url = @base_url + path\n")
	buf.WriteString("      headers = build_headers(headers)\n\n")

	// HTTP client-specific request implementation
	switch data.HTTPLib {
	case "faraday":
		buf.WriteString("      response = @http_client.run_request(method, path, body&.to_json, headers) do |req|\n")
		buf.WriteString("        req.params = params unless params.empty?\n")
		buf.WriteString("      end\n\n")
		buf.WriteString("      handle_response(response)\n")
	case "net-http":
		buf.WriteString("      uri = URI.parse(url)\n")
		buf.WriteString("      uri.query = URI.encode_www_form(params) unless params.empty?\n\n")
		buf.WriteString("      http = Net::HTTP.new(uri.host, uri.port)\n")
		buf.WriteString("      http.use_ssl = uri.scheme == 'https'\n\n")
		buf.WriteString("      request = case method\n")
		buf.WriteString("                when :get then Net::HTTP::Get.new(uri)\n")
		buf.WriteString("                when :post then Net::HTTP::Post.new(uri)\n")
		buf.WriteString("                when :put then Net::HTTP::Put.new(uri)\n")
		buf.WriteString("                when :delete then Net::HTTP::Delete.new(uri)\n")
		buf.WriteString("                else raise ArgumentError, \"Unsupported HTTP method: #{method}\"\n")
		buf.WriteString("                end\n\n")
		buf.WriteString("      headers.each { |k, v| request[k] = v }\n")
		buf.WriteString("      request.body = body.to_json if body\n\n")
		buf.WriteString("      response = http.request(request)\n")
		buf.WriteString("      handle_response(response)\n")
	default:
		buf.WriteString("      # Default HTTP client implementation\n")
		buf.WriteString("      raise 'HTTP client not implemented'\n")
	}

	buf.WriteString("    end\n\n")

	// Build headers method
	buf.WriteString("    def build_headers(additional_headers = {})\n")
	buf.WriteString("      headers = {\n")
	buf.WriteString("        'Content-Type' => 'application/json',\n")
	buf.WriteString("        'Accept' => 'application/json'\n")
	buf.WriteString("      }\n\n")

	// Add authentication headers
	if extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		buf.WriteString("      # Authentication headers\n")
		for _, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case "apiKey":
				if scheme.In == "header" {
					buf.WriteString(fmt.Sprintf("      headers['%s'] = @api_key if @api_key\n", scheme.Name))
				}
			case "http":
				switch scheme.Scheme {
				case "bearer":
					buf.WriteString("      headers['Authorization'] = \"Bearer #{@bearer_token}\" if @bearer_token\n")
				case "basic":
					buf.WriteString("      if @username && @password\n")
					buf.WriteString("        credentials = Base64.strict_encode64(\"#{@username}:#{@password}\")\n")
					buf.WriteString("        headers['Authorization'] = \"Basic #{credentials}\"\n")
					buf.WriteString("      end\n")
				}
			}
			break // Only first scheme for now
		}
	} else {
		buf.WriteString("      headers['Authorization'] = \"Bearer #{@bearer_token}\" if @bearer_token\n")
	}

	buf.WriteString("\n      headers.merge(additional_headers)\n")
	buf.WriteString("    end\n\n")

	// Handle response method
	buf.WriteString("    def handle_response(response)\n")
	switch data.HTTPLib {
	case "faraday":
		buf.WriteString("      if response.success?\n")
		buf.WriteString("        response.body.empty? ? {} : JSON.parse(response.body)\n")
		buf.WriteString("      else\n")
		buf.WriteString("        raise Exceptions::ApiException.new(\n")
		buf.WriteString("          \"API request failed: #{response.status}\",\n")
		buf.WriteString("          status_code: response.status,\n")
		buf.WriteString("          response_body: response.body\n")
		buf.WriteString("        )\n")
		buf.WriteString("      end\n")
	case "net-http":
		buf.WriteString("      case response\n")
		buf.WriteString("      when Net::HTTPSuccess\n")
		buf.WriteString("        response.body.empty? ? {} : JSON.parse(response.body)\n")
		buf.WriteString("      else\n")
		buf.WriteString("        raise Exceptions::ApiException.new(\n")
		buf.WriteString("          \"API request failed: #{response.code}\",\n")
		buf.WriteString("          status_code: response.code.to_i,\n")
		buf.WriteString("          response_body: response.body\n")
		buf.WriteString("        )\n")
		buf.WriteString("      end\n")
	}
	buf.WriteString("    end\n")

	// Retry mechanism methods
	if data.RetryConfig.Enabled {
		buf.WriteString("\n    def request_with_retry(method, path, params: {}, body: nil, headers: {})\n")
		buf.WriteString("      attempt = 0\n")
		buf.WriteString("      last_error = nil\n\n")
		buf.WriteString("      while attempt < @retry_max_attempts\n")
		buf.WriteString("        begin\n")
		buf.WriteString("          return execute_request(method, path, params: params, body: body, headers: headers)\n")
		buf.WriteString("        rescue Exceptions::ApiException => e\n")
		buf.WriteString("          last_error = e\n")
		buf.WriteString("          attempt += 1\n\n")
		buf.WriteString("          break if attempt >= @retry_max_attempts\n")
		buf.WriteString("          break unless retryable_error?(e)\n\n")
		buf.WriteString("          delay = calculate_delay(attempt)\n")
		buf.WriteString("          sleep(delay)\n")
		buf.WriteString("        rescue StandardError => e\n")
		buf.WriteString("          last_error = e\n")
		buf.WriteString("          attempt += 1\n\n")
		buf.WriteString("          break if attempt >= @retry_max_attempts\n\n")
		buf.WriteString("          delay = calculate_delay(attempt)\n")
		buf.WriteString("          sleep(delay)\n")
		buf.WriteString("        end\n")
		buf.WriteString("      end\n\n")
		buf.WriteString("      raise last_error\n")
		buf.WriteString("    end\n\n")

		buf.WriteString("    def retryable_error?(error)\n")
		buf.WriteString("      return false unless error.is_a?(Exceptions::ApiException)\n")
		buf.WriteString("      return false unless error.status_code\n\n")
		buf.WriteString("      @retry_status_codes.include?(error.status_code)\n")
		buf.WriteString("    end\n\n")

		buf.WriteString("    def calculate_delay(attempt)\n")
		buf.WriteString("      delay = case @retry_strategy\n")
		buf.WriteString("              when :exponential\n")
		buf.WriteString("                @retry_initial_delay * (@retry_backoff_multiplier ** (attempt - 1))\n")
		buf.WriteString("              when :linear\n")
		buf.WriteString("                @retry_initial_delay * attempt\n")
		buf.WriteString("              when :fixed\n")
		buf.WriteString("                @retry_initial_delay\n")
		buf.WriteString("              else\n")
		buf.WriteString("                @retry_initial_delay\n")
		buf.WriteString("              end\n\n")
		buf.WriteString("      [delay, @retry_max_delay].min\n")
		buf.WriteString("    end\n")
	}

	buf.WriteString("  end\n")
	buf.WriteString("end\n")

	return buf.String(), nil
}
