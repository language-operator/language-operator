# Ruby SDK for Language Operator

This gem translates a high-level DSL into language operator components.

## Tools

```ruby
require 'langop'

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
    required false
    description "Send as HTML email (default: false)"
    default false
  end

  execute do |params|
    # ...implementation of tool
  end
end


## Agents

```ruby
require 'langop'

agent "demo" do
end
```