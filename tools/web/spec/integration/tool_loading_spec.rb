require_relative '../spec_helper'

RSpec.describe 'Tool Loading Integration' do
  let(:registry) { LanguageOperator::Dsl::Registry.new }
  let(:tool_path) { File.expand_path('../../tools/web.rb', __dir__) }

  before do
    context = LanguageOperator::Dsl::Context.new(registry)
    code = File.read(tool_path)
    context.instance_eval(code, tool_path)
  end

  describe 'tool registration' do
    it 'loads all 4 web tools' do
      expect(registry.get('web_search')).not_to be_nil
      expect(registry.get('web_fetch')).not_to be_nil
      expect(registry.get('web_headers')).not_to be_nil
      expect(registry.get('web_status')).not_to be_nil
    end

    it 'tools have correct metadata' do
      search_tool = registry.get('web_search')
      expect(search_tool.name).to eq('web_search')
      expect(search_tool.description).to include('Search the web')

      fetch_tool = registry.get('web_fetch')
      expect(fetch_tool.name).to eq('web_fetch')
      expect(fetch_tool.description).to include('Fetch and extract')

      headers_tool = registry.get('web_headers')
      expect(headers_tool.name).to eq('web_headers')
      expect(headers_tool.description).to include('HTTP headers')

      status_tool = registry.get('web_status')
      expect(status_tool.name).to eq('web_status')
      expect(status_tool.description).to include('HTTP status')
    end
  end

  describe 'parameter definitions' do
    it 'web_search has required query parameter' do
      tool = registry.get('web_search')
      params = tool.parameters

      query_param = params['query']
      expect(query_param).not_to be_nil
      expect(query_param.instance_variable_get(:@required)).to be true
      expect(query_param.instance_variable_get(:@type)).to eq(:string)
    end

    it 'web_search has optional max_results parameter' do
      tool = registry.get('web_search')
      params = tool.parameters

      max_param = params['max_results']
      expect(max_param).not_to be_nil
      expect(max_param.instance_variable_get(:@required)).to be false
      expect(max_param.instance_variable_get(:@type)).to eq(:number)
      expect(max_param.instance_variable_get(:@default)).to eq(5)
    end

    it 'web_fetch has required url parameter' do
      tool = registry.get('web_fetch')
      params = tool.parameters

      url_param = params['url']
      expect(url_param).not_to be_nil
      expect(url_param.instance_variable_get(:@required)).to be true
      expect(url_param.instance_variable_get(:@type)).to eq(:string)
    end

    it 'web_fetch has optional html parameter' do
      tool = registry.get('web_fetch')
      params = tool.parameters

      html_param = params['html']
      expect(html_param).not_to be_nil
      expect(html_param.instance_variable_get(:@required)).to be false
      expect(html_param.instance_variable_get(:@type)).to eq(:boolean)
      expect(html_param.instance_variable_get(:@default)).to be false
    end
  end
end
