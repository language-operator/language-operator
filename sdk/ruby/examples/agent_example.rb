#!/usr/bin/env ruby
# frozen_string_literal: true

# Example of agent DSL usage
#
# This demonstrates the agent DSL syntax that will be synthesized
# by the Language Operator from natural language instructions.

require_relative '../lib/langop'

# Define an agent
Langop::Dsl.load_agent_file(__FILE__)

agent "kubernetes-news" do
  description "Daily Kubernetes news summarization agent"

  # Distilled persona (would come from operator synthesis)
  persona <<~PERSONA
    You are a technical writer specializing in Kubernetes. When researching and
    summarizing news, maintain a clear and precise tone, always cite your sources,
    use proper technical terminology, and make complex topics accessible without
    unnecessary jargon. Your summaries should be educational and well-structured.
  PERSONA

  # Extracted from: "once a day, preferably around lunchtime"
  schedule "0 12 * * *"

  # Extracted from instructions
  objectives [
    "Search for recent Kubernetes news using web_search tool",
    "Provide a concise summary of findings"
  ]

  # Synthesized workflow
  workflow do
    step :search do
      tool "web_search"
      params query: "Kubernetes news latest"
    end

    step :summarize do
      depends_on :search
      prompt "Provide a concise summary of these Kubernetes news items: {search.output}"
    end
  end

  # Inferred constraints
  constraints do
    max_iterations 20
    timeout "5m"
  end

  # Output destination
  output do
    workspace "summaries/kubernetes-{date}.md"
  end
end

# When run directly, execute the agent
if __FILE__ == $PROGRAM_NAME
  puts "=" * 60
  puts "Agent DSL Example"
  puts "=" * 60
  puts

  agent = Langop::Dsl.agent_registry.get("kubernetes-news")

  if agent
    puts "✅ Agent loaded: #{agent.name}"
    puts "📝 Description: #{agent.description}"
    puts "🗓️  Schedule: #{agent.schedule}"
    puts "📋 Objectives: #{agent.objectives.size}"
    puts "⚙️  Workflow steps: #{agent.workflow&.steps&.size || 0}"
    puts "📊 Constraints: #{agent.constraints.inspect}"
    puts

    puts "To run this agent:"
    puts "  agent.run!"
    puts

    # Uncomment to actually run:
    # agent.run!
  else
    puts "❌ Failed to load agent"
  end
end
