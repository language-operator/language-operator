# frozen_string_literal: true

require 'langop/dsl'
require 'langop/tool_loader'

# Based DSL - Namespace wrapper for backwards compatibility
#
# This module now wraps the Langop SDK gem DSL classes.
# All actual DSL logic lives in the langop gem.
module Based
  module Dsl
    # Alias SDK classes for backwards compatibility
    ToolDefinition = ::Langop::Dsl::ToolDefinition
    ParameterDefinition = ::Langop::Dsl::ParameterDefinition
    Registry = ::Langop::Dsl::Registry
    Config = ::Langop::Dsl::Config
    Helpers = ::Langop::Dsl::Helpers
    HTTP = ::Langop::Dsl::HTTP
    Shell = ::Langop::Dsl::Shell
    Adapter = ::Langop::Dsl::Adapter
    Context = ::Langop::Dsl::Context
    ExecutionContext = ::Langop::Dsl::ExecutionContext
  end
end

# Alias ToolLoader at top level for backwards compatibility
ToolLoader = ::Langop::ToolLoader
