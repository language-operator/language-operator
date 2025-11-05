# frozen_string_literal: true

require 'spec_helper'
require 'aictl/dsl/agent_definition'
require 'aictl/dsl/agent_context'
require 'aictl/dsl/workflow_definition'

RSpec.describe Aictl::Dsl::AgentDefinition do
  describe 'agent DSL' do
    it 'defines a basic agent' do
      registry = Aictl::Dsl::AgentRegistry.new
      context = Aictl::Dsl::AgentContext.new(registry)

      context.agent 'test-agent' do
        description 'A test agent'
        schedule '0 12 * * *'
        objectives ['Goal 1', 'Goal 2']
      end

      agent = registry.get('test-agent')
      expect(agent).not_to be_nil
      expect(agent.name).to eq('test-agent')
      expect(agent.description).to eq('A test agent')
      expect(agent.schedule).to eq('0 12 * * *')
      expect(agent.objectives).to eq(['Goal 1', 'Goal 2'])
      expect(agent.execution_mode).to eq(:scheduled)
    end

    it 'defines agent with persona' do
      registry = Aictl::Dsl::AgentRegistry.new
      context = Aictl::Dsl::AgentContext.new(registry)

      context.agent 'persona-agent' do
        persona 'You are a helpful assistant'
      end

      agent = registry.get('persona-agent')
      expect(agent.persona).to eq('You are a helpful assistant')
    end

    it 'defines agent with workflow' do
      registry = Aictl::Dsl::AgentRegistry.new
      context = Aictl::Dsl::AgentContext.new(registry)

      context.agent 'workflow-agent' do
        workflow do
          step :search, tool: 'web_search', params: { query: 'test' }
          step :summarize, depends_on: :search
        end
      end

      agent = registry.get('workflow-agent')
      expect(agent.workflow).not_to be_nil
      expect(agent.workflow.steps.size).to eq(2)
      expect(agent.workflow.steps.keys).to eq([:search, :summarize])
    end

    it 'defines agent with constraints' do
      registry = Aictl::Dsl::AgentRegistry.new
      context = Aictl::Dsl::AgentContext.new(registry)

      context.agent 'constrained-agent' do
        constraints do
          max_iterations 100
          timeout '30m'
          memory '2Gi'
        end
      end

      agent = registry.get('constrained-agent')
      expect(agent.constraints[:max_iterations]).to eq(100)
      expect(agent.constraints[:timeout]).to eq('30m')
      expect(agent.constraints[:memory]).to eq('2Gi')
    end

    it 'defines agent with output configuration' do
      registry = Aictl::Dsl::AgentRegistry.new
      context = Aictl::Dsl::AgentContext.new(registry)

      context.agent 'output-agent' do
        output do
          workspace 'results/output.md'
        end
      end

      agent = registry.get('output-agent')
      expect(agent.output_config[:workspace]).to eq('results/output.md')
    end

    it 'supports adding objectives individually' do
      registry = Aictl::Dsl::AgentRegistry.new
      context = Aictl::Dsl::AgentContext.new(registry)

      context.agent 'multi-objective' do
        objective 'First goal'
        objective 'Second goal'
        objective 'Third goal'
      end

      agent = registry.get('multi-objective')
      expect(agent.objectives).to eq(['First goal', 'Second goal', 'Third goal'])
    end
  end

  describe Aictl::Dsl::AgentRegistry do
    it 'manages multiple agents' do
      registry = Aictl::Dsl::AgentRegistry.new

      agent1 = Aictl::Dsl::AgentDefinition.new('agent1')
      agent2 = Aictl::Dsl::AgentDefinition.new('agent2')

      registry.register(agent1)
      registry.register(agent2)

      expect(registry.count).to eq(2)
      expect(registry.get('agent1')).to eq(agent1)
      expect(registry.get('agent2')).to eq(agent2)
      expect(registry.all.length).to eq(2)
    end

    it 'clears all agents' do
      registry = Aictl::Dsl::AgentRegistry.new
      registry.register(Aictl::Dsl::AgentDefinition.new('agent1'))

      expect(registry.count).to eq(1)

      registry.clear

      expect(registry.count).to eq(0)
    end
  end

  describe Aictl::Dsl::WorkflowDefinition do
    it 'defines workflow steps' do
      workflow = Aictl::Dsl::WorkflowDefinition.new

      workflow.step :search, tool: 'web_search', params: { query: 'test' }
      workflow.step :analyze, depends_on: :search

      expect(workflow.steps.size).to eq(2)
      expect(workflow.steps[:search]).not_to be_nil
      expect(workflow.steps[:analyze]).not_to be_nil
    end

    it 'tracks step dependencies' do
      workflow = Aictl::Dsl::WorkflowDefinition.new

      workflow.step :step1 do
        tool 'tool1'
      end

      workflow.step :step2 do
        depends_on :step1
        tool 'tool2'
      end

      step2 = workflow.steps[:step2]
      expect(step2.dependencies).to eq([:step1])
    end

    it 'supports prompt-based steps' do
      workflow = Aictl::Dsl::WorkflowDefinition.new

      workflow.step :summarize do
        prompt 'Summarize: {previous.output}'
      end

      step = workflow.steps[:summarize]
      expect(step.prompt).to eq('Summarize: {previous.output}')
    end
  end
end
