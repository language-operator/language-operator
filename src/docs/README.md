# language-operator API Documentation

This directory contains auto-generated API reference documentation for the language-operator Custom Resource Definitions (CRDs).

## Files

- **[api-reference.md](api-reference.md)** - Complete API reference for all CRDs

## Generating Documentation

To regenerate the API documentation after making changes to the CRD types:

```bash
# From the src directory
make docs

# Or from the repository root
make docs
```

This will use [crd-ref-docs](https://github.com/elastic/crd-ref-docs) to scan the Go types in `api/v1alpha1` and generate markdown documentation.

## Available CRDs

The documentation covers the following Custom Resource Definitions:

- **LanguageCluster** - Network-isolated environment for agents, tools, and models
- **LanguageAgent** - Goal-oriented autonomous agents
- **LanguageTool** - MCP-compatible tool servers (web search, email, etc.)
- **LanguageModel** - Model configurations with LiteLLM proxy
- **LanguagePersona** - Reusable personality/instruction templates

## Configuration

The documentation generation is configured via `.crd-ref-docs.yaml`:

- Excludes standard Kubernetes metadata types (TypeMeta, ObjectMeta, etc.)
- Links to Kubernetes 1.28 API documentation
- Groups output by API kind

## Contributing

When adding new fields to CRD types:

1. Add proper godoc comments describing the field
2. Include kubebuilder validation markers (`+kubebuilder:validation:...`)
3. Regenerate docs with `make docs`
4. Commit both the code changes and updated documentation
