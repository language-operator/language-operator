require_relative '../spec_helper'

RSpec.describe 'web_headers tool' do
  let(:registry) { Langop::Dsl::Registry.new }
  let(:tool_path) { File.expand_path('../../tools/web.rb', __dir__) }
  let(:example_headers) { File.read(File.expand_path('../fixtures/example_headers.txt', __dir__)) }

  before do
    load tool_path
    Langop::Dsl.apply_to(registry)
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
      expect(result).to include('HTTP')
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
    it 'returns error message when fetch fails' do
      stub_request(:head, 'https://example.com/error')
        .to_return(status: 500, body: '')

      tool = registry.get('web_headers')

      # Mock curl failure
      allow(tool).to receive(:`).and_return('')
      allow($?).to receive(:success?).and_return(false)

      result = tool.call('url' => 'https://example.com/error')

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
