# frozen_string_literal: true

require 'spec_helper'
require 'langop/dsl'

RSpec.describe Langop::Dsl do
  describe 'tool definition' do
    it 'creates a basic tool' do
      tool = Class.new do
        include Langop::Dsl

        tool 'test_tool' do
          description 'A test tool'
        end
      end

      expect(tool.tool_definition).not_to be_nil
      expect(tool.tool_definition.name).to eq('test_tool')
      expect(tool.tool_definition.description).to eq('A test tool')
    end

    it 'defines parameters with types' do
      tool = Class.new do
        include Langop::Dsl

        tool 'calculator' do
          description 'Simple calculator'

          parameter 'operation' do
            type 'string'
            description 'Math operation'
            required true
            enum %w[add subtract multiply divide]
          end

          parameter 'a' do
            type 'number'
            description 'First number'
            required true
          end

          parameter 'b' do
            type 'number'
            description 'Second number'
            required true
          end
        end
      end

      params = tool.tool_definition.parameters
      expect(params).to have_key('operation')
      expect(params).to have_key('a')
      expect(params).to have_key('b')

      operation_param = params['operation']
      expect(operation_param.type).to eq('string')
      expect(operation_param.required).to be true
      expect(operation_param.enum).to include('add', 'subtract')
    end

    it 'validates required parameters' do
      tool_class = Class.new do
        include Langop::Dsl

        tool 'required_test' do
          parameter 'required_field' do
            type 'string'
            required true
          end

          parameter 'optional_field' do
            type 'string'
            required false
          end
        end

        def execute(params)
          "executed with #{params}"
        end
      end

      tool_instance = tool_class.new

      # Should fail without required parameter
      expect { tool_instance.call({}) }.to raise_error(/required/i)

      # Should succeed with required parameter
      result = tool_instance.call({ 'required_field' => 'value' })
      expect(result[:content][0][:text]).to include('value')
    end
  end

  describe 'parameter types' do
    let(:tool_class) do
      Class.new do
        include Langop::Dsl

        tool 'type_test' do
          parameter 'string_param' do
            type 'string'
          end

          parameter 'number_param' do
            type 'number'
          end

          parameter 'boolean_param' do
            type 'boolean'
          end

          parameter 'array_param' do
            type 'array'
            items type: 'string'
          end
        end

        def execute(params)
          params.inspect
        end
      end
    end

    it 'accepts valid types' do
      tool = tool_class.new

      result = tool.call({
                           'string_param' => 'hello',
                           'number_param' => 42,
                           'boolean_param' => true,
                           'array_param' => %w[a b c]
                         })

      expect(result[:content][0][:text]).to include('hello')
      expect(result[:content][0][:text]).to include('42')
    end
  end

  describe 'execution context' do
    it 'provides context to execute method' do
      tool_class = Class.new do
        include Langop::Dsl

        tool 'context_test' do
          description 'Test context'
        end

        def execute(params)
          "Context available: #{context.class}"
        end
      end

      tool = tool_class.new
      result = tool.call({})

      expect(result[:content][0][:text]).to include('ExecutionContext')
    end
  end

  describe 'error handling' do
    it 'catches and formats errors' do
      tool_class = Class.new do
        include Langop::Dsl

        tool 'error_test' do
          description 'Test error handling'
        end

        def execute(_params)
          raise StandardError, 'Intentional error'
        end
      end

      tool = tool_class.new
      result = tool.call({})

      expect(result[:isError]).to be true
      expect(result[:content][0][:text]).to include('Intentional error')
    end
  end
end
