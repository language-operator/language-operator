#!/usr/bin/env ruby
# frozen_string_literal: true

# Quick test of the agent DSL without requiring full langop gem dependencies

require_relative 'lib/langop/dsl/agent_definition'
require_relative 'lib/langop/dsl/agent_context'
require_relative 'lib/langop/dsl/workflow_definition'

puts "=" * 70
puts "Agent DSL Test"
puts "=" * 70
puts

# Create registry and context
registry = Langop::Dsl::AgentRegistry.new
context = Langop::Dsl::AgentContext.new(registry)

# Define an agent using the DSL (this is what the operator will synthesize)
context.agent "kubernetes-news" do
  description "Daily Kubernetes news summarization agent"

  # Persona (distilled by operator from LanguagePersona)
  persona <<~PERSONA
    You are a technical writer specializing in Kubernetes. When researching and
    summarizing news, maintain a clear and precise tone, always cite your sources,
    use proper technical terminology, and make complex topics accessible without
    unnecessary jargon. Your summaries should be educational and well-structured.
  PERSONA

  # Schedule (extracted from: "once a day, preferably around lunchtime")
  schedule "0 12 * * *"

  # Objectives (extracted from instructions)
  objectives [
    "Search for recent Kubernetes news using web_search tool",
    "Provide a concise summary of findings"
  ]

  # Workflow (synthesized by operator)
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

  # Constraints (inferred by operator)
  constraints do
    max_iterations 20
    timeout "5m"
  end

  # Output (inferred from workspace settings)
  output do
    workspace "summaries/kubernetes-{date}.md"
  end
end

# Retrieve and display the agent
agent = registry.get("kubernetes-news")

if agent
  puts "✅ Agent Definition Loaded"
  puts
  puts "Name:         #{agent.name}"
  puts "Description:  #{agent.description}"
  puts "Mode:         #{agent.execution_mode}"
  puts "Schedule:     #{agent.schedule}"
  puts
  puts "📋 Persona:"
  puts agent.persona.lines.map { |l| "  #{l}" }.join
  puts
  puts "🎯 Objectives (#{agent.objectives.size}):"
  agent.objectives.each_with_index do |obj, i|
    puts "  #{i + 1}. #{obj}"
  end
  puts
  puts "⚙️  Workflow Steps (#{agent.workflow.steps.size}):"
  agent.workflow.steps.each do |name, step|
    deps = step.dependencies.empty? ? "" : " [depends on: #{step.dependencies.join(', ')}]"
    tool_info = step.tool ? " (tool: #{step.tool})" : ""
    prompt_info = step.prompt ? " (prompt: #{step.prompt[0..50]}...)" : ""
    puts "  - #{name}#{deps}#{tool_info}#{prompt_info}"
  end
  puts
  puts "📊 Constraints:"
  agent.constraints.each do |key, value|
    puts "  #{key}: #{value}"
  end
  puts
  puts "📤 Output:"
  agent.output_config.each do |key, value|
    puts "  #{key}: #{value}"
  end
  puts
  puts "=" * 70
  puts "✅ DSL Test Complete - Agent ready for execution!"
  puts "=" * 70
else
  puts "❌ Failed to load agent"
  exit 1
end
