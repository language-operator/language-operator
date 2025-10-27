require 'net/http'
require 'uri'
require 'json'
require 'base64'

# SMS tools for MCP

tool "send_sms" do
  description "Send an SMS message via Twilio or Vonage"

  parameter "to" do
    type :string
    required true
    description "Recipient phone number (E.164 format: +1234567890)"
  end

  parameter "message" do
    type :string
    required true
    description "SMS message text (max 160 characters for single SMS)"
  end

  parameter "from" do
    type :string
    required false
    description "Sender phone number (defaults to SMS_FROM env variable)"
  end

  execute do |params|
    provider = ENV.fetch('SMS_PROVIDER', 'twilio').downcase
    to = params['to']
    message = params['message']
    from = params['from'] || ENV['SMS_FROM']

    # Validate phone number format
    unless to =~ /^\+\d{10,15}$/
      return "Error: Invalid phone number format. Use E.164 format (e.g., +1234567890)"
    end

    # Send via appropriate provider
    case provider
    when 'twilio'
      send_twilio_sms(to, message, from)
    when 'vonage', 'nexmo'
      send_vonage_sms(to, message, from)
    else
      "Error: Unsupported SMS provider '#{provider}'. Supported: twilio, vonage"
    end
  end
end

tool "sms_config" do
  description "Display current SMS configuration (without sensitive data)"

  execute do |params|
    provider = ENV.fetch('SMS_PROVIDER', 'twilio').downcase

    config = case provider
    when 'twilio'
      account_sid = ENV['TWILIO_ACCOUNT_SID'] || ENV['SMS_ACCOUNT_SID'] || '(not set)'
      auth_token_set = (ENV['TWILIO_AUTH_TOKEN'] || ENV['SMS_AUTH_TOKEN']) ? 'Yes (hidden)' : 'No'
      from = ENV['TWILIO_FROM'] || ENV['SMS_FROM'] || '(not set)'

      <<~CONFIG
        SMS Configuration (Twilio):

        Provider: Twilio
        Account SID: #{account_sid}
        Auth Token: #{auth_token_set}
        From Number: #{from}

        Required environment variables:
        - TWILIO_ACCOUNT_SID (or SMS_ACCOUNT_SID)
        - TWILIO_AUTH_TOKEN (or SMS_AUTH_TOKEN)
        - TWILIO_FROM (or SMS_FROM)
      CONFIG

    when 'vonage', 'nexmo'
      api_key = ENV['VONAGE_API_KEY'] || '(not set)'
      api_secret_set = ENV['VONAGE_API_SECRET'] ? 'Yes (hidden)' : 'No'
      from = ENV['VONAGE_FROM'] || ENV['SMS_FROM'] || '(not set)'

      <<~CONFIG
        SMS Configuration (Vonage):

        Provider: Vonage/Nexmo
        API Key: #{api_key}
        API Secret: #{api_secret_set}
        From Number: #{from}

        Required environment variables:
        - VONAGE_API_KEY
        - VONAGE_API_SECRET
        - VONAGE_FROM (or SMS_FROM)
      CONFIG

    else
      "Error: Unknown provider '#{provider}'"
    end

    config
  end
end

tool "test_sms" do
  description "Test SMS configuration without sending a message"

  execute do |params|
    provider = ENV.fetch('SMS_PROVIDER', 'twilio').downcase

    case provider
    when 'twilio'
      account_sid = ENV['TWILIO_ACCOUNT_SID'] || ENV['SMS_ACCOUNT_SID']
      auth_token = ENV['TWILIO_AUTH_TOKEN'] || ENV['SMS_AUTH_TOKEN']
      from = ENV['TWILIO_FROM'] || ENV['SMS_FROM']

      missing = []
      missing << 'TWILIO_ACCOUNT_SID' unless account_sid
      missing << 'TWILIO_AUTH_TOKEN' unless auth_token
      missing << 'TWILIO_FROM' unless from

      if missing.empty?
        "SMS Configuration Test: OK\n\nProvider: Twilio\nAccount SID: #{account_sid}\nFrom: #{from}\n\nConfiguration appears valid. Use 'send_sms' to send a test message."
      else
        "SMS Configuration Test: FAILED\n\nMissing: #{missing.join(', ')}"
      end

    when 'vonage', 'nexmo'
      api_key = ENV['VONAGE_API_KEY']
      api_secret = ENV['VONAGE_API_SECRET']
      from = ENV['VONAGE_FROM'] || ENV['SMS_FROM']

      missing = []
      missing << 'VONAGE_API_KEY' unless api_key
      missing << 'VONAGE_API_SECRET' unless api_secret
      missing << 'VONAGE_FROM' unless from

      if missing.empty?
        "SMS Configuration Test: OK\n\nProvider: Vonage\nAPI Key: #{api_key}\nFrom: #{from}\n\nConfiguration appears valid. Use 'send_sms' to send a test message."
      else
        "SMS Configuration Test: FAILED\n\nMissing: #{missing.join(', ')}"
      end

    else
      "Error: Unknown provider '#{provider}'"
    end
  end
end

# Helper method for Twilio
def send_twilio_sms(to, message, from)
  account_sid = ENV['TWILIO_ACCOUNT_SID'] || ENV['SMS_ACCOUNT_SID']
  auth_token = ENV['TWILIO_AUTH_TOKEN'] || ENV['SMS_AUTH_TOKEN']
  from ||= ENV['TWILIO_FROM']

  unless account_sid && auth_token && from
    return "Error: Missing Twilio configuration. Set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, and TWILIO_FROM"
  end

  # Build the request
  uri = URI("https://api.twilio.com/2010-04-01/Accounts/#{account_sid}/Messages.json")

  http = Net::HTTP.new(uri.host, uri.port)
  http.use_ssl = true

  request = Net::HTTP::Post.new(uri)
  request.basic_auth(account_sid, auth_token)
  request.set_form_data({
    'To' => to,
    'From' => from,
    'Body' => message
  })

  begin
    response = http.request(request)

    if response.code == '201'
      data = JSON.parse(response.body)
      "SMS sent successfully via Twilio!\n\nTo: #{to}\nFrom: #{from}\nMessage SID: #{data['sid']}\nStatus: #{data['status']}"
    else
      error = JSON.parse(response.body)
      "Error sending SMS: #{error['message']} (Code: #{error['code']})"
    end
  rescue => e
    "Error sending SMS: #{e.message}"
  end
end

# Helper method for Vonage
def send_vonage_sms(to, message, from)
  api_key = ENV['VONAGE_API_KEY']
  api_secret = ENV['VONAGE_API_SECRET']
  from ||= ENV['VONAGE_FROM']

  unless api_key && api_secret && from
    return "Error: Missing Vonage configuration. Set VONAGE_API_KEY, VONAGE_API_SECRET, and VONAGE_FROM"
  end

  # Build the request
  uri = URI('https://rest.nexmo.com/sms/json')

  http = Net::HTTP.new(uri.host, uri.port)
  http.use_ssl = true

  request = Net::HTTP::Post.new(uri)
  request['Content-Type'] = 'application/x-www-form-urlencoded'
  request.set_form_data({
    'api_key' => api_key,
    'api_secret' => api_secret,
    'to' => to.gsub(/^\+/, ''),  # Vonage doesn't want the + prefix
    'from' => from,
    'text' => message
  })

  begin
    response = http.request(request)

    if response.code == '200'
      data = JSON.parse(response.body)
      msg = data['messages'][0]

      if msg['status'] == '0'
        "SMS sent successfully via Vonage!\n\nTo: #{to}\nFrom: #{from}\nMessage ID: #{msg['message-id']}\nPrice: #{msg['message-price']} #{msg['network']}"
      else
        "Error sending SMS: #{msg['error-text']} (Status: #{msg['status']})"
      end
    else
      "Error sending SMS: HTTP #{response.code}"
    end
  rescue => e
    "Error sending SMS: #{e.message}"
  end
end
