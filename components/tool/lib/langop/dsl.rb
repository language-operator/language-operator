# frozen_string_literal: true

require 'langop/dsl'
require 'langop/tool_loader'

# Convenience - export Langop classes at top level for tool definitions
#
# This allows tool files to use simplified syntax:
#   tool "example" do
#     ...
#   end
#
# Instead of:
#   Langop::Dsl.define do
#     tool "example" do
#       ...
#     end
#   end

# Alias ToolLoader at top level for convenience
ToolLoader = ::Langop::ToolLoader
