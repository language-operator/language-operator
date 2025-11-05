# frozen_string_literal: true

require "aictl/dsl'
require "aictl/tool_loader'

# Convenience - export Aictl classes at top level for tool definitions
#
# This allows tool files to use simplified syntax:
#   tool "example" do
#     ...
#   end
#
# Instead of:
#   Aictl::Dsl.define do
#     tool "example" do
#       ...
#     end
#   end

# Alias ToolLoader at top level for convenience
ToolLoader = ::Aictl::ToolLoader
