require 'spec_helper'

RSpec.describe 'email_config tool' do
  let(:registry) do
    load_email_tools
  end

  let(:tool) { registry.get('email_config') }

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

  it 'displays all configuration values' do
    result = tool.call({})

    expect(result).to include('Email Configuration:')
    expect(result).to include('SMTP_HOST: smtp.example.com')
    expect(result).to include('SMTP_PORT: 587')
    expect(result).to include('SMTP_USER: test@example.com')
    expect(result).to include('SMTP_FROM: sender@example.com')
    expect(result).to include('SMTP_TLS: true')
  end

  it 'shows password as hidden when set' do
    result = tool.call({})

    expect(result).to include('SMTP_PASSWORD: Yes (hidden)')
    expect(result).not_to include('password123')
  end

  it 'shows password as not set when missing' do
    ENV.delete('SMTP_PASSWORD')

    result = tool.call({})

    expect(result).to include('SMTP_PASSWORD: No')
  end

  it 'shows (not set) for missing SMTP_HOST' do
    ENV.delete('SMTP_HOST')

    result = tool.call({})

    expect(result).to include('SMTP_HOST: (not set)')
  end

  it 'shows (not set) for missing SMTP_USER' do
    ENV.delete('SMTP_USER')

    result = tool.call({})

    expect(result).to include('SMTP_USER: (not set)')
  end

  it 'uses default port 587 when SMTP_PORT not set' do
    ENV.delete('SMTP_PORT')

    result = tool.call({})

    expect(result).to include('SMTP_PORT: 587')
  end

  it 'uses SMTP_USER as SMTP_FROM when SMTP_FROM not set' do
    ENV.delete('SMTP_FROM')

    result = tool.call({})

    expect(result).to include('SMTP_FROM: test@example.com')
  end

  it 'shows (not set) for SMTP_FROM when both SMTP_FROM and SMTP_USER missing' do
    ENV.delete('SMTP_FROM')
    ENV.delete('SMTP_USER')

    result = tool.call({})

    expect(result).to include('SMTP_FROM: (not set)')
  end

  it 'uses default true for SMTP_TLS when not set' do
    ENV.delete('SMTP_TLS')

    result = tool.call({})

    expect(result).to include('SMTP_TLS: true')
  end

  it 'includes usage note' do
    result = tool.call({})

    expect(result).to include('Note: Set these via environment variables when running the container.')
  end

  it 'displays custom port value' do
    ENV['SMTP_PORT'] = '465'

    result = tool.call({})

    expect(result).to include('SMTP_PORT: 465')
  end

  it 'displays TLS disabled status' do
    ENV['SMTP_TLS'] = 'false'

    result = tool.call({})

    expect(result).to include('SMTP_TLS: false')
  end
end
