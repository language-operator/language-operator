# frozen_string_literal: true

require 'spec_helper'
require 'language_operator/dsl/execution_context'

RSpec.describe LanguageOperator::Dsl::ExecutionContext do
  let(:params) { { 'input' => 'test', 'email' => 'user@example.com' } }
  let(:context) { described_class.new(params) }

  describe '#initialize' do
    it 'stores parameters' do
      expect(context.instance_variable_get(:@params)).to eq(params)
    end
  end

  describe 'helper methods' do
    describe '#validate_email' do
      it 'validates correct email' do
        expect(context.validate_email('user@example.com')).to be_nil
      end

      it 'rejects invalid email' do
        result = context.validate_email('invalid-email')
        expect(result).to include('Invalid email format')
      end
    end

    describe '#validate_url' do
      it 'validates correct URL' do
        expect(context.validate_url('https://example.com')).to be_nil
      end

      it 'rejects invalid URL' do
        result = context.validate_url('not-a-url')
        expect(result).to include('Invalid URL')
      end
    end

    describe '#validate_phone' do
      it 'validates E.164 phone number' do
        expect(context.validate_phone('+1234567890')).to be_nil
      end

      it 'rejects invalid phone number' do
        result = context.validate_phone('123')
        expect(result).to include('Invalid phone number format')
      end
    end

    describe '#run_command' do
      it 'executes shell command' do
        result = context.run_command('echo "hello"')
        expect(result[:success]).to be true
        expect(result[:output]).to include('hello')
      end

      it 'captures exit codes' do
        result = context.run_command('false')
        expect(result[:success]).to be false
        expect(result[:exitcode]).not_to eq(0)
      end
    end

    describe '#env_get' do
      it 'retrieves environment variables' do
        ENV['TEST_VAR'] = 'test_value'
        expect(context.env_get('TEST_VAR')).to eq('test_value')
        ENV.delete('TEST_VAR')
      end

      it 'returns default for missing variables' do
        expect(context.env_get('NONEXISTENT_VAR', default: 'default')).to eq('default')
      end

      it 'tries multiple keys' do
        ENV['SECOND_VAR'] = 'second_value'
        result = context.env_get('FIRST_VAR', 'SECOND_VAR', default: 'default')
        expect(result).to eq('second_value')
        ENV.delete('SECOND_VAR')
      end
    end

    describe '#env_required' do
      it 'returns nil when all variables present' do
        ENV['REQUIRED1'] = 'value1'
        ENV['REQUIRED2'] = 'value2'
        expect(context.env_required('REQUIRED1', 'REQUIRED2')).to be_nil
        ENV.delete('REQUIRED1')
        ENV.delete('REQUIRED2')
      end

      it 'returns error for missing variables' do
        result = context.env_required('MISSING_VAR')
        expect(result).to include('Missing required environment variables')
      end
    end

    describe '#truncate' do
      it 'truncates long text' do
        long_text = 'a' * 3000
        result = context.truncate(long_text, max_length: 100)
        expect(result.length).to be <= 103 # 100 + '...'
      end

      it 'leaves short text unchanged' do
        short_text = 'short'
        expect(context.truncate(short_text)).to eq('short')
      end
    end

    describe '#parse_csv' do
      it 'parses comma-separated values' do
        result = context.parse_csv('a, b, c')
        expect(result).to eq(%w[a b c])
      end

      it 'handles empty string' do
        expect(context.parse_csv('')).to eq([])
      end

      it 'handles nil' do
        expect(context.parse_csv(nil)).to eq([])
      end
    end

    describe '#error and #success' do
      it 'formats error messages' do
        expect(context.error('something failed')).to eq('Error: something failed')
      end

      it 'formats success messages' do
        expect(context.success('operation completed')).to eq('operation completed')
      end
    end
  end
end
