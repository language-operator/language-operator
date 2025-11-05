# frozen_string_literal: true

require_relative 'lib/aictl/version'

Gem::Specification.new do |spec|
  spec.name          = 'aictl'
  spec.version       = Aictl::VERSION
  spec.authors       = ['Language Operator Team']
  spec.email         = ['noreply@langop.io']

  spec.summary       = 'Beautiful CLI for language-operator - create autonomous AI agents with natural language'
  spec.description   = 'aictl is the command-line interface for language-operator, providing a beautiful natural language experience for creating and managing autonomous AI agents on Kubernetes'
  spec.homepage      = 'https://github.com/langop/language-operator'
  spec.license       = 'MIT'
  spec.required_ruby_version = '>= 3.0.0'

  spec.metadata['homepage_uri'] = spec.homepage
  spec.metadata['source_code_uri'] = 'https://github.com/langop/language-operator'
  spec.metadata['changelog_uri'] = 'https://github.com/langop/language-operator/blob/main/sdk/ruby/CHANGELOG.md'

  # Specify which files should be added to the gem when it is released.
  spec.files = Dir.chdir(File.expand_path(__dir__)) do
    `git ls-files -z`.split("\x0").reject do |f|
      (f == __FILE__) || f.match(%r{\A(?:(?:bin|test|spec|features)/|\.(?:git|travis|circleci)|appveyor)})
    end
  end
  spec.bindir        = 'bin'
  spec.executables   = ['aictl']
  spec.require_paths = ['lib']

  # Runtime dependencies
  spec.add_dependency 'mcp', '~> 0.4'
  spec.add_dependency 'ruby_llm', '~> 1.8'
  spec.add_dependency 'ruby_llm-mcp', '~> 0.1'
  spec.add_dependency 'thor', '~> 1.3'

  # Development dependencies
  spec.add_development_dependency 'bundler', '~> 2.0'
  spec.add_development_dependency 'rake', '~> 13.0'
  spec.add_development_dependency 'rspec', '~> 3.0'
  spec.add_development_dependency 'rubocop', '~> 1.60'
  spec.add_development_dependency 'rubocop-performance', '~> 1.20'
  spec.add_development_dependency 'webmock', '~> 3.23'
  spec.add_development_dependency 'yard', '~> 0.9.37'
end
