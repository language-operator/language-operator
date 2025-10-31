module Based
  module Dsl
    class Registry
      def initialize
        @tools = {}
      end

      def register(tool)
        @tools[tool.name] = tool
      end

      def get(name)
        @tools[name]
      end

      def all
        @tools.values
      end

      def clear
        @tools.clear
      end
    end
  end
end
