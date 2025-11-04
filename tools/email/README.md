# email

An MCP server that provides email sending capabilities via SMTP. Built on top of [based/svc/mcp](../mcp), this server allows AI assistants and other tools to send emails programmatically.

## Quick Start

Run the server with SMTP credentials:

```bash
docker run -p 8080:80 \
  -e SMTP_HOST=smtp.gmail.com \
  -e SMTP_PORT=587 \
  -e SMTP_USER=your-email@gmail.com \
  -e SMTP_PASSWORD=your-app-password \
  -e SMTP_FROM=your-email@gmail.com \
  based/svc/email:latest
```

Send an email:

```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "send_email",
    "arguments": {
      "to": "recipient@example.com",
      "subject": "Hello from MCP",
      "body": "This is a test email sent via MCP!"
    }
  }'
```

## Available Tools

### `send_email`
Send an email via SMTP.

**Parameters:**
- `to` (string, required) - Recipient email address (comma-separated for multiple)
- `subject` (string, required) - Email subject line
- `body` (string, required) - Email body content
- `from` (string, optional) - Sender email address (defaults to SMTP_FROM)
- `cc` (string, optional) - CC email addresses (comma-separated)
- `bcc` (string, optional) - BCC email addresses (comma-separated)
- `html` (boolean, optional) - Send as HTML email (default: false)

**Example:**
```bash
# Simple email
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "send_email",
    "arguments": {
      "to": "user@example.com",
      "subject": "Meeting Reminder",
      "body": "Don'\''t forget our meeting at 3pm!"
    }
  }'

# Email with CC and HTML
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "send_email",
    "arguments": {
      "to": "user@example.com",
      "cc": "manager@example.com",
      "subject": "Project Update",
      "body": "<h1>Project Status</h1><p>All tasks completed!</p>",
      "html": true
    }
  }'
```

### `test_smtp`
Test SMTP connection and configuration.

**Parameters:** None

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"test_smtp","arguments":{}}'
```

### `email_config`
Display current email configuration (without showing sensitive data).

**Parameters:** None

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"email_config","arguments":{}}'
```

## Configuration

Email configuration is provided via environment variables:

| Environment Variable | Required | Default | Description |
| -- | -- | -- | -- |
| SMTP_HOST | Yes | - | SMTP server hostname (e.g., smtp.gmail.com) |
| SMTP_PORT | No | 587 | SMTP port (usually 587 for TLS, 465 for SSL) |
| SMTP_USER | Yes | - | SMTP username (often your email address) |
| SMTP_PASSWORD | Yes | - | SMTP password or app-specific password |
| SMTP_FROM | No | SMTP_USER | Default sender email address |
| SMTP_TLS | No | true | Use TLS encryption |

## Common SMTP Providers

### Gmail
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password  # Generate at https://myaccount.google.com/apppasswords
```

### Outlook/Office 365
```bash
SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USER=your-email@outlook.com
SMTP_PASSWORD=your-password
```

### SendGrid
```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASSWORD=your-api-key
```

### Mailgun
```bash
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@your-domain.mailgun.org
SMTP_PASSWORD=your-smtp-password
```

## Development

Build the image:

```bash
make build
```

Run the server with test credentials:

```bash
SMTP_HOST=smtp.gmail.com \
SMTP_USER=test@gmail.com \
SMTP_PASSWORD=app-password \
make run
```

Run unit tests:

```bash
make spec
```

Test all endpoints (integration):

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

### Testing

The project includes comprehensive test coverage using RSpec:

- **Unit tests** for all 3 email tools (`send_email`, `test_smtp`, `email_config`)
- **Integration tests** for tool loading and registry
- **Mocked SMTP connections** using RSpec doubles to avoid external dependencies
- **Environment variable testing** for configuration validation

Test coverage includes:
- Parameter validation (to, subject, body, from, cc, bcc, html)
- SMTP configuration validation
- Email address parsing and formatting
- Error handling (missing config, connection errors, authentication failures)
- TLS/non-TLS connection modes
- Multiple recipient support (to, cc, bcc)
- HTML vs plain text email modes
- Configuration display (without exposing secrets)

Run tests with `make spec` to execute the full test suite in Docker.

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

## Security Considerations

- **Never commit credentials**: Use environment variables or secrets management
- **App-specific passwords**: For Gmail, use app-specific passwords, not your main password
- **TLS encryption**: Always use TLS (enabled by default)
- **Rate limiting**: Be aware of your SMTP provider's rate limits
- **Validation**: The server does basic email validation but doesn't prevent spam

## Use Cases

- **Notifications**: Send alerts and notifications from automated systems
- **AI Assistants**: Allow AI to send emails on behalf of users
- **Workflows**: Integrate email into automated workflows
- **Reports**: Send automated reports and summaries
- **Alerts**: System monitoring and alerting

## Architecture

This image extends `based/svc/mcp:latest` and uses the MCP DSL to define email tools. The tools are defined in [tools/email.rb](tools/email.rb) and use Ruby's built-in `net/smtp` library along with the `mail` gem for email formatting.

## Troubleshooting

### "SMTP configuration missing" error
Make sure all required environment variables are set: `SMTP_HOST`, `SMTP_USER`, `SMTP_PASSWORD`

### Connection timeout
- Check firewall settings
- Verify SMTP_HOST and SMTP_PORT are correct
- Some networks block outbound SMTP connections

### Authentication failed
- Verify credentials are correct
- For Gmail, use an app-specific password
- Check if 2FA is enabled on your account

### Emails not received
- Check spam folder
- Verify recipient address is correct
- Check SMTP provider logs for delivery status
