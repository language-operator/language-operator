# Requirement: Makefile Help Target

**Status**: REQUIRED
**Applies to**: All Makefiles in repository
**RFC 2119**: MUST
**Check**: Run `make` (or `make help`) in each directory with a Makefile

## Description

For any Makefile in this repo, running `make` without a target MUST display documentation for each target. The help target MUST be the default target (first in file or `.DEFAULT_GOAL`).

## Example Implementation

```
help:
	@echo "Language Operator"
	@echo ""
	@echo "Build & Management:"
	@echo "  build          - Build all Docker images"
	@echo "  operator       - Build and deploy the language operator"
	@echo "  docs           - Generate CRD API reference documentation"
	@echo ""
	@echo "Kubernetes Operations:"
	@echo "  k8s-install    - Install the language operator to Kubernetes"
	@echo "  k8s-uninstall  - Uninstall the language operator from Kubernetes"
	@echo "  k8s-status     - Check status of all language resources"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. Ensure you have a Kubernetes cluster (kind, minikube, etc.)"
	@echo "  2. Run 'make operator' to build and deploy the operator"
	@echo "  3. Apply LanguageCluster, LanguageAgent, and LanguageTool CRDs"
	@echo "  4. Check status with 'make k8s-status'"
```

## Compliance

To check compliance:
```bash
# Find all Makefiles
find . -name Makefile -o -name "*.mk"

# Test each one
make -C <directory>  # Should show help output
```

## Rationale

- Improves discoverability of Makefile targets
- Consistent UX across all project Makefiles
- Self-documenting build system
- Reduces time-to-productivity for new contributors

---

**Note**: When implementing this requirement, commit changes with a one-line summary.