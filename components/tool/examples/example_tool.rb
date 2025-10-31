# Example tool demonstrating all the new helper features

# Example 1: Using built-in parameter validators
tool "send_message" do
  description "Send a message to an email or phone number"

  parameter "recipient" do
    type :string
    required true
    description "Email address or phone number"
    # You can use built-in validators
    # email_format or phone_format
  end

  parameter "message" do
    type :string
    required true
    description "The message to send"
  end

  execute do |params|
    recipient = params["recipient"]
    message = params["message"]

    # Use validation helpers
    if recipient.include?('@')
      error = validate_email(recipient)
      return error if error
      "Email sent to #{recipient}: #{message}"
    else
      error = validate_phone(recipient)
      return error if error
      "SMS sent to #{recipient}: #{message}"
    end
  end
end

# Example 2: Using Config helper for environment variables
tool "check_smtp_config" do
  description "Check SMTP configuration with fallback keys"

  execute do |params|
    # Get config with multiple fallback keys
    host = Config.get('SMTP_HOST', 'MAIL_HOST', default: 'localhost')
    port = Config.get_int('SMTP_PORT', 'MAIL_PORT', default: 587)
    use_tls = Config.get_bool('SMTP_TLS', 'MAIL_TLS', default: true)

    # Check required configs
    missing = Config.check_required('SMTP_USER', 'SMTP_PASSWORD')
    return error("Missing configuration: #{missing.join(', ')}") unless missing.empty?

    success <<~CONFIG
      SMTP Configuration:
      Host: #{host}
      Port: #{port}
      TLS: #{use_tls}
      User: #{Config.get('SMTP_USER')}
    CONFIG
  end
end

# Example 3: Using HTTP helper
tool "fetch_json" do
  description "Fetch JSON data from a URL"

  parameter "url" do
    type :string
    required true
    description "URL to fetch"
    url_format  # Built-in validator
  end

  execute do |params|
    url = params["url"]

    # Use HTTP helper instead of curl
    response = HTTP.get(url, headers: { 'Accept' => 'application/json' })

    if response[:success]
      if response[:json]
        "Fetched JSON with #{response[:json].keys.length} keys"
      else
        "Response received but not JSON: #{truncate(response[:body])}"
      end
    else
      error("Failed to fetch URL: #{response[:error]}")
    end
  end
end

# Example 4: Using Shell helper for safe command execution
tool "safe_grep" do
  description "Safely search for a pattern in a file"

  parameter "pattern" do
    type :string
    required true
    description "Search pattern"
  end

  parameter "file" do
    type :string
    required true
    description "File to search in"
  end

  execute do |params|
    # This is safe from injection attacks!
    result = Shell.run('grep', params["pattern"], params["file"])

    if result[:success]
      "Found matches:\n#{result[:output]}"
    else
      "No matches found"
    end
  end
end

# Example 5: Using custom validation
tool "restricted_command" do
  description "Run a command from an allowed list"

  parameter "command" do
    type :string
    required true
    description "Command to run (must be in allowed list)"
    validate ->(cmd) {
      allowed = ['ls', 'pwd', 'whoami', 'date']
      allowed.include?(cmd) || "Command '#{cmd}' not allowed. Allowed: #{allowed.join(', ')}"
    }
  end

  execute do |params|
    result = Shell.run(params["command"])
    result[:output]
  end
end

# Example 6: Using multiple helpers together
tool "web_health_check" do
  description "Check if a web service is healthy"

  parameter "url" do
    type :string
    required true
    description "Service URL to check"
    url_format
  end

  parameter "expected_status" do
    type :number
    required false
    description "Expected HTTP status code"
    default 200
  end

  execute do |params|
    url = params["url"]
    expected = params["expected_status"] || 200

    # Use HTTP helper
    response = HTTP.head(url)

    if response[:error]
      error("Health check failed: #{response[:error]}")
    elsif response[:status] == expected
      success("✓ Service healthy (HTTP #{response[:status]})")
    else
      error("Service returned HTTP #{response[:status]}, expected #{expected}")
    end
  end
end

# Example 7: Using env_required helper
tool "database_info" do
  description "Show database connection info"

  execute do |params|
    # Check required env vars
    error = env_required('DATABASE_URL', 'DB_HOST')
    return error if error

    # Safe to use now
    db_url = env_get('DATABASE_URL', 'DB_URL')

    success("Database configured: #{db_url}")
  end
end
