# Requirement: YARD Documentation for Public APIs

**Status**: REQUIRED
**Applies to**: All Ruby library code (lib/ directories)
**RFC 2119**: MUST
**Check**: Run `yard stats --list-undoc` and verify 0 undocumented public methods

## Description

All public methods, classes, and modules MUST have YARD documentation including:
- Method description
- `@param` tags for all parameters
- `@return` tag describing return value
- `@example` tag showing usage
- `@raise` tag for exceptions (if applicable)

## Example Implementation

### Module Documentation
```ruby
# MCP server framework for building language tools
#
# @example Creating a simple tool
#   class MyTool
#     include Langop::Tool::DSL
#
#     tool "weather" do
#       description "Get weather information"
#       param :city, type: String, required: true
#       execute { |args| fetch_weather(args[:city]) }
#     end
#   end
module Langop
  module Tool
    # Tool definition DSL
    module DSL
    end
  end
end
```

### Method Documentation
```ruby
# Fetch weather data for a given city
#
# @param city [String] The city name to get weather for
# @return [Hash] Weather data including temperature and conditions
# @raise [ArgumentError] if city is nil or empty
# @raise [APIError] if weather service is unavailable
# @example
#   weather = fetch_weather('San Francisco')
#   puts weather[:temperature]
def fetch_weather(city)
  raise ArgumentError, 'City cannot be empty' if city.to_s.strip.empty?

  # Implementation
end
```

## .yardopts Configuration

Projects MUST include a `.yardopts` file:

```
--markup markdown
--readme README.md
--output-dir doc
--protected
--private
lib/**/*.rb
```

## Compliance

To check compliance:
```bash
# Install YARD
bundle exec yard gems

# Generate documentation and check coverage
bundle exec yard stats --list-undoc

# Should show 100% for public APIs
# Example acceptable output:
# 100.00% documented
```

## Rationale

- Self-documenting code improves maintainability
- Examples in docs prevent misuse
- YARD integrates with IDE tooling for inline help
- Generated documentation serves as API reference
- Catches parameter mismatches during doc generation

## Coverage Targets

- **Public APIs**: 100% MUST be documented
- **Protected methods**: SHOULD be documented
- **Private methods**: MAY be documented

---

**Note**: When implementing this requirement, commit changes with a one-line summary.
