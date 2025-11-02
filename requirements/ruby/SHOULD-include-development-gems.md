# Requirement: Standard Development Gems

**Status**: RECOMMENDED
**Applies to**: All Ruby projects with Gemfile
**RFC 2119**: SHOULD
**Check**: Verify `rubocop`, `rubocop-performance`, and `yard` in development group

## Description

Ruby projects SHOULD include standard development gems for code quality and documentation. These tools ensure consistent style and comprehensive API documentation across the project.

## Example Implementation

```ruby
source 'https://rubygems.org'

# Production dependencies
gem 'sinatra', '~> 4.0'

group :development do
  gem 'rubocop', '~> 1.60'
  gem 'rubocop-performance', '~> 1.20'
  gem 'yard', '~> 0.9.37'
end
```

## Required Development Gems

### Minimum Set (SHOULD include)
- **rubocop** - Ruby style checker and formatter
- **rubocop-performance** - Performance-focused RuboCop rules
- **yard** - Documentation generator

### Optional (MAY include)
- **rspec** - Testing framework
- **simplecov** - Code coverage
- **pry** - Debugging REPL

## Anti-patterns

```ruby
# WRONG - Including rdoc (fails on NixOS)
group :development do
  gem 'rdoc'  # Don't use this
end

# WRONG - Production dependencies in development group
group :development do
  gem 'sinatra'  # Should be in production
end
```

## Compliance

To check compliance:
```bash
# Check if required gems are present
grep -E "(rubocop|yard)" Gemfile

# Verify they're in development group
awk '/group :development/,/^end/' Gemfile | grep -E "(rubocop|yard)"
```

## Rationale

- **rubocop**: Enforces consistent code style across contributors
- **rubocop-performance**: Catches performance anti-patterns early
- **yard**: Generates browsable API documentation

## Exceptions

- Minimal production images MAY omit development gems
- Libraries intended for inclusion in other projects MUST include these gems for maintainability

---

**Note**: When implementing this requirement, commit changes with a one-line summary.
