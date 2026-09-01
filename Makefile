GIT_SHA   := $(shell git rev-parse --short HEAD)
DEV_IMAGE := language-operator:$(GIT_SHA)

.PHONY: help build test dev setup-hooks install upgrade uninstall wipe k8s-status agent-supervisor docs-serve docs-build

# Build the operator binary
build:
	@cd src && $(MAKE) build

# Set up git hooks for development
setup-hooks:
	@./scripts/setup-hooks

# Run Go tests
test:
	@cd src && $(MAKE) test

# Build, load into k3s, and upgrade the release (development inner loop)
dev:
	@cd src && $(MAKE) build
	docker build -t $(DEV_IMAGE) .
	docker save $(DEV_IMAGE) | sudo k3s ctr images import -
	helm upgrade --install language-operator charts/language-operator \
		--namespace language-operator \
		--create-namespace \
		--values charts/language-operator/values.local.yaml \
		--set image.repository=docker.io/library/language-operator \
		--set-string image.tag=$(GIT_SHA) \
		--set image.pullPolicy=Never \
		--wait --timeout 2m
	helm dependency build charts/language-operator-runtimes
	helm upgrade --install language-operator-runtimes charts/language-operator-runtimes \
		--namespace language-operator \
		--values charts/language-operator-runtimes/values.local.yaml \
		--wait --timeout 2m
	kubectl rollout restart deployment language-operator -n language-operator
	kubectl rollout status deployment language-operator -n language-operator --timeout=2m

# Install the Helm chart using charts/language-operator/values.local.yaml
install:
	@cd charts/language-operator && $(MAKE) install

# Upgrade the Helm release using charts/language-operator/values.local.yaml
upgrade:
	@cd charts/language-operator && $(MAKE) upgrade

# Uninstall the Helm release
uninstall:
	@cd charts/language-operator && $(MAKE) uninstall

# Install the runtimes chart (requires language-operator chart with CRDs installed first)
install-runtimes:
	helm dependency build charts/language-operator-runtimes
	helm upgrade --install language-operator-runtimes charts/language-operator-runtimes \
		--namespace language-operator \
		--create-namespace \
		--wait --timeout 2m

# Upgrade the runtimes release
upgrade-runtimes:
	helm dependency build charts/language-operator-runtimes
	helm upgrade language-operator-runtimes charts/language-operator-runtimes \
		--namespace language-operator \
		--wait --timeout 2m

# Uninstall the runtimes release
uninstall-runtimes:
	helm uninstall language-operator-runtimes --namespace language-operator

# Custom resource kinds, in teardown order: dependents before the cluster that owns them.
LANGOP_KINDS := languageagentselfconfigs languageagents languagetools languagemodels languagepersonas languageclusters languageagentruntimes

# Wipe everything `make dev` installs — CRs, both Helm releases, CRDs (ours and
# Argo's), the namespace, orphaned cluster-scoped resources, and the dev images
# imported into k3s. Use this to get back to a clean cluster state.
#
# This must work whether or not the operator is running, which drives the order:
#
#   1. Webhook configurations go first. They have failurePolicy: Fail and point at
#      a Service in the operator namespace, so once the operator is gone they
#      reject every write to a langop.io resource — including the patch that
#      removes a finalizer. Leaving them up deadlocks the rest of this target.
#   2. Deletes are issued non-blocking, then waited on with a bounded timeout.
#      A running operator gets its chance to finalize gracefully (agents tear down
#      their Workflows, a LanguageCluster deletes the namespace it created), but a
#      dead operator can never finalize and an unbounded --wait would hang forever.
#   3. Whatever is still terminating after that has its finalizers stripped.
wipe:
	@echo "Removing admission webhooks (they block writes once the operator is gone)..."
	@kubectl delete validatingwebhookconfiguration language-operator-validating-webhook --ignore-not-found 2>/dev/null || true
	@kubectl delete mutatingwebhookconfiguration language-operator-mutating-webhook --ignore-not-found 2>/dev/null || true
	@echo "Deleting all language operator custom resources..."
	@for kind in $(LANGOP_KINDS); do \
		kubectl delete $$kind --all -A --ignore-not-found --wait=false 2>/dev/null || true; \
	done
	@echo "Waiting up to 60s for graceful finalization..."
	@for kind in $(LANGOP_KINDS); do \
		kubectl wait --for=delete $$kind --all -A --timeout=60s >/dev/null 2>&1 || true; \
	done
	@echo "Stripping finalizers from anything still terminating..."
	@for kind in $(LANGOP_KINDS); do \
		kubectl get $$kind -A -o jsonpath='{range .items[*]}{.metadata.namespace}{"|"}{.metadata.name}{"\n"}{end}' 2>/dev/null \
		| while IFS='|' read -r ns name; do \
			[ -n "$$name" ] || continue; \
			if [ -n "$$ns" ]; then \
				kubectl patch $$kind "$$name" -n "$$ns" --type=merge -p '{"metadata":{"finalizers":null}}' >/dev/null 2>&1 || true; \
			else \
				kubectl patch $$kind "$$name" --type=merge -p '{"metadata":{"finalizers":null}}' >/dev/null 2>&1 || true; \
			fi; \
		done; \
	done
	@echo "Uninstalling Helm releases..."
	@helm uninstall language-operator-runtimes --namespace language-operator --ignore-not-found 2>/dev/null || true
	@helm uninstall language-operator --namespace language-operator --ignore-not-found 2>/dev/null || true
	@echo "Deleting CRDs..."
	@kubectl get crds -o name 2>/dev/null | grep langop.io | xargs -r kubectl delete --ignore-not-found
	@# Argo's CRDs are applied by a pre-install hook Job, so they are not part of
	@# the Helm release and survive `helm uninstall`. Deleting them also cascades
	@# away any Workflow objects left behind in cluster namespaces.
	@echo "Deleting Argo Workflows CRDs..."
	@kubectl get crds -o name 2>/dev/null | grep argoproj.io | xargs -r kubectl delete --ignore-not-found
	@# Defensive: the hook's cluster-scoped RBAC self-deletes on success, but a
	@# hook that failed part way through leaves it behind.
	@kubectl delete clusterrole,clusterrolebinding -l app.kubernetes.io/name=argo-workflows-crd-install --ignore-not-found 2>/dev/null || true
	@# Cluster-scoped chart resources are orphaned if the namespace is ever deleted
	@# before `helm uninstall` runs — the release secret lives in that namespace, so
	@# losing it strands every ClusterRole, ClusterRoleBinding, and webhook config.
	@echo "Sweeping orphaned cluster-scoped resources..."
	@kubectl get clusterrole,clusterrolebinding,validatingwebhookconfiguration,mutatingwebhookconfiguration -o json 2>/dev/null \
		| jq -r '.items[] | select(.metadata.annotations."meta.helm.sh/release-name" == "language-operator" or .metadata.annotations."meta.helm.sh/release-name" == "language-operator-runtimes") | "\(.kind|ascii_downcase)/\(.metadata.name)"' 2>/dev/null \
		| xargs -r kubectl delete --ignore-not-found 2>/dev/null || true
	@echo "Deleting namespace..."
	@kubectl delete namespace language-operator --ignore-not-found 2>/dev/null || true
	@echo "Removing dev images from k3s..."
	@sudo k3s ctr images ls -q 2>/dev/null | grep '^docker.io/library/language-operator:' | xargs -r sudo k3s ctr images rm >/dev/null 2>&1 || true
	@echo "Done."

# Check Kubernetes resources status
k8s-status:
	@echo "Language Operator Resources:"
	@kubectl get languageclusters,languageagents,languagetools,languagemodels -A
	@echo ""
	@echo "Operator Status:"
	@kubectl get pods -n language-operator

dev-supervisor:
	claude "/delegate"

dev-worker-%:
	claude "/watch $*"

# Preview the documentation site locally (http://localhost:8000)
docs-serve:
	@uv run mkdocs serve

# Build the documentation site (strict — fails on broken links/nav)
docs-build:
	@uv run mkdocs build --strict

# Show help
help:
	@echo "Targets:"
	@echo "  build        - Build operator binary"
	@echo "  test         - Run Go test suite"
	@echo "  dev          - Build, load into k3s, and upgrade (inner loop)"
	@echo "  setup-hooks  - Install git pre-commit hooks"
	@echo "  install      - Install Helm chart (charts/language-operator/values.local.yaml)"
	@echo "  upgrade      - Upgrade Helm release"
	@echo "  uninstall    - Uninstall Helm release"
	@echo "  wipe         - Remove everything 'make dev' installs: CRs, both Helm releases,"
	@echo "                 langop.io + argoproj.io CRDs, namespace, and k3s dev images"
	@echo "  k8s-status   - Check status of all language resources"
	@echo "  docs-serve   - Preview the docs site locally (uv run mkdocs serve)"
	@echo "  docs-build   - Build the docs site strictly (uv run mkdocs build)"
	@echo "  dev-supervisor   - Run the supervisor agent (triage issues into queues)"
	@echo "  dev-worker-N     - Run worker agent for queue N (0, 1, or 2)"
