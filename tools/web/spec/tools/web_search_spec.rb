require_relative '../spec_helper'

RSpec.describe 'web_search tool' do
  let(:registry) { LanguageOperator::Dsl::Registry.new }
  let(:tool_path) { File.expand_path('../../tools/web.rb', __dir__) }
  let(:ddg_search_html) { File.read(File.expand_path('../fixtures/ddg_search_ruby.html', __dir__)) }
  let(:ddg_no_results_html) { File.read(File.expand_path('../fixtures/ddg_no_results.html', __dir__)) }

  before do
    context = LanguageOperator::Dsl::Context.new(registry)
    code = File.read(tool_path)
    context.instance_eval(code, tool_path)
  end

  describe 'successful search with results' do
    it 'returns formatted search results' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=ruby/)
        .to_return(status: 200, body: ddg_search_html)

      tool = registry.get('web_search')
      result = tool.call('query' => 'ruby', 'max_results' => 5)

      expect(result).to include('Search results for: ruby')
      expect(result).to include('Ruby Programming Language')
      expect(result).to include('https://www.ruby-lang.org/en/')
      expect(result).to include('Ruby on GitHub')
      expect(result).to include('https://github.com/ruby/ruby')
    end

    it 'respects max_results limit' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=ruby/)
        .to_return(status: 200, body: ddg_search_html)

      tool = registry.get('web_search')
      result = tool.call('query' => 'ruby', 'max_results' => 2)

      # Count the number of results (each result has a "URL:" line)
      url_count = result.scan(/URL:/).length
      expect(url_count).to eq(2)
    end

    it 'uses default max_results of 5 when not specified' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=ruby/)
        .to_return(status: 200, body: ddg_search_html)

      tool = registry.get('web_search')
      result = tool.call('query' => 'ruby')

      expect(result).to include('Search results for: ruby')
      expect(result).to include('Ruby Programming Language')
    end
  end

  describe 'search with no results' do
    it 'returns no results message' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=nosuchquery123xyz/)
        .to_return(status: 200, body: ddg_no_results_html)

      tool = registry.get('web_search')
      result = tool.call('query' => 'nosuchquery123xyz')

      expect(result).to eq('No results found for: nosuchquery123xyz')
    end
  end

  describe 'query encoding' do
    it 'properly encodes spaces in query' do
      stub_request(:get, 'https://html.duckduckgo.com/html/?q=ruby+programming')
        .to_return(status: 200, body: ddg_search_html)

      tool = registry.get('web_search')
      result = tool.call('query' => 'ruby programming')

      expect(result).to include('Search results for: ruby programming')
    end

    it 'properly encodes special characters' do
      stub_request(:get, %r{html\.duckduckgo\.com/html/\?q=ruby%26rails})
        .to_return(status: 200, body: ddg_no_results_html)

      tool = registry.get('web_search')
      result = tool.call('query' => 'ruby&rails')

      expect(result).to include('ruby&rails')
    end
  end

  describe 'URL cleanup' do
    it 'removes DuckDuckGo redirect wrapper from URLs' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=test/)
        .to_return(status: 200, body: ddg_search_html)

      tool = registry.get('web_search')
      result = tool.call('query' => 'test')

      # Should not contain DuckDuckGo redirect URLs
      expect(result).not_to include('duckduckgo.com/l/')
      expect(result).not_to include('uddg=')

      # Should contain clean URLs
      expect(result).to include('https://www.ruby-lang.org/en/')
    end
  end

  describe 'error handling' do
    it 'returns error message when HTTP request fails' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=test/)
        .to_return(status: 500, body: 'Internal Server Error')

      tool = registry.get('web_search')
      result = tool.call('query' => 'test')

      expect(result).to include('Error')
      expect(result).to include('Failed to fetch search results')
    end

    it 'returns error message on network timeout' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=timeout/)
        .to_timeout

      tool = registry.get('web_search')
      result = tool.call('query' => 'timeout')

      expect(result).to include('Error')
    end
  end

  describe 'parameter validation' do
    it 'requires query parameter' do
      tool = registry.get('web_search')

      expect { tool.call({}) }.to raise_error
    end

    it 'accepts optional max_results parameter' do
      stub_request(:get, /html\.duckduckgo\.com\/html\/\?q=ruby/)
        .to_return(status: 200, body: ddg_search_html)

      tool = registry.get('web_search')

      expect { tool.call('query' => 'ruby', 'max_results' => 3) }.not_to raise_error
    end
  end
end
