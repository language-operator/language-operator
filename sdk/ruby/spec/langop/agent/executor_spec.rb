# frozen_string_literal: true

require 'spec_helper'
require 'aictl/agent/executor'

RSpec.describe Aictl::Agent::Executor do
  let(:agent_double) do
    instance_double(
      Aictl::Agent::Base,
      workspace_path: '/workspace',
      servers_info: [{ name: 'test-server', url: 'http://localhost:8080' }],
      config: { 'agent' => { 'instructions' => 'Test instructions' } },
      send_message: 'Test response'
    )
  end

  let(:executor) { described_class.new(agent_double) }

  describe '#initialize' do
    it 'sets iteration count to zero' do
      expect(executor.instance_variable_get(:@iteration_count)).to eq(0)
    end

    it 'sets default max iterations' do
      expect(executor.instance_variable_get(:@max_iterations)).to eq(100)
    end

    it 'initializes logger' do
      logger = executor.instance_variable_get(:@logger)
      expect(logger).to be_a(Aictl::Logger)
    end

    it 'reads SHOW_FULL_RESPONSES from environment' do
      ENV['SHOW_FULL_RESPONSES'] = 'true'
      exec = described_class.new(agent_double)
      expect(exec.instance_variable_get(:@show_full_responses)).to be true
      ENV.delete('SHOW_FULL_RESPONSES')
    end
  end

  describe '#execute' do
    it 'increments iteration count' do
      expect {
        executor.execute('test task')
      }.to change { executor.instance_variable_get(:@iteration_count) }.by(1)
    end

    it 'calls agent send_message' do
      expect(agent_double).to receive(:send_message).with('test task')
      executor.execute('test task')
    end

    it 'returns result from agent' do
      allow(agent_double).to receive(:send_message).and_return('Success')
      result = executor.execute('test task')
      expect(result).to eq('Success')
    end

    it 'handles errors gracefully' do
      allow(agent_double).to receive(:send_message).and_raise(StandardError, 'LLM error')

      result = executor.execute('test task')
      expect(result).to include('Error executing task')
      expect(result).to include('LLM error')
    end

    it 'handles timeout errors' do
      allow(agent_double).to receive(:send_message).and_raise(Timeout::Error)

      result = executor.execute('test task')
      expect(result).to include('Error executing task')
    end

    it 'handles connection errors' do
      allow(agent_double).to receive(:send_message).and_raise(StandardError, 'Connection refused')

      result = executor.execute('test task')
      expect(result).to include('Error executing task')
      expect(result).to include('Connection refused')
    end
  end

  describe '#run_loop' do
    before do
      # Mock sleep to avoid delays
      allow(executor).to receive(:sleep)
    end

    it 'executes instructions from config' do
      allow(agent_double).to receive(:send_message).and_return('Iteration result')

      # Set max iterations to 1 to avoid infinite loop
      executor.instance_variable_set(:@max_iterations, 1)

      expect(agent_double).to receive(:send_message).with('Test instructions')
      executor.run_loop
    end

    it 'respects max iterations limit' do
      allow(agent_double).to receive(:send_message).and_return('Result')

      executor.instance_variable_set(:@max_iterations, 3)

      expect(agent_double).to receive(:send_message).exactly(3).times
      executor.run_loop
    end

    it 'uses AGENT_INSTRUCTIONS environment variable' do
      ENV['AGENT_INSTRUCTIONS'] = 'Custom instructions'
      allow(agent_double).to receive(:config).and_return({})
      allow(agent_double).to receive(:send_message).and_return('Result')

      executor.instance_variable_set(:@max_iterations, 1)
      expect(agent_double).to receive(:send_message).with('Custom instructions')

      executor.run_loop
      ENV.delete('AGENT_INSTRUCTIONS')
    end

    it 'falls back to default instructions' do
      allow(agent_double).to receive(:config).and_return({})
      allow(agent_double).to receive(:send_message).and_return('Result')

      executor.instance_variable_set(:@max_iterations, 1)
      expect(agent_double).to receive(:send_message).with('Monitor workspace and respond to changes')

      executor.run_loop
    end

    it 'logs server connection info' do
      allow(agent_double).to receive(:send_message).and_return('Result')
      executor.instance_variable_set(:@max_iterations, 1)

      logger = executor.instance_variable_get(:@logger)
      expect(logger).to receive(:info).at_least(:once)

      executor.run_loop
    end
  end

  describe 'error categorization' do
    let(:handle_error) { executor.send(:handle_error, error) }

    context 'with timeout error' do
      let(:error) { Timeout::Error.new }

      it 'categorizes as timeout' do
        logger = executor.instance_variable_get(:@logger)
        expect(logger).to receive(:error).with('Request timeout', any_args)
        handle_error
      end
    end

    context 'with connection error' do
      let(:error) { StandardError.new('Connection refused') }

      xit 'categorizes as connection failure' do
        logger = executor.instance_variable_get(:@logger)
        expect(logger).to receive(:error).with('Connection failed', any_args)
        handle_error
      end
    end

    context 'with generic error' do
      let(:error) { StandardError.new('Generic error') }

      it 'categorizes as task execution failed' do
        logger = executor.instance_variable_get(:@logger)
        expect(logger).to receive(:error).with('Task execution failed', any_args)
        handle_error
      end
    end
  end
end
