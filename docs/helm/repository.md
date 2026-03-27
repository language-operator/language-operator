# Helm Repository

The Language Operator Helm repository is hosted on GitHub Pages.

## Repository URL

```
https://language-operator.github.io/language-operator
```

## Adding the Repository

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator
helm repo update
```

## Searching for Charts

List all available charts:

```bash
helm search repo language-operator
```

List all versions:

```bash
helm search repo language-operator --versions
```

## Chart Versions

The repository follows semantic versioning. Charts are automatically built and published from git tags matching `v*`.

## Development Builds

Development builds from the `main` branch are also published with the `latest` tag.

## Manual Repository Access

You can also browse the Helm repository directly:

- **Index:** [index.yaml](https://language-operator.github.io/language-operator/index.yaml)
- **Charts:** [Browse on GitHub](https://github.com/language-operator/language-operator/tree/gh-pages)

## Using Specific Versions

Install a specific chart version:

```bash
helm install language-operator language-operator/language-operator \
  --version 1.0.0
```

## Local Chart Development

To test the chart from source:

```bash
# Clone the repository
git clone https://github.com/language-operator/language-operator
cd language-operator/chart

# Install from local directory
helm install language-operator . --namespace language-operator-system
```
