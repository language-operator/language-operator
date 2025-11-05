require_relative '../spec_helper'

RSpec.describe 'web_headers tool' do
  let(:registry) { LanguageOperator::Dsl::Registry.new }
  let(:tool_path) { File.expand_path('../../tools/web.rb', __dir__) }
  let(:example_headers) { File.read(File.expand_path('../fixtures/example_headers.txt', __dir__)) }

  before do
    context = LanguageOperator::Dsl::Context.new(registry)
    code = File.read(tool_path)
    context.instance_eval(code, tool_path)
  end

  describe 'successful header fetch' do
    it 'returns HTTP headers' do
      stub_request(:head, 'https://example.com/')
        .to_return(status: 200, headers: {
          'Content-Type' => 'text/html',
          'Server' => 'nginx/1.18.0',
          'Cache-Control' => 'max-age=3600'
        })

      tool = registry.get('web_headers')
      result = tool.call('url' => 'https://example.com/')

      expect(result).to include('Headers for https://example.com/')
      expect(result).to include('content-type')
    end

    it 'includes various header fields' do
      stub_request(:head, 'https://example.com/')
        .to_return(status: 200, body: example_headers)

      tool = registry.get('web_headers')
      result = tool.call('url' => 'https://example.com/')

      expect(result).to include('Headers for https://example.com/')
    end
  end

  describe 'URL validation' do
    it 'rejects URLs without http:// or https://' do
      tool = registry.get('web_headers')
      result = tool.call('url' => 'example.com')

      expect(result).to include('Error: Invalid URL')
      expect(result).to include('Must start with http:// or https://')
    end

    it 'accepts http:// URLs' do
      stub_request(:head, 'http://example.com/')
        .to_return(status: 200, body: 'HTTP/1.1 200 OK')

      tool = registry.get('web_headers')
      result = tool.call('url' => 'http://example.com/')

      expect(result).to include('Headers for http://example.com/')
    end

    it 'accepts https:// URLs' do
      stub_request(:head, 'https://example.com/')
        .to_return(status: 200, body: 'HTTP/1.1 200 OK')

      tool = registry.get('web_headers')
      result = tool.call('url' => 'https://example.com/')

      expect(result).to include('Headers for https://example.com/')
    end
  end

  describe 'error handling' do
    it 'returns error message when fetch fails with 500' do
      stub_request(:head, 'https://example.com/error')
        .to_return(status: 500, body: '')

      tool = registry.get('web_headers')
      result = tool.call('url' => 'https://example.com/error')

      expect(result).to include('Error: Failed to fetch headers')
      expect(result).to include('https://example.com/error')
    end

    it 'returns error message on network timeout' do
      stub_request(:head, 'https://example.com/timeout')
        .to_timeout

      tool = registry.get('web_headers')
      result = tool.call('url' => 'https://example.com/timeout')

      expect(result).to include('Error: Failed to fetch headers')
    end

    it 'returns error message on DNS resolution failure' do
      stub_request(:head, 'https://invalid.example.test/')
        .to_raise(SocketError.new('getaddrinfo: Name or service not known'))

      tool = registry.get('web_headers')
      result = tool.call('url' => 'https://invalid.example.test/')

      expect(result).to include('Error: Failed to fetch headers')
    end
  end

  describe 'parameter validation' do
    it 'requires url parameter' do
      tool = registry.get('web_headers')

      expect { tool.call({}) }.to raise_error
    end
  end
end
