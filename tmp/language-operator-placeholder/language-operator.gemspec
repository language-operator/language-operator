# frozen_string_literal: true

require_relative "lib/language_operator/version"

Gem::Specification.new do |spec|
  spec.name          = "language-operator"
  spec.version       = LanguageOperator::VERSION
  spec.authors       = ["James Ryan"]
  spec.email         = ["james@theryans.io"]

  spec.summary       = "Placeholder - Language Operator for Kubernetes"
  spec.description   = <<~DESC
    This gem name is reserved for the Language Operator project.

    Language Operator is a Kubernetes operator for deploying and managing
    language model agents, tools, and models in Kubernetes clusters.

    For more information, visit the project homepage.
  DESC

  spec.homepage      = "https://github.com/language-operator"
  spec.license       = "MIT"
  spec.required_ruby_version = ">= 3.0.0"

  spec.metadata = {
    "homepage_uri"      => spec.homepage,
    "source_code_uri"   => spec.homepage,
    "bug_tracker_uri"   => spec.homepage,
    "documentation_uri" => spec.homepage
  }

  spec.files = [
    "README.md",
    "LICENSE",
    "lib/language_operator.rb",
    "lib/language_operator/version.rb"
  ]

  spec.require_paths = ["lib"]
end
