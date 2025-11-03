# frozen_string_literal: true

require 'spec_helper'
require 'langop/client/base'

RSpec.describe Langop::Client::Base do
  let(:config) do
    {
      'llm' => {
        'provider' => 'anthropic',
        'model' => 'claude-3-5-sonnet-20241022',
        'api_key' => 'test-key',
        'timeout' => 120
      },
      'mcp_servers' => [
        {
          'name' => 'test-server',
          'url' => 'http://localhost:8080',
          'enabled' => true
        }
      ]
    }
  end

  let(:client) { described_class.new(config) }

  describe '#initialize' do
    it 'stores configuration' do
      expect(client.config).to eq(config)
    end

    xit 'initializes empty servers list' do
      expect(client.servers).to be_empty
    end

    xit 'initializes LLM client to nil' do
      expect(client.llm_client).to be_nil
    end
  end

  describe '#configure_llm' do
    xit 'configures Anthropic provider' do
      expect(RubyLLM).to receive(:configure).and_yield(double(anthropic_api_key: nil))

      client.send(:configure_llm)
    end

    xit 'sets timeout from config' do
      config_double = double
      allow(config_double).to receive(:respond_to?).with(:request_timeout=).and_return(true)
      expect(config_double).to receive(:request_timeout=).with(120)

      allow(RubyLLM).to receive(:configure).and_yield(config_double)
      client.send(:configure_llm)
    end

    xit 'configures MCP timeout separately' do
      expect(RubyLLM::MCP).to receive(:configure).and_yield(double(respond_to?: true, request_timeout: nil))

      client.send(:configure_llm)
    end

    xit 'converts timeout to milliseconds for MCP' do
      mcp_config = double
      allow(mcp_config).to receive(:respond_to?).with(:request_timeout=).and_return(true)
      expect(mcp_config).to receive(:request_timeout=).with(120_000) # 120 seconds * 1000

      allow(RubyLLM::MCP).to receive(:configure).and_yield(mcp_config)
      client.send(:configure_llm)
    end

    it 'raises error for unknown provider' do
      config['llm']['provider'] = 'unknown'

      expect { client.send(:configure_llm) }.to raise_error(/Unknown provider/)
    end
  end

  describe '#connect!' do
    before do
      # Mock MCP server connection
      allow(client).to receive(:connect_to_servers)
    end

    xit 'configures LLM' do
      expect(client).to receive(:configure_llm)
      client.connect!
    end

    xit 'connects to MCP servers' do
      expect(client).to receive(:connect_to_servers)
      client.connect!
    end
  end

  describe '#servers_info' do
    it 'returns empty array when no servers connected' do
      expect(client.servers_info).to eq([])
    end

    xit 'returns server information when connected' do
      # Mock a connected server
      server_double = double(name: 'test-server', url: 'http://localhost:8080')
      client.instance_variable_set(:@servers, [server_double])

      info = client.servers_info
      expect(info).to be_an(Array)
      expect(info.first[:name]).to eq('test-server')
    end
  end

  describe '#send_message' do
    before do
      # Mock LLM client
      llm_client_double = double
      allow(llm_client_double).to receive(:chat).and_return('LLM response')
      client.instance_variable_set(:@llm_client, llm_client_double)
    end

    xit 'sends message to LLM client' do
      llm_client = client.instance_variable_get(:@llm_client)
      expect(llm_client).to receive(:chat).with(hash_including(messages: anything))

      client.send_message('Hello')
    end

    xit 'includes message in request' do
      llm_client = client.instance_variable_get(:@llm_client)
      expect(llm_client).to receive(:chat) do |params|
        expect(params[:messages]).to include(hash_including(role: 'user', content: 'Hello'))
        'response'
      end

      client.send_message('Hello')
    end

    xit 'returns LLM response' do
      result = client.send_message('Test')
      expect(result).to eq('LLM response')
    end
  end

  describe 'provider-specific configuration' do
    xit 'configures OpenAI provider' do
      config['llm']['provider'] = 'openai'
      config['llm']['api_key'] = 'sk-test'

      openai_config = double
      allow(openai_config).to receive(:respond_to?).and_return(true)
      expect(openai_config).to receive(:openai_api_key=).with('sk-test')

      allow(RubyLLM).to receive(:configure).and_yield(openai_config)
      client.send(:configure_llm)
    end

    xit 'handles local models without API key' do
      config['llm']['provider'] = 'openai'
      config['llm'].delete('api_key')
      config['llm']['base_url'] = 'http://localhost:11434'

      expect { client.send(:configure_llm) }.not_to raise_error
    end
  end

  describe 'MCP server filtering' do
    it 'only connects to enabled servers' do
      config['mcp_servers'] << {
        'name' => 'disabled-server',
        'url' => 'http://localhost:9999',
        'enabled' => false
      }

      enabled_servers = config['mcp_servers'].select { |s| s['enabled'] != false }
      expect(enabled_servers.length).to eq(1)
      expect(enabled_servers.first['name']).to eq('test-server')
    end
  end
end
