# frozen_string_literal: true

require 'spec_helper'
require 'langop/dsl/execution_context'

RSpec.describe Langop::Dsl::ExecutionContext do
  let(:tool_name) { 'test_tool' }
  let(:params) { { 'input' => 'test' } }
  let(:context) { described_class.new(tool_name, params) }

  describe '#initialize' do
    it 'stores tool name and parameters' do
      expect(context.tool_name).to eq(tool_name)
      expect(context.params).to eq(params)
    end

    it 'initializes metadata as empty hash' do
      expect(context.metadata).to eq({})
    end

    it 'accepts optional metadata' do
      ctx = described_class.new(tool_name, params, request_id: '123')
      expect(ctx.metadata[:request_id]).to eq('123')
    end
  end

  describe '#shell' do
    it 'executes shell commands' do
      result = context.shell('echo "hello"')
      expect(result[:stdout]).to include('hello')
      expect(result[:exit_code]).to eq(0)
    end

    it 'captures stderr' do
      result = context.shell('echo "error" >&2')
      expect(result[:stderr]).to include('error')
    end

    it 'returns non-zero exit codes' do
      result = context.shell('exit 1')
      expect(result[:exit_code]).to eq(1)
    end

    it 'handles command timeout' do
      result = context.shell('sleep 10', timeout: 1)
      expect(result[:timeout]).to be true
    end
  end

  describe '#http_get' do
    it 'makes HTTP GET requests' do
      # Mock HTTP response
      stub_request(:get, 'http://example.com/api')
        .to_return(status: 200, body: '{"status":"ok"}', headers: { 'Content-Type' => 'application/json' })

      result = context.http_get('http://example.com/api')
      expect(result[:status]).to eq(200)
      expect(result[:body]).to include('ok')
    end

    it 'handles HTTP errors' do
      stub_request(:get, 'http://example.com/not-found')
        .to_return(status: 404, body: 'Not Found')

      result = context.http_get('http://example.com/not-found')
      expect(result[:status]).to eq(404)
    end

    it 'includes custom headers' do
      stub_request(:get, 'http://example.com/auth')
        .with(headers: { 'Authorization' => 'Bearer token123' })
        .to_return(status: 200, body: 'authenticated')

      result = context.http_get('http://example.com/auth',
                                headers: { 'Authorization' => 'Bearer token123' })
      expect(result[:status]).to eq(200)
    end
  end

  describe '#http_post' do
    it 'makes HTTP POST requests with body' do
      stub_request(:post, 'http://example.com/api')
        .with(body: '{"data":"value"}')
        .to_return(status: 201, body: '{"created":true}')

      result = context.http_post('http://example.com/api', body: '{"data":"value"}')
      expect(result[:status]).to eq(201)
      expect(result[:body]).to include('created')
    end
  end

  describe '#file_read' do
    let(:temp_file) { Tempfile.new('test') }

    before do
      temp_file.write('test content')
      temp_file.close
    end

    after do
      temp_file.unlink
    end

    it 'reads file contents' do
      result = context.file_read(temp_file.path)
      expect(result[:content]).to include('test content')
      expect(result[:success]).to be true
    end

    it 'handles missing files' do
      result = context.file_read('/nonexistent/file.txt')
      expect(result[:success]).to be false
      expect(result[:error]).to match(/not found|no such file/i)
    end
  end

  describe '#file_write' do
    let(:temp_dir) { Dir.mktmpdir }

    after do
      FileUtils.rm_rf(temp_dir)
    end

    it 'writes content to file' do
      file_path = File.join(temp_dir, 'output.txt')
      result = context.file_write(file_path, 'new content')

      expect(result[:success]).to be true
      expect(File.read(file_path)).to eq('new content')
    end

    it 'creates parent directories if needed' do
      file_path = File.join(temp_dir, 'nested', 'deep', 'file.txt')
      result = context.file_write(file_path, 'content')

      expect(result[:success]).to be true
      expect(File.exist?(file_path)).to be true
    end
  end

  describe '#log' do
    it 'logs messages' do
      expect { context.log('Test message') }.not_to raise_error
    end

    it 'logs with different levels' do
      expect { context.log('Info', level: :info) }.not_to raise_error
      expect { context.log('Warning', level: :warn) }.not_to raise_error
      expect { context.log('Error', level: :error) }.not_to raise_error
    end
  end

  describe '#env' do
    it 'accesses environment variables' do
      ENV['TEST_VAR'] = 'test_value'
      expect(context.env('TEST_VAR')).to eq('test_value')
      ENV.delete('TEST_VAR')
    end

    it 'returns nil for missing variables' do
      expect(context.env('NONEXISTENT_VAR')).to be_nil
    end

    it 'returns default value when variable missing' do
      expect(context.env('NONEXISTENT_VAR', 'default')).to eq('default')
    end
  end

  describe '#metadata' do
    it 'allows storing custom metadata' do
      context.metadata[:custom_key] = 'custom_value'
      expect(context.metadata[:custom_key]).to eq('custom_value')
    end

    it 'persists across method calls' do
      context.metadata[:counter] = 0
      context.metadata[:counter] += 1
      expect(context.metadata[:counter]).to eq(1)
    end
  end
end
