# language-operator

Custom resource definitions for language-based automations.


## Quick Start


```

## Architecture

Based is deployed as a Kubernetes-native platform using custom resource definitions (CRDs):

- **LanguageCluster**: Defines a cluster of language agents with shared network policies
- **LanguageAgent**: Represents a language model agent with specific capabilities
- **LanguageClient**: Client applications that interact with language agents
- **LanguageTool**: Deployable tools that agents can invoke (email, SMS, web search, etc.)

All components use the MCP (Model Context Protocol) JSON-RPC 2.0 format for communication.

### Example Tool Call

```bash
# Call a tool service via the Kubernetes service endpoint
kubectl run curl-test --rm -it --restart=Never --image=curlimages/curl -- \
  curl -X POST http://web-tool.default.svc.cluster.local/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "web_status",
      "arguments": {"url": "https://example.com"}
    }
  }'
```

## Configuration

Configuration is managed through Kubernetes ConfigMaps and Secrets referenced in the CRD specs. See the [examples directory](kubernetes/language-operator/examples/) for sample configurations.

## Development Workflow

1. Ensure you have a Kubernetes cluster (kind, minikube, or other):
   ```bash
   kind create cluster --name based
   ```

2. Build and deploy the language operator:
   ```bash
   make operator
   ```

3. Check the operator status:
   ```bash
   make k8s-status
   ```

4. Apply your LanguageCluster, LanguageAgent, and LanguageTool resources:
   ```bash
   kubectl apply -f kubernetes/language-operator/examples/
   ```

5. Make changes to the operator code and redeploy:
   ```bash
   make operator
   ```

## Available Make Targets

Run `make help` to see all available targets:

- `make build` - Build all Docker images
- `make operator` - Build and deploy the language operator
- `make k8s-install` - Install the language operator to Kubernetes
- `make k8s-uninstall` - Uninstall the language operator from Kubernetes
- `make k8s-status` - Check status of all language resources

## Project Structure

```
based/
├── components/             # Container components for agents and tools
│   ├── server/            # MCP server framework
│   ├── client/            # MCP client library
│   └── ...                # Other components
├── kubernetes/
│   ├── language-operator/ # Kubernetes operator for managing language agents
│   │   ├── api/           # CRD definitions
│   │   ├── controllers/   # Reconciliation logic
│   │   └── config/        # Operator manifests
│   ├── charts/            # Helm charts for deployment
│   │   └── language-operator/
│   └── utilities/         # Helper scripts and tools
├── scripts/
│   └── build              # Build all images script
└── Makefile               # Build and deployment commands
```