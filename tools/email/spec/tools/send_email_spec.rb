require 'spec_helper'
require 'net/smtp'

RSpec.describe 'send_email tool' do
  let(:registry) do
    load_email_tools
  end

  let(:tool) { registry.get('send_email') }

  let(:smtp_env) do
    {
      'SMTP_HOST' => 'smtp.example.com',
      'SMTP_PORT' => '587',
      'SMTP_USER' => 'test@example.com',
      'SMTP_PASSWORD' => 'password123',
      'SMTP_FROM' => 'test@example.com',
      'SMTP_TLS' => 'true'
    }
  end

  before do
    smtp_env.each { |k, v| ENV[k] = v }
  end

  after do
    smtp_env.keys.each { |k| ENV.delete(k) }
  end

  it 'sends a simple email successfully' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => 'recipient@example.com',
      'subject' => 'Test Subject',
      'body' => 'Test body content'
    })

    expect(result).to include('Email sent successfully')
    expect(result).to include('recipient@example.com')
  end

  it 'sends email to multiple recipients' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => 'user1@example.com, user2@example.com, user3@example.com',
      'subject' => 'Test',
      'body' => 'Hello'
    })

    expect(result).to include('Email sent successfully')
    expect(result).to include('user1@example.com')
    expect(result).to include('user2@example.com')
    expect(result).to include('user3@example.com')
  end

  it 'sends email with CC recipients' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => 'primary@example.com',
      'cc' => 'cc1@example.com, cc2@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Email sent successfully')
  end

  it 'sends email with BCC recipients' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => 'primary@example.com',
      'bcc' => 'bcc1@example.com, bcc2@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Email sent successfully')
  end

  it 'sends HTML email when html parameter is true' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'HTML Test',
      'body' => '<h1>Hello</h1><p>World</p>',
      'html' => true
    })

    expect(result).to include('Email sent successfully')
  end

  it 'sends plain text email by default' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Plain Text',
      'body' => 'Simple text content'
    })

    expect(result).to include('Email sent successfully')
  end

  it 'uses custom from address when provided' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => 'user@example.com',
      'from' => 'custom@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Email sent successfully')
  end

  it 'returns error when SMTP_HOST is missing' do
    ENV.delete('SMTP_HOST')

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Error: SMTP configuration missing')
  end

  it 'returns error when SMTP_USER is missing' do
    ENV.delete('SMTP_USER')

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Error: SMTP configuration missing')
  end

  it 'returns error when SMTP_PASSWORD is missing' do
    ENV.delete('SMTP_PASSWORD')

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Error: SMTP configuration missing')
  end

  it 'returns error when from address cannot be determined' do
    ENV.delete('SMTP_FROM')
    ENV.delete('SMTP_USER')

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Error: No sender address specified')
  end

  it 'handles SMTP connection errors gracefully' do
    allow(Net::SMTP).to receive(:start).and_raise(StandardError.new('Connection refused'))

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Error sending email')
    expect(result).to include('Connection refused')
  end

  it 'handles authentication errors gracefully' do
    allow(Net::SMTP).to receive(:start).and_raise(Net::SMTPAuthenticationError.new('Invalid credentials'))

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Error sending email')
  end

  it 'handles network timeout errors' do
    allow(Net::SMTP).to receive(:start).and_raise(Timeout::Error.new('Connection timeout'))

    result = tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Error sending email')
    expect(result).to include('Connection timeout')
  end

  it 'trims whitespace from email addresses' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)
    allow(smtp_mock).to receive(:send_message)

    result = tool.call({
      'to' => '  user1@example.com  ,  user2@example.com  ',
      'subject' => 'Test',
      'body' => 'Content'
    })

    expect(result).to include('Email sent successfully')
  end

  it 'uses TLS by default' do
    expect(Net::SMTP).to receive(:start).with(
      'smtp.example.com',
      587,
      'localhost',
      'test@example.com',
      'password123',
      :login
    ).and_yield(instance_double(Net::SMTP, send_message: nil))

    tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })
  end

  it 'disables TLS when SMTP_TLS is false' do
    ENV['SMTP_TLS'] = 'false'

    expect(Net::SMTP).to receive(:start).with(
      'smtp.example.com',
      587,
      'localhost',
      'test@example.com',
      'password123'
    ).and_yield(instance_double(Net::SMTP, send_message: nil))

    tool.call({
      'to' => 'user@example.com',
      'subject' => 'Test',
      'body' => 'Content'
    })
  end
end
