require_relative '../spec_helper'

RSpec.describe 'web_status tool' do
  let(:registry) { LanguageOperator::Dsl::Registry.new }
  let(:tool_path) { File.expand_path('../../tools/web.rb', __dir__) }

  before do
    context = LanguageOperator::Dsl::Context.new(registry)
    code = File.read(tool_path)
    context.instance_eval(code, tool_path)
  end

  describe 'successful status checks' do
    it 'returns 200 OK status' do
      stub_request(:get, 'https://example.com/')
        .to_return(status: 200)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/')

      expect(result).to eq('Status for https://example.com/: 200 OK')
    end

    it 'returns 301 Moved Permanently' do
      stub_request(:get, 'https://example.com/old')
        .to_return(status: 301)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/old')

      expect(result).to eq('Status for https://example.com/old: 301 Moved Permanently')
    end

    it 'returns 302 Found (Redirect)' do
      stub_request(:get, 'https://example.com/redirect')
        .to_return(status: 302)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/redirect')

      expect(result).to eq('Status for https://example.com/redirect: 302 Found (Redirect)')
    end

    it 'returns 304 Not Modified' do
      stub_request(:get, 'https://example.com/cached')
        .to_return(status: 304)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/cached')

      expect(result).to eq('Status for https://example.com/cached: 304 Not Modified')
    end

    it 'returns 400 Bad Request' do
      stub_request(:get, 'https://example.com/bad')
        .to_return(status: 400)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/bad')

      expect(result).to eq('Status for https://example.com/bad: 400 Bad Request')
    end

    it 'returns 401 Unauthorized' do
      stub_request(:get, 'https://example.com/auth')
        .to_return(status: 401)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/auth')

      expect(result).to eq('Status for https://example.com/auth: 401 Unauthorized')
    end

    it 'returns 403 Forbidden' do
      stub_request(:get, 'https://example.com/forbidden')
        .to_return(status: 403)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/forbidden')

      expect(result).to eq('Status for https://example.com/forbidden: 403 Forbidden')
    end

    it 'returns 404 Not Found' do
      stub_request(:get, 'https://example.com/missing')
        .to_return(status: 404)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/missing')

      expect(result).to eq('Status for https://example.com/missing: 404 Not Found')
    end

    it 'returns 500 Internal Server Error' do
      stub_request(:get, 'https://example.com/error')
        .to_return(status: 500)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/error')

      expect(result).to eq('Status for https://example.com/error: 500 Internal Server Error')
    end

    it 'returns 502 Bad Gateway' do
      stub_request(:get, 'https://example.com/gateway')
        .to_return(status: 502)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/gateway')

      expect(result).to eq('Status for https://example.com/gateway: 502 Bad Gateway')
    end

    it 'returns 503 Service Unavailable' do
      stub_request(:get, 'https://example.com/unavailable')
        .to_return(status: 503)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/unavailable')

      expect(result).to eq('Status for https://example.com/unavailable: 503 Service Unavailable')
    end

    it 'returns Unknown for unmapped status codes' do
      stub_request(:get, 'https://example.com/teapot')
        .to_return(status: 418)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/teapot')

      expect(result).to eq('Status for https://example.com/teapot: 418 Unknown')
    end
  end

  describe 'URL validation' do
    it 'rejects URLs without http:// or https://' do
      tool = registry.get('web_status')
      result = tool.call('url' => 'example.com')

      expect(result).to include('Error: Invalid URL')
      expect(result).to include('Must start with http:// or https://')
    end

    it 'accepts http:// URLs' do
      stub_request(:get, 'http://example.com/')
        .to_return(status: 200)

      tool = registry.get('web_status')
      result = tool.call('url' => 'http://example.com/')

      expect(result).to include('Status for http://example.com/')
    end

    it 'accepts https:// URLs' do
      stub_request(:get, 'https://example.com/')
        .to_return(status: 200)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/')

      expect(result).to include('Status for https://example.com/')
    end
  end

  describe 'error handling' do
    it 'returns error message on network timeout' do
      stub_request(:get, 'https://example.com/timeout')
        .to_timeout

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/timeout')

      expect(result).to include('Status for https://example.com/timeout: 0')
    end

    it 'returns error message on connection refused' do
      stub_request(:get, 'https://example.com/refused')
        .to_raise(Errno::ECONNREFUSED)

      tool = registry.get('web_status')
      result = tool.call('url' => 'https://example.com/refused')

      expect(result).to include('Status for https://example.com/refused: 0')
    end
  end

  describe 'parameter validation' do
    it 'requires url parameter' do
      tool = registry.get('web_status')

      expect { tool.call({}) }.to raise_error
    end
  end
end
