# Requirements Index

This directory contains project requirements and conventions. Each requirement follows RFC 2119 key words (MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, OPTIONAL).

## Active Requirements

### Makefile Requirements
- [MUST-have-help-target](makefile/MUST-have-help-target.md) - All Makefiles must provide help documentation
- [MUST-include-docker-targets](makefile/MUST-include-docker-targets.md) - Makefiles with Dockerfile must include build, scan, shell, run, publish targets

### Ruby Requirements
- [MUST-use-pessimistic-versioning](ruby/MUST-use-pessimistic-versioning.md) - All gems must use ~> version constraints
- [MUST-document-public-apis](ruby/MUST-document-public-apis.md) - Public methods must have YARD documentation
- [SHOULD-include-development-gems](ruby/SHOULD-include-development-gems.md) - Include rubocop, yard in development group

## Checking Compliance

Each requirement file includes a "Compliance" section with instructions for checking adherence. To run a full audit:

```bash
# TODO: Add automated compliance checker script
./scripts/check-requirements.sh
```

## Adding New Requirements

1. Create a new file in the appropriate category directory
2. Use naming: `{RFC-2119-KEYWORD}-{short-description}.md`
   - Example: `MUST-use-semantic-commits.md`
   - Example: `SHOULD-include-tests.md`
3. Include metadata: Status, Applies to, RFC 2119, Check
4. Add to this index
5. Commit with one-line summary

## RFC 2119 Keywords

- **MUST** / **REQUIRED** / **SHALL** - Absolute requirement
- **MUST NOT** / **SHALL NOT** - Absolute prohibition
- **SHOULD** / **RECOMMENDED** - Strong suggestion, may ignore with valid reason
- **SHOULD NOT** / **NOT RECOMMENDED** - Strong discouragement, may do with valid reason
- **MAY** / **OPTIONAL** - Truly optional, implementer discretion
