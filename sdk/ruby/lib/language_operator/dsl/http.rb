# frozen_string_literal: true

require 'net/http'
require 'uri'
require 'json'

module LanguageOperator
  module Dsl
    # HTTP client helper for MCP tools
    #
    # Provides convenient methods for making HTTP requests from within tools.
    # Supports GET, POST, PUT, DELETE, HEAD with automatic JSON parsing.
    #
    # @example Making a GET request
    #   result = HTTP.get('https://api.example.com/users')
    #   if result[:success]
    #     users = result[:json]
    #   end
    class HTTP
      # Perform a GET request
      def self.get(url, headers: {}, follow_redirects: true, timeout: 30)
        uri = parse_uri(url)
        return { error: "Invalid URL: #{url}" } unless uri

        http = build_http(uri, timeout: timeout)
        request = Net::HTTP::Get.new(uri)
        add_headers(request, headers)

        execute_request(http, request, follow_redirects: follow_redirects)
      end

      # Perform a POST request
      def self.post(url, body: nil, json: nil, headers: {}, auth: nil, timeout: 30)
        uri = parse_uri(url)
        return { error: "Invalid URL: #{url}" } unless uri

        http = build_http(uri, timeout: timeout)
        request = Net::HTTP::Post.new(uri)

        # Set body
        if json
          request.body = json.to_json
          request['Content-Type'] = 'application/json'
        elsif body
          request.body = body
        end

        add_headers(request, headers)
        add_auth(request, auth) if auth

        execute_request(http, request)
      end

      # Perform a PUT request
      def self.put(url, body: nil, json: nil, headers: {}, auth: nil, timeout: 30)
        uri = parse_uri(url)
        return { error: "Invalid URL: #{url}" } unless uri

        http = build_http(uri, timeout: timeout)
        request = Net::HTTP::Put.new(uri)

        if json
          request.body = json.to_json
          request['Content-Type'] = 'application/json'
        elsif body
          request.body = body
        end

        add_headers(request, headers)
        add_auth(request, auth) if auth

        execute_request(http, request)
      end

      # Perform a DELETE request
      def self.delete(url, headers: {}, auth: nil, timeout: 30)
        uri = parse_uri(url)
        return { error: "Invalid URL: #{url}" } unless uri

        http = build_http(uri, timeout: timeout)
        request = Net::HTTP::Delete.new(uri)

        add_headers(request, headers)
        add_auth(request, auth) if auth

        execute_request(http, request)
      end

      # Get just the headers from a URL
      def self.head(url, headers: {}, timeout: 30)
        uri = parse_uri(url)
        return { error: "Invalid URL: #{url}" } unless uri

        http = build_http(uri, timeout: timeout)
        request = Net::HTTP::Head.new(uri)
        add_headers(request, headers)

        execute_request(http, request)
      end

      # Wrapper for curl commands (when you need curl-specific features)
      def self.curl(url, options: [])
        require 'shellwords'

        cmd_parts = ['curl', '-s']
        cmd_parts.concat(options)
        cmd_parts << Shellwords.escape(url)

        output = `#{cmd_parts.join(' ')}`

        {
          success: $?.success?,
          output: output,
          exitcode: $?.exitstatus
        }
      end

      class << self
        private

        def parse_uri(url)
          URI.parse(url)
        rescue URI::InvalidURIError
          nil
        end

        def build_http(uri, timeout: 30)
          http = Net::HTTP.new(uri.host, uri.port)
          http.use_ssl = (uri.scheme == 'https')
          http.open_timeout = timeout
          http.read_timeout = timeout
          http
        end

        def add_headers(request, headers)
          # Add default user agent if not provided
          request['User-Agent'] ||= 'Langop-SDK/1.0'

          headers.each do |key, value|
            request[key] = value
          end
        end

        def add_auth(request, auth)
          case auth[:type]
          when :basic
            request.basic_auth(auth[:username], auth[:password])
          when :bearer
            request['Authorization'] = "Bearer #{auth[:token]}"
          when :token
            request['Authorization'] = "Token #{auth[:token]}"
          end
        end

        def execute_request(http, request, follow_redirects: false)
          response = http.request(request)

          # Handle redirects
          if follow_redirects && response.is_a?(Net::HTTPRedirection)
            location = response['location']
            return get(location, headers: request.to_hash, follow_redirects: true)
          end

          # Parse response
          result = {
            status: response.code.to_i,
            headers: response.to_hash,
            body: response.body,
            success: response.is_a?(Net::HTTPSuccess)
          }

          # Try to parse JSON if content-type indicates JSON
          if response['content-type']&.include?('application/json')
            begin
              result[:json] = JSON.parse(response.body)
            rescue JSON::ParserError
              # Body is not valid JSON, leave as string
            end
          end

          result
        rescue StandardError => e
          {
            error: e.message,
            success: false
          }
        end
      end
    end
  end
end
