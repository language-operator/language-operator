require 'net/smtp'
require 'mail'

# Email tools for MCP

tool "send_email" do
  description "Send an email via SMTP"

  parameter "to" do
    type :string
    required true
    description "Recipient email address (comma-separated for multiple recipients)"
  end

  parameter "subject" do
    type :string
    required true
    description "Email subject line"
  end

  parameter "body" do
    type :string
    required true
    description "Email body content (plain text or HTML)"
  end

  parameter "from" do
    type :string
    required false
    description "Sender email address (defaults to SMTP_FROM env variable)"
  end

  parameter "cc" do
    type :string
    required false
    description "CC email addresses (comma-separated)"
  end

  parameter "bcc" do
    type :string
    required false
    description "BCC email addresses (comma-separated)"
  end

  parameter "html" do
    type :boolean
    description "Send as HTML email (default: false)"
    default false
  end

  execute do |params|
    # Get SMTP configuration from environment
    smtp_host = ENV['SMTP_HOST']
    smtp_port = ENV.fetch('SMTP_PORT', '587').to_i
    smtp_user = ENV['SMTP_USER']
    smtp_password = ENV['SMTP_PASSWORD']
    smtp_from = params['from'] || ENV['SMTP_FROM'] || smtp_user
    smtp_tls = ENV.fetch('SMTP_TLS', 'true') == 'true'

    # Validate from address first (more specific error)
    unless smtp_from
      next "Error: No sender address specified. Set SMTP_FROM or provide 'from' parameter."
    end

    # Validate general SMTP configuration
    unless smtp_host && smtp_user && smtp_password
      next "Error: SMTP configuration missing. Please set SMTP_HOST, SMTP_USER, and SMTP_PASSWORD environment variables."
    end

    # Parse recipients
    to_addresses = params['to'].split(',').map(&:strip)
    cc_addresses = params['cc'] ? params['cc'].split(',').map(&:strip) : []
    bcc_addresses = params['bcc'] ? params['bcc'].split(',').map(&:strip) : []

    # Build the email using the mail gem
    mail = Mail.new do
      from     smtp_from
      to       to_addresses
      cc       cc_addresses unless cc_addresses.empty?
      bcc      bcc_addresses unless bcc_addresses.empty?
      subject  params['subject']

      if params['html']
        content_type 'text/html; charset=UTF-8'
        body params['body']
      else
        text_part do
          body params['body']
        end
      end
    end

    begin
      # Send via SMTP
      smtp_params = [smtp_host, smtp_port]

      if smtp_tls
        # Use STARTTLS
        Net::SMTP.start(smtp_host, smtp_port, 'localhost', smtp_user, smtp_password, :login) do |smtp|
          smtp.send_message(mail.to_s, smtp_from, to_addresses + cc_addresses + bcc_addresses)
        end
      else
        # Plain SMTP
        Net::SMTP.start(smtp_host, smtp_port, 'localhost', smtp_user, smtp_password) do |smtp|
          smtp.send_message(mail.to_s, smtp_from, to_addresses + cc_addresses + bcc_addresses)
        end
      end

      "Email sent successfully to #{to_addresses.join(', ')}"
    rescue StandardError => e
      "Error sending email: #{e.message}"
    end
  end
end

tool "test_smtp" do
  description "Test SMTP connection and configuration"

  execute do |params|
    smtp_host = ENV['SMTP_HOST']
    smtp_port = ENV.fetch('SMTP_PORT', '587').to_i
    smtp_user = ENV['SMTP_USER']
    smtp_password = ENV['SMTP_PASSWORD']
    smtp_from = ENV['SMTP_FROM'] || smtp_user
    smtp_tls = ENV.fetch('SMTP_TLS', 'true') == 'true'

    # Check configuration
    missing = []
    missing << 'SMTP_HOST' unless smtp_host
    missing << 'SMTP_USER' unless smtp_user
    missing << 'SMTP_PASSWORD' unless smtp_password

    unless missing.empty?
      next "Error: Missing SMTP configuration: #{missing.join(', ')}"
    end

    # Test connection
    begin
      Net::SMTP.start(smtp_host, smtp_port, 'localhost', smtp_user, smtp_password, :login) do |smtp|
        # Connection successful
      end

      <<~RESULT
        SMTP Configuration Test: SUCCESS

        Host: #{smtp_host}
        Port: #{smtp_port}
        User: #{smtp_user}
        From: #{smtp_from}
        TLS: #{smtp_tls}

        Connection to SMTP server successful!
      RESULT
    rescue StandardError => e
      <<~RESULT
        SMTP Configuration Test: FAILED

        Host: #{smtp_host}
        Port: #{smtp_port}
        User: #{smtp_user}

        Error: #{e.message}
      RESULT
    end
  end
end

tool "email_config" do
  description "Display current email configuration (without sensitive data)"

  execute do |params|
    smtp_host = ENV['SMTP_HOST'] || '(not set)'
    smtp_port = ENV.fetch('SMTP_PORT', '587')
    smtp_user = ENV['SMTP_USER'] || '(not set)'
    smtp_from = ENV['SMTP_FROM'] || smtp_user
    smtp_tls = ENV.fetch('SMTP_TLS', 'true')

    password_set = ENV['SMTP_PASSWORD'] ? 'Yes (hidden)' : 'No'

    <<~CONFIG
      Email Configuration:

      SMTP_HOST: #{smtp_host}
      SMTP_PORT: #{smtp_port}
      SMTP_USER: #{smtp_user}
      SMTP_FROM: #{smtp_from}
      SMTP_TLS: #{smtp_tls}
      SMTP_PASSWORD: #{password_set}

      Note: Set these via environment variables when running the container.
    CONFIG
  end
end
