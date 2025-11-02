# Requirement: Pessimistic Version Constraints in Gemfile

**Status**: REQUIRED
**Applies to**: All Gemfile files in Ruby projects
**RFC 2119**: MUST
**Check**: `grep -E "gem '[^']+'" */Gemfile | grep -v '~>'` should return nothing

## Description

All gem dependencies in Gemfile MUST use pessimistic version constraint operator (`~>`). This ensures dependencies stay within compatible minor versions while allowing patch updates.

## Example Implementation

```ruby
source 'https://rubygems.org'

# Production dependencies - use pessimistic versioning
gem 'sinatra', '~> 4.0'
gem 'json', '~> 2.7'
gem 'puma', '~> 6.4'

group :development do
  gem 'rubocop', '~> 1.60'
  gem 'rubocop-performance', '~> 1.20'
  gem 'yard', '~> 0.9.37'
end
```

## Anti-patterns

```ruby
# WRONG - No version constraint
gem 'sinatra'

# WRONG - Exact version (too strict)
gem 'sinatra', '4.0.1'

# WRONG - >= constraint (too permissive)
gem 'sinatra', '>= 4.0'
```

## Compliance

To check compliance:
```bash
# Find all Gemfiles
find . -name Gemfile

# Check each for gems without ~>
grep -E "gem '[^']+'" path/to/Gemfile | grep -v '~>'
```

## Rationale

- Prevents accidental major version upgrades that break compatibility
- Allows automatic patch updates for security fixes
- Follows Ruby community best practices
- Ensures reproducible builds across environments

## Exceptions

NixOS-incompatible gems MAY be documented with comments explaining compatibility issues, but MUST still use pessimistic versioning.

---

**Note**: When implementing this requirement, commit changes with a one-line summary.
