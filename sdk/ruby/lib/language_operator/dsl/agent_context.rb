# frozen_string_literal: true

module LanguageOperator
  module Dsl
    # DSL context for defining agents
    #
    # Provides the evaluation context for agent definition files. Agents are
    # defined using the `agent` method within this context.
    #
    # @example Agent definition file
    #   agent "news-summarizer" do
    #     description "Daily news summarization"
    #
    #     schedule "0 12 * * *"
    #
    #     objectives [
    #       "Search for recent news",
    #       "Summarize findings"
    #     ]
    #
    #     workflow do
    #       step :search, tool: "web_search"
    #       step :summarize, depends_on: :search
    #     end
    #   end
    class AgentContext
      # Initialize context with registry
      #
      # @param registry [LanguageOperator::Dsl::AgentRegistry] Agent registry
      def initialize(registry)
        @registry = registry
      end

      # Define an agent
      #
      # @param name [String] Agent name
      # @yield Agent definition block
      # @return [void]
      def agent(name, &block)
        agent_def = AgentDefinition.new(name)
        agent_def.instance_eval(&block) if block_given?
        @registry.register(agent_def)
      end
    end

    # Registry for agents (similar to tool registry)
    class AgentRegistry
      def initialize
        @agents = {}
      end

      # Register an agent
      #
      # @param agent_def [AgentDefinition] Agent definition
      # @return [void]
      def register(agent_def)
        @agents[agent_def.name] = agent_def
      end

      # Get an agent by name
      #
      # @param name [String] Agent name
      # @return [AgentDefinition, nil] Agent definition
      def get(name)
        @agents[name.to_s]
      end

      # Get all agents
      #
      # @return [Array<AgentDefinition>] All registered agents
      def all
        @agents.values
      end

      # Clear all agents
      #
      # @return [void]
      def clear
        @agents.clear
      end

      # Get count of registered agents
      #
      # @return [Integer] Number of agents
      def count
        @agents.size
      end
    end
  end
end
