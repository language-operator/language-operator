require 'spec_helper'
require 'net/smtp'

RSpec.describe 'test_smtp tool' do
  let(:registry) do
    load_email_tools
  end

  let(:tool) { registry.get('test_smtp') }

  let(:smtp_env) do
    {
      'SMTP_HOST' => 'smtp.example.com',
      'SMTP_PORT' => '587',
      'SMTP_USER' => 'test@example.com',
      'SMTP_PASSWORD' => 'password123',
      'SMTP_FROM' => 'sender@example.com',
      'SMTP_TLS' => 'true'
    }
  end

  before do
    smtp_env.each { |k, v| ENV[k] = v }
  end

  after do
    smtp_env.keys.each { |k| ENV.delete(k) }
  end

  it 'reports success when SMTP connection succeeds' do
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)

    result = tool.call({})

    expect(result).to include('SMTP Configuration Test: SUCCESS')
    expect(result).to include('Host: smtp.example.com')
    expect(result).to include('Port: 587')
    expect(result).to include('User: test@example.com')
    expect(result).to include('From: sender@example.com')
    expect(result).to include('TLS: true')
    expect(result).to include('Connection to SMTP server successful!')
  end

  it 'reports failure when SMTP connection fails' do
    allow(Net::SMTP).to receive(:start).and_raise(StandardError.new('Connection refused'))

    result = tool.call({})

    expect(result).to include('SMTP Configuration Test: FAILED')
    expect(result).to include('Host: smtp.example.com')
    expect(result).to include('Port: 587')
    expect(result).to include('User: test@example.com')
    expect(result).to include('Error: Connection refused')
  end

  it 'reports missing SMTP_HOST configuration' do
    ENV.delete('SMTP_HOST')

    result = tool.call({})

    expect(result).to include('Error: Missing SMTP configuration')
    expect(result).to include('SMTP_HOST')
  end

  it 'reports missing SMTP_USER configuration' do
    ENV.delete('SMTP_USER')

    result = tool.call({})

    expect(result).to include('Error: Missing SMTP configuration')
    expect(result).to include('SMTP_USER')
  end

  it 'reports missing SMTP_PASSWORD configuration' do
    ENV.delete('SMTP_PASSWORD')

    result = tool.call({})

    expect(result).to include('Error: Missing SMTP configuration')
    expect(result).to include('SMTP_PASSWORD')
  end

  it 'reports all missing configuration items' do
    ENV.delete('SMTP_HOST')
    ENV.delete('SMTP_USER')
    ENV.delete('SMTP_PASSWORD')

    result = tool.call({})

    expect(result).to include('Error: Missing SMTP configuration')
    expect(result).to include('SMTP_HOST')
    expect(result).to include('SMTP_USER')
    expect(result).to include('SMTP_PASSWORD')
  end

  it 'uses default port 587 when SMTP_PORT not set' do
    ENV.delete('SMTP_PORT')
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)

    result = tool.call({})

    expect(result).to include('Port: 587')
  end

  it 'uses SMTP_USER as from address when SMTP_FROM not set' do
    ENV.delete('SMTP_FROM')
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)

    result = tool.call({})

    expect(result).to include('From: test@example.com')
  end

  it 'handles authentication errors' do
    allow(Net::SMTP).to receive(:start).and_raise(Net::SMTPAuthenticationError.new('535 Authentication failed'))

    result = tool.call({})

    expect(result).to include('SMTP Configuration Test: FAILED')
    expect(result).to include('Error:')
  end

  it 'handles timeout errors' do
    allow(Net::SMTP).to receive(:start).and_raise(Timeout::Error.new('Connection timeout'))

    result = tool.call({})

    expect(result).to include('SMTP Configuration Test: FAILED')
    expect(result).to include('Connection timeout')
  end

  it 'shows TLS status from environment' do
    ENV['SMTP_TLS'] = 'false'
    smtp_mock = instance_double(Net::SMTP)
    allow(Net::SMTP).to receive(:start).and_yield(smtp_mock)

    result = tool.call({})

    expect(result).to include('TLS: false')
  end
end
