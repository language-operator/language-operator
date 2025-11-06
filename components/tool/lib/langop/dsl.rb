# frozen_string_literal: true

require "language_operator"

# Convenience - export LanguageOperator classes at top level for tool definitions
#
# This allows tool files to use simplified syntax:
#   tool "example" do
#     ...
#   end
#
# Instead of:
#   LanguageOperator::Dsl.define do
#     tool "example" do
#       ...
#     end
#   end

# Alias ToolLoader at top level for convenience
ToolLoader = ::LanguageOperator::ToolLoader
