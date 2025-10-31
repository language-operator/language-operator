.PHONY: help install upgrade uninstall dry-run status logs restart check-health validate

# Cilium Helm chart configuration
CILIUM_VERSION ?= 1.16.5
CILIUM_NAMESPACE ?= kube-system
HELM_RELEASE_NAME ?= cilium
HELM_REPO ?= cilium
HELM_REPO_URL ?= https://helm.cilium.io/
VALUES_FILE ?= values.yaml

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "Cilium Management Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Installation

install: add-repo ## Install Cilium with current values.yaml
	@echo "$(GREEN)Installing Cilium $(CILIUM_VERSION)...$(NC)"
	helm install $(HELM_RELEASE_NAME) $(HELM_REPO)/cilium \
		--namespace $(CILIUM_NAMESPACE) \
		--version $(CILIUM_VERSION) \
		--values $(VALUES_FILE) \
		--wait
	@echo "$(GREEN)✓ Cilium installed successfully$(NC)"
	@$(MAKE) status

upgrade: add-repo ## Upgrade Cilium with current values.yaml
	@echo "$(YELLOW)Upgrading Cilium to $(CILIUM_VERSION)...$(NC)"
	helm upgrade $(HELM_RELEASE_NAME) $(HELM_REPO)/cilium \
		--namespace $(CILIUM_NAMESPACE) \
		--version $(CILIUM_VERSION) \
		--values $(VALUES_FILE) \
		--reuse-values \
		--wait
	@echo "$(GREEN)✓ Cilium upgraded successfully$(NC)"
	@$(MAKE) status

uninstall: ## Uninstall Cilium
	@echo "$(RED)Uninstalling Cilium...$(NC)"
	@read -p "Are you sure? This will remove Cilium CNI [y/N]: " confirm && \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		helm uninstall $(HELM_RELEASE_NAME) --namespace $(CILIUM_NAMESPACE) || true; \
		echo "$(GREEN)✓ Cilium uninstalled$(NC)"; \
	else \
		echo "Cancelled."; \
	fi

##@ Testing & Validation

dry-run: add-repo ## Dry-run installation to see what would be applied
	@echo "$(YELLOW)Running dry-run for Cilium $(CILIUM_VERSION)...$(NC)"
	helm install $(HELM_RELEASE_NAME) $(HELM_REPO)/cilium \
		--namespace $(CILIUM_NAMESPACE) \
		--version $(CILIUM_VERSION) \
		--values $(VALUES_FILE) \
		--dry-run \
		--debug

template: add-repo ## Generate Kubernetes manifests from Helm chart
	@echo "$(YELLOW)Generating manifests...$(NC)"
	helm template $(HELM_RELEASE_NAME) $(HELM_REPO)/cilium \
		--namespace $(CILIUM_NAMESPACE) \
		--version $(CILIUM_VERSION) \
		--values $(VALUES_FILE)

validate: add-repo ## Validate values.yaml syntax
	@echo "$(YELLOW)Validating values.yaml...$(NC)"
	@helm template $(HELM_RELEASE_NAME) $(HELM_REPO)/cilium \
		--namespace $(CILIUM_NAMESPACE) \
		--version $(CILIUM_VERSION) \
		--values $(VALUES_FILE) \
		--validate > /dev/null 2>&1 && \
		echo "$(GREEN)✓ values.yaml is valid$(NC)" || \
		(echo "$(RED)✗ Validation failed$(NC)" && exit 1)

diff: add-repo ## Show diff between current and new configuration
	@echo "$(YELLOW)Showing configuration diff...$(NC)"
	@helm diff upgrade $(HELM_RELEASE_NAME) $(HELM_REPO)/cilium \
		--namespace $(CILIUM_NAMESPACE) \
		--version $(CILIUM_VERSION) \
		--values $(VALUES_FILE) || \
		echo "$(YELLOW)Note: helm-diff plugin not installed. Install with: helm plugin install https://github.com/databus23/helm-diff$(NC)"

##@ Status & Monitoring

status: ## Show Cilium installation status
	@echo "$(GREEN)Cilium Status:$(NC)"
	@echo ""
	@echo "Helm Release:"
	@helm list -n $(CILIUM_NAMESPACE) | grep $(HELM_RELEASE_NAME) || echo "Not installed"
	@echo ""
	@echo "Cilium Pods:"
	@kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-agent
	@echo ""
	@echo "Cilium Operator:"
	@kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-operator

check-health: ## Check Cilium health status
	@echo "$(YELLOW)Checking Cilium health...$(NC)"
	@kubectl exec -n $(CILIUM_NAMESPACE) ds/cilium -- cilium-health status || \
		echo "$(RED)Health check failed. Is Cilium running?$(NC)"

connectivity-test: ## Run Cilium connectivity test
	@echo "$(YELLOW)Running Cilium connectivity test...$(NC)"
	kubectl apply -f https://raw.githubusercontent.com/cilium/cilium/v$(CILIUM_VERSION)/examples/kubernetes/connectivity-check/connectivity-check.yaml
	@echo ""
	@echo "$(YELLOW)Wait for test pods to complete, then check:$(NC)"
	@echo "  kubectl get pods -n cilium-test"

connectivity-cleanup: ## Clean up connectivity test
	kubectl delete ns cilium-test --ignore-not-found

logs: ## Show Cilium agent logs (from first pod)
	@echo "$(YELLOW)Cilium Agent Logs:$(NC)"
	@POD=$$(kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-agent -o jsonpath='{.items[0].metadata.name}'); \
	kubectl logs -n $(CILIUM_NAMESPACE) $$POD --tail=100 -f

operator-logs: ## Show Cilium operator logs
	@echo "$(YELLOW)Cilium Operator Logs:$(NC)"
	@POD=$$(kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-operator -o jsonpath='{.items[0].metadata.name}'); \
	kubectl logs -n $(CILIUM_NAMESPACE) $$POD --tail=100 -f

##@ Maintenance

restart: ## Restart Cilium pods
	@echo "$(YELLOW)Restarting Cilium...$(NC)"
	kubectl rollout restart ds/cilium -n $(CILIUM_NAMESPACE)
	kubectl rollout restart deployment/cilium-operator -n $(CILIUM_NAMESPACE)
	@echo "$(GREEN)✓ Cilium restarted$(NC)"

watch: ## Watch Cilium pods status
	watch kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/part-of=cilium

config: ## Show current Cilium configuration
	@echo "$(YELLOW)Current Cilium ConfigMap:$(NC)"
	kubectl get configmap -n $(CILIUM_NAMESPACE) cilium-config -o yaml

version: ## Show installed and available Cilium versions
	@echo "$(GREEN)Installed Version:$(NC)"
	@helm list -n $(CILIUM_NAMESPACE) -o json | jq -r '.[] | select(.name=="$(HELM_RELEASE_NAME)") | .app_version' || echo "Not installed"
	@echo ""
	@echo "$(GREEN)Target Version:$(NC)"
	@echo "$(CILIUM_VERSION)"
	@echo ""
	@echo "$(GREEN)Available Versions:$(NC)"
	@helm search repo $(HELM_REPO)/cilium --versions | head -10

##@ Debugging

debug-endpoints: ## Show Cilium endpoints
	@POD=$$(kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-agent -o jsonpath='{.items[0].metadata.name}'); \
	kubectl exec -n $(CILIUM_NAMESPACE) $$POD -- cilium endpoint list

debug-policy: ## Show Cilium network policies
	@POD=$$(kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-agent -o jsonpath='{.items[0].metadata.name}'); \
	kubectl exec -n $(CILIUM_NAMESPACE) $$POD -- cilium policy get

debug-service-list: ## Show Cilium service list
	@POD=$$(kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-agent -o jsonpath='{.items[0].metadata.name}'); \
	kubectl exec -n $(CILIUM_NAMESPACE) $$POD -- cilium service list

debug-status: ## Show detailed Cilium status
	@POD=$$(kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-agent -o jsonpath='{.items[0].metadata.name}'); \
	kubectl exec -n $(CILIUM_NAMESPACE) $$POD -- cilium status --verbose

debug-bpf: ## Show BPF maps
	@POD=$$(kubectl get pods -n $(CILIUM_NAMESPACE) -l app.kubernetes.io/name=cilium-agent -o jsonpath='{.items[0].metadata.name}'); \
	kubectl exec -n $(CILIUM_NAMESPACE) $$POD -- cilium bpf lb list

##@ Helpers

add-repo: ## Add/update Cilium Helm repository
	@helm repo add $(HELM_REPO) $(HELM_REPO_URL) >/dev/null 2>&1 || true
	@helm repo update $(HELM_REPO) >/dev/null 2>&1

backup: ## Backup current values.yaml
	@echo "$(YELLOW)Backing up values.yaml...$(NC)"
	@cp $(VALUES_FILE) $(VALUES_FILE).backup.$$(date +%Y%m%d-%H%M%S)
	@echo "$(GREEN)✓ Backup created$(NC)"

restore-backup: ## Restore values.yaml from latest backup
	@LATEST=$$(ls -t $(VALUES_FILE).backup.* 2>/dev/null | head -1); \
	if [ -z "$$LATEST" ]; then \
		echo "$(RED)No backups found$(NC)"; \
		exit 1; \
	fi; \
	echo "$(YELLOW)Restoring from $$LATEST...$(NC)"; \
	cp "$$LATEST" $(VALUES_FILE); \
	echo "$(GREEN)✓ Restored$(NC)"

list-backups: ## List available backups
	@echo "$(GREEN)Available backups:$(NC)"
	@ls -lh $(VALUES_FILE).backup.* 2>/dev/null || echo "No backups found"

clean-backups: ## Remove old backups (keep last 5)
	@echo "$(YELLOW)Cleaning old backups...$(NC)"
	@ls -t $(VALUES_FILE).backup.* 2>/dev/null | tail -n +6 | xargs rm -f 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(NC)"
