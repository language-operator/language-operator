# sms

An MCP server that provides SMS sending capabilities via Twilio or Vonage. Built on top of [based/svc/mcp](../mcp), this server allows AI assistants and other tools to send SMS messages programmatically.

## Quick Start

### Using Twilio

Run the server with Twilio credentials:

```bash
docker run -p 8080:80 \
  -e SMS_PROVIDER=twilio \
  -e TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxx \
  -e TWILIO_AUTH_TOKEN=your_auth_token \
  -e TWILIO_FROM=+1234567890 \
  based/svc/sms:latest
```

### Using Vonage/Nexmo

Run the server with Vonage credentials:

```bash
docker run -p 8080:80 \
  -e SMS_PROVIDER=vonage \
  -e VONAGE_API_KEY=your_api_key \
  -e VONAGE_API_SECRET=your_api_secret \
  -e VONAGE_FROM=YourBrand \
  based/svc/sms:latest
```

### Send an SMS

```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "send_sms",
    "arguments": {
      "to": "+1234567890",
      "message": "Hello from MCP SMS!"
    }
  }'
```

## Available Tools

### `send_sms`
Send an SMS message via your configured provider.

**Parameters:**
- `to` (string, required) - Recipient phone number in E.164 format (e.g., +1234567890)
- `message` (string, required) - SMS message text (max 160 characters for single SMS)
- `from` (string, optional) - Sender phone number or ID (defaults to SMS_FROM)

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "send_sms",
    "arguments": {
      "to": "+1234567890",
      "message": "Your verification code is: 123456"
    }
  }'
```

### `test_sms`
Test SMS configuration without sending a message.

**Parameters:** None

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"test_sms","arguments":{}}'
```

### `sms_config`
Display current SMS configuration (without showing sensitive data).

**Parameters:** None

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"sms_config","arguments":{}}'
```

## Configuration

### General Configuration

| Environment Variable | Required | Default | Description |
| -- | -- | -- | -- |
| SMS_PROVIDER | No | twilio | SMS provider to use (twilio or vonage) |

### Twilio Configuration

| Environment Variable | Required | Description |
| -- | -- | -- |
| TWILIO_ACCOUNT_SID | Yes | Your Twilio Account SID (starts with AC) |
| TWILIO_AUTH_TOKEN | Yes | Your Twilio Auth Token |
| TWILIO_FROM | Yes | Your Twilio phone number (E.164 format: +1234567890) |

Get credentials at: https://console.twilio.com/

### Vonage/Nexmo Configuration

| Environment Variable | Required | Description |
| -- | -- | -- |
| VONAGE_API_KEY | Yes | Your Vonage API key |
| VONAGE_API_SECRET | Yes | Your Vonage API secret |
| VONAGE_FROM | Yes | Your sender ID (phone number or alphanumeric) |

Get credentials at: https://dashboard.nexmo.com/

## Phone Number Format

All phone numbers must be in **E.164 format**:
- Start with `+`
- Country code (1-3 digits)
- National number (up to 12 digits)
- No spaces, dashes, or parentheses

**Examples:**
- ✅ `+14155551234` (US)
- ✅ `+442071234567` (UK)
- ✅ `+819012345678` (Japan)
- ❌ `(415) 555-1234` (invalid)
- ❌ `14155551234` (missing +)

## Provider Comparison

### Twilio
**Pros:**
- Most popular SMS API
- Excellent documentation
- Good free trial ($15 credit)
- Global coverage
- Reliable delivery

**Cons:**
- Slightly more expensive
- Requires phone number verification for trial

**Pricing:** ~$0.0075 per SMS (US)

### Vonage/Nexmo
**Pros:**
- Competitive pricing
- Good international coverage
- Alphanumeric sender IDs
- No phone verification needed for trial

**Cons:**
- Less documentation
- Smaller community

**Pricing:** ~$0.0057 per SMS (US)

## Development

Build the image:

```bash
make build
```

Run the server with test credentials:

```bash
SMS_PROVIDER=twilio \
TWILIO_ACCOUNT_SID=ACxxxx \
TWILIO_AUTH_TOKEN=token \
TWILIO_FROM=+1234567890 \
make run
```

Test the endpoints:

```bash
make test
```

Run linter:

```bash
make lint
```

Auto-fix linting issues:

```bash
make lint-fix
```

## Documentation

Generate API documentation with YARD:

```bash
make doc
```

Serve documentation locally on http://localhost:8808:

```bash
make doc-serve
```

Clean generated documentation:

```bash
make doc-clean
```

## Use Cases

- **Two-Factor Authentication (2FA)**: Send verification codes
- **Notifications**: Alert users about important events
- **Reminders**: Send appointment or deadline reminders
- **Alerts**: System monitoring and critical alerts
- **Marketing**: Send promotional messages (with consent)
- **Customer Service**: Automated customer updates
- **AI Assistants**: Allow AI to send SMS on behalf of users

## Security Considerations

- **Never commit credentials**: Use environment variables or secrets management
- **Rate limiting**: Both providers have rate limits
- **Phone number validation**: Always validate before sending
- **Consent**: Only send to numbers that have opted in
- **Message content**: Be aware of spam regulations
- **Cost**: Monitor usage to avoid unexpected charges

## Message Limits

### Single SMS (160 characters)
- GSM-7 encoding: 160 characters
- Unicode/Emoji: 70 characters

### Multi-part SMS
Messages longer than the limit are split and charged as multiple SMS:
- 153 characters per segment (GSM-7)
- 67 characters per segment (Unicode)

## Troubleshooting

### "Missing configuration" error
Make sure all required environment variables are set for your chosen provider.

### "Invalid phone number format" error
Ensure the phone number is in E.164 format (starts with +, country code, no spaces).

### Message not received
- Check the recipient's phone number is correct
- Verify the number can receive SMS
- Check spam/blocked messages on recipient device
- Review provider dashboard for delivery status

### Twilio trial limitations
- Can only send to verified phone numbers
- Messages include "Sent from a Twilio trial account"
- Upgrade to remove these restrictions

### High costs
- Monitor message length (stay under 160 chars)
- Batch notifications if possible
- Check provider's pricing for your destination country

## Examples

### Send verification code
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "send_sms",
    "arguments": {
      "to": "+1234567890",
      "message": "Your verification code is: 847293. Valid for 5 minutes."
    }
  }'
```

### Send appointment reminder
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "send_sms",
    "arguments": {
      "to": "+1234567890",
      "message": "Reminder: Your appointment is tomorrow at 2:00 PM. Reply CONFIRM to confirm."
    }
  }'
```

## Architecture

This image extends `based/svc/mcp:latest` and uses the MCP DSL to define SMS tools. The tools are defined in [tools/sms.rb](tools/sms.rb) and use REST APIs to communicate with Twilio and Vonage services.

## Links

- [Twilio SMS Documentation](https://www.twilio.com/docs/sms)
- [Vonage SMS API](https://developer.vonage.com/messaging/sms/overview)
- [E.164 Phone Number Format](https://en.wikipedia.org/wiki/E.164)
