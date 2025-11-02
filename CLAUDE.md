When committing code, use semantic messages like "feat: " with a brief one-line summary, not a multiline essay.

# Project Defaults

## Registry
- Private registry: git.theryans.io
- Registry namespace: langop
- Example image: git.theryans.io/langop/tool:latest

## Infrastructure
- Kubernetes cluster: k3s on 5 nodes (dl1-dl5)
- SSH access: james@dl1, james@dl2, james@dl3, james@dl4, james@dl5
- Master node: dl1 (runs k3s service)
- Agent nodes: dl2-dl5 (run k3s service, not k3s-agent)

## Operator
- Operator namespace: kube-system
- Operator deployment: language-operator
- CRDs: LanguageCluster, LanguageModel, LanguageTool, LanguageAgent

## Testing
- Never commit code that hasn't been tested
- Verify scripts must work end-to-end before committing
- Registry authentication must be verified before testing image pulls