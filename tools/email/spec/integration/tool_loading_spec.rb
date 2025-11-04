require 'spec_helper'

RSpec.describe 'Email tool loading' do
  let(:registry) do
    Langop::ToolLoader.load_from_directory('/mcp')
  end

  it 'loads all email tools' do
    tools = registry.list

    expect(tools.map(&:name)).to include('send_email', 'test_smtp', 'email_config')
  end

  it 'loads send_email tool with correct parameters' do
    tool = registry.get('send_email')

    expect(tool).not_to be_nil
    expect(tool.name).to eq('send_email')
    expect(tool.description).to include('Send an email')

    schema = tool.input_schema
    expect(schema['properties']).to have_key('to')
    expect(schema['properties']).to have_key('subject')
    expect(schema['properties']).to have_key('body')
    expect(schema['properties']).to have_key('from')
    expect(schema['properties']).to have_key('cc')
    expect(schema['properties']).to have_key('bcc')
    expect(schema['properties']).to have_key('html')

    expect(schema['required']).to include('to', 'subject', 'body')
  end

  it 'loads test_smtp tool with correct definition' do
    tool = registry.get('test_smtp')

    expect(tool).not_to be_nil
    expect(tool.name).to eq('test_smtp')
    expect(tool.description).to include('Test SMTP connection')

    schema = tool.input_schema
    expect(schema['properties']).to be_empty
  end

  it 'loads email_config tool with correct definition' do
    tool = registry.get('email_config')

    expect(tool).not_to be_nil
    expect(tool.name).to eq('email_config')
    expect(tool.description).to include('Display current email configuration')

    schema = tool.input_schema
    expect(schema['properties']).to be_empty
  end

  it 'provides correct parameter types for send_email' do
    tool = registry.get('send_email')
    schema = tool.input_schema

    expect(schema['properties']['to']['type']).to eq('string')
    expect(schema['properties']['subject']['type']).to eq('string')
    expect(schema['properties']['body']['type']).to eq('string')
    expect(schema['properties']['from']['type']).to eq('string')
    expect(schema['properties']['cc']['type']).to eq('string')
    expect(schema['properties']['bcc']['type']).to eq('string')
    expect(schema['properties']['html']['type']).to eq('boolean')
  end

  it 'provides parameter descriptions for send_email' do
    tool = registry.get('send_email')
    schema = tool.input_schema

    expect(schema['properties']['to']['description']).to include('Recipient email address')
    expect(schema['properties']['subject']['description']).to include('Email subject')
    expect(schema['properties']['body']['description']).to include('Email body')
    expect(schema['properties']['from']['description']).to include('Sender email')
    expect(schema['properties']['cc']['description']).to include('CC email')
    expect(schema['properties']['bcc']['description']).to include('BCC email')
    expect(schema['properties']['html']['description']).to include('HTML email')
  end
end
