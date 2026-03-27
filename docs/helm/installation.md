# Helm Installation

Install Language Operator using Helm.

## Quick Install

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator
helm repo update
helm install language-operator language-operator/language-operator \
  --create-namespace \
  --namespace language-operator-system
```

For detailed installation instructions, see the [Getting Started Guide](../getting-started/installation.md).

## Chart Repository

The Helm chart is hosted on GitHub Pages at:

```
https://language-operator.github.io/language-operator
```

Browse available versions:

```bash
helm search repo language-operator --versions
```

## Values

See [Configuration](configuration.md) for all available Helm values.

## Development

To test the chart locally:

```bash
# From repository root
cd chart
helm lint .
helm template language-operator . --debug
```
