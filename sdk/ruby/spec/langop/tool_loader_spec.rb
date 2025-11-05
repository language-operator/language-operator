# frozen_string_literal: true

require 'spec_helper'
require 'aictl/tool_loader'

RSpec.describe Aictl::ToolLoader do
  let(:tool_dir) { File.join(__dir__, '../fixtures/tools') }
  let(:registry) { Aictl::Dsl::Registry.new }
  let(:loader) { described_class.new(registry, tool_dir) }

  before do
    # Create fixture tools directory
    FileUtils.mkdir_p(tool_dir)

    # Create a sample tool file
    File.write(File.join(tool_dir, 'sample_tool.rb'), <<~RUBY)
      tool 'sample' do
        description 'A sample tool for testing'

        parameter 'input' do
          type 'string'
          required true
        end
      end

      def execute(params)
        "Sample tool executed with: \#{params['input']}"
      end
    RUBY

    # Create another tool file
    File.write(File.join(tool_dir, 'math_tool.rb'), <<~RUBY)
      tool 'add' do
        description 'Adds two numbers'

        parameter 'a' do
          type 'number'
          required true
        end

        parameter 'b' do
          type 'number'
          required true
        end
      end

      def execute(params)
        result = params['a'] + params['b']
        "Result: \#{result}"
      end
    RUBY
  end

  after do
    # Clean up fixture files
    FileUtils.rm_rf(tool_dir)
  end

  describe '#load_tools' do
    it 'loads all tool files from directory' do
      loader.load_tools

      expect(registry.all.length).to be >= 2
      expect(registry.get('sample')).not_to be_nil
      expect(registry.get('add')).not_to be_nil
    end

    xit 'creates tool instances with correct definitions' do
      loader.load_tools

      sample_tool = registry.get('sample')

      expect(sample_tool.tool_definition.name).to eq('sample')
      expect(sample_tool.tool_definition.description).to eq('A sample tool for testing')
      expect(sample_tool.tool_definition.parameters).to have_key('input')
    end

    it 'skips non-Ruby files' do
      File.write(File.join(tool_dir, 'readme.txt'), 'Not a tool')

      expect { loader.load_tools }.not_to raise_error
    end

    it 'handles subdirectories' do
      subdir = File.join(tool_dir, 'advanced')
      FileUtils.mkdir_p(subdir)

      File.write(File.join(subdir, 'advanced_tool.rb'), <<~RUBY)
        tool 'advanced' do
          description 'Advanced tool'
        end

        def execute(params)
          'advanced'
        end
      RUBY

      loader.load_tools

      expect(registry.get('advanced')).not_to be_nil
    end
  end

  describe '#reload_tools' do
    xit 'reloads tools when files change' do
      loader.load_tools

      # Modify a tool file
      File.write(File.join(tool_dir, 'sample_tool.rb'), <<~RUBY)
        tool 'sample' do
          description 'Modified description'
          parameter 'input' do
            type 'string'
            required true
          end
        end

        def execute(params)
          "Modified: \#{params['input']}"
        end
      RUBY

      loader.reload_tools

      sample_tool = registry.get('sample')
      expect(sample_tool.tool_definition.description).to eq('Modified description')
    end
  end

  describe 'error handling' do
    it 'reports tools with syntax errors' do
      File.write(File.join(tool_dir, 'broken_tool.rb'), <<~RUBY)
        tool 'broken' do
          description 'This tool has a syntax error
        end
      RUBY

      expect { loader.load_tools }.to raise_error(/syntax error/i)
    end

    it 'handles missing tool directory gracefully' do
      loader = described_class.new(registry, '/nonexistent/path')

      expect { loader.load_tools }.not_to raise_error  # It just skips loading
    end
  end
end
