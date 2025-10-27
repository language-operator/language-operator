module Based
  module Dsl
    class ParameterDefinition
      attr_reader :name, :type, :required, :description, :enum, :default, :validator

      def initialize(name)
        @name = name
        @required = false
        @validator = nil
      end

      def type(val)
        @type = val
      end

      def required(val = true)
        @required = val
      end

      def description(val)
        @description = val
      end

      def enum(val)
        @enum = val
      end

      def default(val)
        @default = val
      end

      # Custom validation with proc or regex
      def validate(proc_or_regex)
        @validator = proc_or_regex
      end

      # Built-in validators
      def url_format
        @validator = /^https?:\/\//
      end

      def email_format
        @validator = /\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i
      end

      def phone_format
        @validator = /^\+\d{10,15}$/
      end

      # Validate a value against this parameter's validator
      def validate_value(value)
        return nil unless @validator

        case @validator
        when Regexp
          unless value.to_s =~ @validator
            return "Parameter '#{@name}' has invalid format"
          end
        when Proc
          result = @validator.call(value)
          return result if result.is_a?(String) # Error message
          return "Parameter '#{@name}' validation failed" unless result
        end

        nil # No error
      end

      def to_schema
        schema = {
          "type" => map_type(@type),
          "description" => @description
        }
        schema["enum"] = @enum if @enum
        schema["default"] = @default if @default
        schema
      end

      private

      def map_type(ruby_type)
        case ruby_type
        when :string then "string"
        when :number, :integer then "number"
        when :boolean then "boolean"
        when :array then "array"
        when :object then "object"
        else "string"
        end
      end
    end
  end
end
