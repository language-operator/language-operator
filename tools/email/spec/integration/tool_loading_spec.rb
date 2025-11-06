require 'spec_helper'

RSpec.describe 'Email tool loading' do
  let(:registry) { LanguageOperator::Dsl::Registry.new }
  let(:tool_path) { File.expand_path('../../tools/email.rb', __dir__) }

  before do
    context = LanguageOperator::Dsl::Context.new(registry)
    code = File.read(tool_path)
    context.instance_eval(code, tool_path)
  end

  it 'loads all email tools' do
    expect(registry.get('send_email')).not_to be_nil
    expect(registry.get('test_smtp')).not_to be_nil
    expect(registry.get('email_config')).not_to be_nil
  end

  it 'loads send_email tool with correct parameters' do
    tool = registry.get('send_email')

    expect(tool).not_to be_nil
    expect(tool.name).to eq('send_email')
    expect(tool.description).to include('Send an email')

    schema = tool.to_schema
    expect(schema['inputSchema']['properties']).to have_key('to')
    expect(schema['inputSchema']['properties']).to have_key('subject')
    expect(schema['inputSchema']['properties']).to have_key('body')
    expect(schema['inputSchema']['properties']).to have_key('from')
    expect(schema['inputSchema']['properties']).to have_key('cc')
    expect(schema['inputSchema']['properties']).to have_key('bcc')
    expect(schema['inputSchema']['properties']).to have_key('html')

    expect(schema['inputSchema']['required']).to include('to', 'subject', 'body')
  end

  it 'loads test_smtp tool with correct definition' do
    tool = registry.get('test_smtp')

    expect(tool).not_to be_nil
    expect(tool.name).to eq('test_smtp')
    expect(tool.description).to include('Test SMTP connection')

    schema = tool.to_schema
    expect(schema['inputSchema']['properties']).to be_empty
  end

  it 'loads email_config tool with correct definition' do
    tool = registry.get('email_config')

    expect(tool).not_to be_nil
    expect(tool.name).to eq('email_config')
    expect(tool.description).to include('Display current email configuration')

    schema = tool.to_schema
    expect(schema['inputSchema']['properties']).to be_empty
  end

  it 'provides correct parameter types for send_email' do
    tool = registry.get('send_email')
    schema = tool.to_schema

    expect(schema['inputSchema']['properties']['to']['type']).to eq('string')
    expect(schema['inputSchema']['properties']['subject']['type']).to eq('string')
    expect(schema['inputSchema']['properties']['body']['type']).to eq('string')
    expect(schema['inputSchema']['properties']['from']['type']).to eq('string')
    expect(schema['inputSchema']['properties']['cc']['type']).to eq('string')
    expect(schema['inputSchema']['properties']['bcc']['type']).to eq('string')
    expect(schema['inputSchema']['properties']['html']['type']).to eq('boolean')
  end

  it 'provides parameter descriptions for send_email' do
    tool = registry.get('send_email')
    schema = tool.to_schema

    expect(schema['inputSchema']['properties']['to']['description']).to include('Recipient email address')
    expect(schema['inputSchema']['properties']['subject']['description']).to include('Email subject')
    expect(schema['inputSchema']['properties']['body']['description']).to include('Email body')
    expect(schema['inputSchema']['properties']['from']['description']).to include('Sender email')
    expect(schema['inputSchema']['properties']['cc']['description']).to include('CC email')
    expect(schema['inputSchema']['properties']['bcc']['description']).to include('BCC email')
    expect(schema['inputSchema']['properties']['html']['description']).to include('HTML email')
  end
end
