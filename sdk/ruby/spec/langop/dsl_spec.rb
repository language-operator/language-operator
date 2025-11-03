# frozen_string_literal: true

require 'spec_helper'
require 'langop/dsl'

RSpec.describe Langop::Dsl do
  # Clean registry before each test
  before do
    Langop::Dsl.instance_variable_set(:@registry, nil)
  end

  describe '.define' do
    it 'defines a basic tool' do
      Langop::Dsl.define do
        tool 'test_tool' do
          description 'A test tool'
        end
      end

      registry = Langop::Dsl.registry
      expect(registry.all.length).to eq(1)
      expect(registry.get('test_tool')).not_to be_nil
    end

    it 'defines tools with parameters' do
      Langop::Dsl.define do
        tool 'calculator' do
          description 'Simple calculator'

          parameter 'operation' do
            type :string
            description 'Math operation'
            required true
            enum %w[add subtract multiply divide]
          end

          parameter 'a' do
            type :number
            description 'First number'
            required true
          end

          parameter 'b' do
            type :number
            description 'Second number'
            required true
          end

          execute do |params|
            case params['operation']
            when 'add'
              params['a'] + params['b']
            when 'subtract'
              params['a'] - params['b']
            when 'multiply'
              params['a'] * params['b']
            when 'divide'
              params['a'] / params['b']
            end
          end
        end
      end

      registry = Langop::Dsl.registry
      tool_class = registry.get('calculator')

      expect(tool_class).not_to be_nil
      expect(tool_class.tool_definition.name).to eq('calculator')
      expect(tool_class.tool_definition.parameters.keys).to include('operation', 'a', 'b')
    end
  end

  describe 'tool execution' do
    before do
      Langop::Dsl.define do
        tool 'greeter' do
          description 'Greets a user'

          parameter 'name' do
            type :string
            required true
          end

          execute do |params|
            "Hello, #{params['name']}!"
          end
        end
      end
    end

    it 'executes tool with valid parameters' do
      tool_class = Langop::Dsl.registry.get('greeter')
      tool_instance = tool_class.new

      result = tool_instance.call({ 'name' => 'Alice' })
      expect(result[:content][0][:text]).to include('Hello, Alice!')
    end

    it 'validates required parameters' do
      tool_class = Langop::Dsl.registry.get('greeter')
      tool_instance = tool_class.new

      expect { tool_instance.call({}) }.to raise_error(/required/i)
    end
  end

  describe 'parameter types' do
    before do
      Langop::Dsl.define do
        tool 'type_tester' do
          description 'Test parameter types'

          parameter 'string_param' do
            type :string
          end

          parameter 'number_param' do
            type :number
          end

          parameter 'boolean_param' do
            type :boolean
          end

          execute do |params|
            params.inspect
          end
        end
      end
    end

    it 'accepts various parameter types' do
      tool_class = Langop::Dsl.registry.get('type_tester')
      tool_instance = tool_class.new

      result = tool_instance.call({
        'string_param' => 'hello',
        'number_param' => 42,
        'boolean_param' => true
      })

      expect(result[:content][0][:text]).to include('hello')
      expect(result[:content][0][:text]).to include('42')
    end
  end

  describe 'error handling' do
    before do
      Langop::Dsl.define do
        tool 'error_tool' do
          description 'Tool that raises error'

          execute do |_params|
            raise StandardError, 'Intentional error'
          end
        end
      end
    end

    it 'catches and formats errors' do
      tool_class = Langop::Dsl.registry.get('error_tool')
      tool_instance = tool_class.new

      result = tool_instance.call({})
      expect(result[:isError]).to be true
      expect(result[:content][0][:text]).to include('Intentional error')
    end
  end

  describe '.registry' do
    it 'returns global registry' do
      registry = Langop::Dsl.registry
      expect(registry).to be_a(Langop::Dsl::Registry)
    end

    it 'maintains state across calls' do
      registry1 = Langop::Dsl.registry
      registry2 = Langop::Dsl.registry
      expect(registry1).to be(registry2)
    end
  end
end
