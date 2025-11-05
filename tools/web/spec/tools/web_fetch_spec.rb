require_relative '../spec_helper'

RSpec.describe 'web_fetch tool' do
  let(:registry) { LanguageOperator::Dsl::Registry.new }
  let(:tool_path) { File.expand_path('../../tools/web.rb', __dir__) }
  let(:example_html) { File.read(File.expand_path('../fixtures/example_page.html', __dir__)) }

  before do
    context = LanguageOperator::Dsl::Context.new(registry)
    code = File.read(tool_path)
    context.instance_eval(code, tool_path)
  end

  describe 'fetching in text mode (default)' do
    it 'returns text content with HTML tags stripped' do
      stub_request(:get, 'https://example.com/')
        .to_return(status: 200, body: example_html)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/')

      expect(result).to include('Content from https://example.com/')
      expect(result).to include('Example Domain')
      expect(result).to include('This domain is for use in documentation')
      expect(result).not_to include('<html>')
      expect(result).not_to include('<body>')
    end

    it 'removes script and style tags' do
      html_with_scripts = <<~HTML
        <html>
        <head>
          <script>alert('test');</script>
          <style>body { color: red; }</style>
        </head>
        <body>
          <p>Visible content</p>
        </body>
        </html>
      HTML

      stub_request(:get, 'https://example.com/test')
        .to_return(status: 200, body: html_with_scripts)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/test')

      expect(result).to include('Visible content')
      expect(result).not_to include('alert')
      expect(result).not_to include('color: red')
    end

    it 'truncates content to 2000 characters' do
      long_content = '<html><body><p>' + ('a' * 3000) + '</p></body></html>'

      stub_request(:get, 'https://example.com/long')
        .to_return(status: 200, body: long_content)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/long')

      # Should include truncation indicator
      expect(result).to include('...')
      # Should not include all 3000 characters
      expect(result.length).to be < 2100
    end

    it 'returns message when content is empty' do
      stub_request(:get, 'https://example.com/empty')
        .to_return(status: 200, body: '<html><body></body></html>')

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/empty')

      expect(result).to eq('No text content found at: https://example.com/empty')
    end
  end

  describe 'fetching in HTML mode' do
    it 'returns raw HTML when html=true' do
      stub_request(:get, 'https://example.com/')
        .to_return(status: 200, body: example_html)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/', 'html' => true)

      expect(result).to include('<!doctype html>')
      expect(result).to include('<h1>Example Domain</h1>')
      expect(result).to include('<style>')
    end

    it 'does not strip tags or truncate in HTML mode' do
      long_html = '<html><body>' + ('<p>test</p>' * 500) + '</body></html>'

      stub_request(:get, 'https://example.com/long')
        .to_return(status: 200, body: long_html)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/long', 'html' => true)

      expect(result).to eq(long_html)
      expect(result).to include('<p>test</p>')
    end
  end

  describe 'URL validation' do
    it 'rejects URLs without http:// or https://' do
      tool = registry.get('web_fetch')
      result = tool.call('url' => 'example.com')

      expect(result).to include('Error: Invalid URL')
      expect(result).to include('Must start with http:// or https://')
    end

    it 'accepts http:// URLs' do
      stub_request(:get, 'http://example.com/')
        .to_return(status: 200, body: example_html)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'http://example.com/')

      expect(result).to include('Content from http://example.com/')
    end

    it 'accepts https:// URLs' do
      stub_request(:get, 'https://example.com/')
        .to_return(status: 200, body: example_html)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/')

      expect(result).to include('Content from https://example.com/')
    end
  end

  describe 'error handling' do
    it 'returns error message when fetch fails with 500' do
      stub_request(:get, 'https://example.com/error')
        .to_return(status: 500, body: 'Internal Server Error')

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/error')

      expect(result).to include('Error: Failed to fetch URL')
      expect(result).to include('https://example.com/error')
    end

    it 'returns error message on network timeout' do
      stub_request(:get, 'https://example.com/timeout')
        .to_timeout

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/timeout')

      expect(result).to include('Error: Failed to fetch URL')
    end

    it 'returns error message on connection refused' do
      stub_request(:get, 'https://example.com/refused')
        .to_raise(Errno::ECONNREFUSED)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/refused')

      expect(result).to include('Error: Failed to fetch URL')
    end
  end

  describe 'redirect handling' do
    it 'follows redirects automatically' do
      stub_request(:get, 'https://example.com/redirect')
        .to_return(status: 301, headers: { 'Location' => 'https://example.com/final' })

      stub_request(:get, 'https://example.com/final')
        .to_return(status: 200, body: example_html)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/redirect')

      expect(result).to include('Content from https://example.com/redirect')
      expect(result).to include('Example Domain')
    end

    it 'handles multiple redirects' do
      stub_request(:get, 'https://example.com/redirect1')
        .to_return(status: 302, headers: { 'Location' => 'https://example.com/redirect2' })

      stub_request(:get, 'https://example.com/redirect2')
        .to_return(status: 302, headers: { 'Location' => 'https://example.com/final' })

      stub_request(:get, 'https://example.com/final')
        .to_return(status: 200, body: example_html)

      tool = registry.get('web_fetch')
      result = tool.call('url' => 'https://example.com/redirect1')

      expect(result).to include('Example Domain')
    end
  end

  describe 'parameter validation' do
    it 'requires url parameter' do
      tool = registry.get('web_fetch')

      expect { tool.call({}) }.to raise_error
    end

    it 'accepts optional html parameter' do
      stub_request(:get, 'https://example.com/')
        .to_return(status: 200, body: example_html)

      tool = registry.get('web_fetch')

      expect { tool.call('url' => 'https://example.com/', 'html' => false) }.not_to raise_error
      expect { tool.call('url' => 'https://example.com/', 'html' => true) }.not_to raise_error
    end
  end
end
