#!/bin/bash
# Language Operator End-to-End Verification Script
#
# Tests the complete language operator stack by deploying and verifying:
# - LanguageCluster
# - LanguageModel (with LiteLLM proxy)
# - LanguageTool (with sidecar injection)
# - LanguageAgent (with workspace and tool integration)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="demo"
TIMEOUT="${TIMEOUT:-120}" # 2 minutes default
VERBOSE="${VERBOSE:-false}"
FAIL_FAST="${FAIL_FAST:-true}"
CLEANUP_ONLY="${CLEANUP_ONLY:-false}"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --verbose|-v)
      VERBOSE=true
      shift
      ;;
    --no-fail-fast)
      FAIL_FAST=false
      shift
      ;;
    --cleanup-only)
      CLEANUP_ONLY=true
      shift
      ;;
    --timeout)
      TIMEOUT="$2"
      shift 2
      ;;
    --help|-h)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --verbose, -v       Show detailed kubectl output"
      echo "  --no-fail-fast      Continue testing even after failures"
      echo "  --cleanup-only      Only delete resources, don't test"
      echo "  --timeout SECONDS   Timeout for resource readiness (default: 120)"
      echo "  --help, -h          Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Helper functions
log() {
  echo -e "${BLUE}[INFO]${NC} $*"
}

success() {
  echo -e "${GREEN}[]${NC} $*"
}

warning() {
  echo -e "${YELLOW}[� ]${NC} $*"
}

error() {
  echo -e "${RED}[L]${NC} $*"
}

fail() {
  error "$*"
  if [ "$FAIL_FAST" = true ]; then
    exit 1
  fi
  return 1
}

run_kubectl() {
  if [ "$VERBOSE" = true ]; then
    kubectl "$@"
  else
    kubectl "$@" >/dev/null 2>&1
  fi
}

# Check if kubectl is available
check_prerequisites() {
  log "Checking prerequisites..."

  if ! command -v kubectl &> /dev/null; then
    fail "kubectl not found. Please install kubectl."
  fi
  success "kubectl is installed"

  if ! kubectl cluster-info &> /dev/null; then
    fail "Cannot connect to Kubernetes cluster"
  fi
  success "Connected to Kubernetes cluster"

  # Check if operator is running
  if ! kubectl get deployment language-operator -n kube-system &> /dev/null; then
    fail "Language operator not found in kube-system namespace"
  fi

  local ready_replicas
  ready_replicas=$(kubectl get deployment language-operator -n kube-system -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
  if [ "$ready_replicas" -eq 0 ]; then
    fail "Language operator is not ready"
  fi
  success "Language operator is running ($ready_replicas replica(s))"
}

# Cleanup existing resources
cleanup() {
  log "Cleaning up existing resources..."

  # Delete namespaced resources first
  if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    # Delete resources in order (namespaced)
    kubectl delete languageagent --all -n "$NAMESPACE" --ignore-not-found=true --wait=false
    kubectl delete languagetool --all -n "$NAMESPACE" --ignore-not-found=true --wait=false
    kubectl delete languagemodel --all -n "$NAMESPACE" --ignore-not-found=true --wait=false
    kubectl delete languagepersona --all -n "$NAMESPACE" --ignore-not-found=true --wait=false

    # Wait a bit for finalizers
    sleep 3
  fi

  # Delete cluster-scoped LanguageCluster (not namespaced)
  if kubectl get languagecluster demo &> /dev/null; then
    kubectl delete languagecluster demo --ignore-not-found=true --wait=false
  fi

  # Wait a bit for finalizers
  sleep 5

  # Force delete namespace if it exists
  if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true --wait=true --timeout=60s || true
  fi

  success "Cleanup completed"
}

# Wait for resource to be ready
wait_for_resource() {
  local resource_type=$1
  local resource_name=$2
  local namespace=$3
  local timeout=${4:-$TIMEOUT}

  log "Waiting for $resource_type/$resource_name to be ready (timeout: ${timeout}s)..."

  local elapsed=0
  local interval=5

  while [ $elapsed -lt "$timeout" ]; do
    local status
    status=$(kubectl get "$resource_type" "$resource_name" -n "$namespace" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")

    if [ "$status" = "Ready" ] || [ "$status" = "Running" ]; then
      success "$resource_type/$resource_name is ready"
      return 0
    elif [ "$status" = "Failed" ]; then
      fail "$resource_type/$resource_name has failed"
      return 1
    fi

    sleep $interval
    elapsed=$((elapsed + interval))
  done

  fail "Timeout waiting for $resource_type/$resource_name to be ready"
  return 1
}

# Wait for pod to be running
wait_for_pod() {
  local pod_selector=$1
  local namespace=$2
  local expected_ready=$3  # e.g., "2/2" for sidecar
  local timeout=${4:-$TIMEOUT}

  log "Waiting for pod matching '$pod_selector' to be running (timeout: ${timeout}s)..."

  local elapsed=0
  local interval=2  # Check more frequently

  while [ $elapsed -lt "$timeout" ]; do
    local pod_status
    pod_status=$(kubectl get pods -n "$namespace" -l "$pod_selector" -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "NotFound")

    # Check for image pull errors specifically
    local container_statuses
    container_statuses=$(kubectl get pods -n "$namespace" -l "$pod_selector" -o jsonpath='{.items[0].status.containerStatuses[*].state.waiting.reason}' 2>/dev/null || echo "")

    if [[ "$container_statuses" =~ ImagePullBackOff|ErrImagePull ]]; then
      error "Pod matching '$pod_selector' cannot pull images"
      error "This likely means registry credentials are missing."
      error ""
      error "To fix, create the registry-credentials secret in namespace $namespace:"
      error "  kubectl create secret docker-registry registry-credentials \\"
      error "    --docker-server=git.theryans.io \\"
      error "    --docker-username=ci \\"
      error "    --docker-password=YOUR_PASSWORD \\"
      error "    -n $namespace"
      error ""
      error "Then patch the default ServiceAccount:"
      error "  kubectl patch serviceaccount default -n $namespace \\"
      error "    -p '{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}'"
      fail "ImagePullBackOff detected"
      return 1
    fi

    if [ "$pod_status" = "Running" ]; then
      # Check if containers are ready
      local ready_status
      ready_status=$(kubectl get pods -n "$namespace" -l "$pod_selector" -o jsonpath='{.items[0].status.containerStatuses[*].ready}' 2>/dev/null || echo "")

      if [ "$expected_ready" = "any" ]; then
        success "Pod matching '$pod_selector' is running"
        return 0
      fi

      # Count ready containers
      local ready_count
      ready_count=$(echo "$ready_status" | grep -o "true" | wc -l | tr -d '[:space:]' || echo "0")
      local expected_count
      expected_count=$(echo "$expected_ready" | cut -d'/' -f2 | tr -d '[:space:]')

      if [ "$ready_count" -ge "$expected_count" ]; then
        success "Pod matching '$pod_selector' is running with $ready_count/$expected_count containers ready"
        return 0
      fi
    elif [[ "$pod_status" =~ Error|Failed|CrashLoopBackOff ]]; then
      fail "Pod matching '$pod_selector' has failed with status: $pod_status"
      return 1
    fi

    sleep $interval
    elapsed=$((elapsed + interval))
  done

  fail "Timeout waiting for pod matching '$pod_selector' to be running"
  return 1
}

# Apply manifests in order
apply_manifests() {
  log "Applying manifests in order..."

  # 1. Cluster (creates namespace)
  log "Applying cluster.yaml..."
  run_kubectl apply -f cluster.yaml
  success "Applied cluster.yaml"

  # Wait for namespace
  log "Waiting for namespace $NAMESPACE to be created..."
  local elapsed=0
  while [ $elapsed -lt 30 ]; do
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
      success "Namespace $NAMESPACE created"
      break
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done

  # 2. Model
  log "Applying model.yaml..."
  run_kubectl apply -f model.yaml
  success "Applied model.yaml"

  # 3. Persona
  log "Applying persona.yaml..."
  run_kubectl apply -f persona.yaml
  success "Applied persona.yaml"

  # 4. Tool
  log "Applying tool.yaml..."
  run_kubectl apply -f tool.yaml
  success "Applied tool.yaml"

  # 5. Agent
  log "Applying agent.yaml..."
  run_kubectl apply -f agent.yaml
  success "Applied agent.yaml"
}

# Verify cluster
verify_cluster() {
  log "Verifying LanguageCluster..."

  # LanguageClusters are cluster-scoped, not namespaced
  if ! kubectl get languagecluster demo &> /dev/null; then
    fail "LanguageCluster demo not found"
    return 1
  fi

  # Wait for the cluster to be ready
  log "Waiting for LanguageCluster demo to be ready (timeout: ${TIMEOUT}s)..."
  local elapsed=0
  local interval=5

  while [ $elapsed -lt "$TIMEOUT" ]; do
    local status
    status=$(kubectl get languagecluster demo -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")

    if [ "$status" = "Ready" ]; then
      success "LanguageCluster demo is ready"
      return 0
    elif [ "$status" = "Failed" ]; then
      fail "LanguageCluster demo has failed"
      return 1
    fi

    sleep $interval
    elapsed=$((elapsed + interval))
  done

  fail "Timeout waiting for LanguageCluster demo to be ready"
  return 1
}

# Verify model
verify_model() {
  log "Verifying LanguageModel..."

  if ! kubectl get languagemodel magistral-small-2509 -n "$NAMESPACE" &> /dev/null; then
    fail "LanguageModel magistral-small-2509 not found"
    return 1
  fi

  wait_for_resource languagemodel magistral-small-2509 "$NAMESPACE"

  # Check deployment
  log "Checking model deployment..."
  wait_for_pod "app.kubernetes.io/name=magistral-small-2509" "$NAMESPACE" "1/1"

  # Check service
  if ! kubectl get service magistral-small-2509 -n "$NAMESPACE" &> /dev/null; then
    fail "Model service not found"
    return 1
  fi
  success "Model service exists"

  # Check NetworkPolicy
  if ! kubectl get networkpolicy magistral-small-2509-egress -n "$NAMESPACE" &> /dev/null; then
    warning "Model NetworkPolicy not found (may not be created yet)"
  else
    success "Model NetworkPolicy exists"
  fi
}

# Verify tool
verify_tool() {
  log "Verifying LanguageTool..."

  if ! kubectl get languagetool web-tool -n "$NAMESPACE" &> /dev/null; then
    fail "LanguageTool web-tool not found"
    return 1
  fi

  wait_for_resource languagetool web-tool "$NAMESPACE"

  # Tool is in sidecar mode, so no standalone pod
  local deployment_mode
  deployment_mode=$(kubectl get languagetool web-tool -n "$NAMESPACE" -o jsonpath='{.spec.deploymentMode}' 2>/dev/null || echo "")

  if [ "$deployment_mode" = "sidecar" ]; then
    success "Tool is configured in sidecar mode"
  else
    # Check for standalone deployment
    log "Checking tool deployment..."
    wait_for_pod "app.kubernetes.io/name=web-tool" "$NAMESPACE" "1/1"
  fi

  # Check NetworkPolicy
  if ! kubectl get networkpolicy web-tool-egress -n "$NAMESPACE" &> /dev/null; then
    warning "Tool NetworkPolicy not found (may not be created yet)"
  else
    success "Tool NetworkPolicy exists"
  fi
}

# Verify agent
verify_agent() {
  log "Verifying LanguageAgent..."

  if ! kubectl get languageagent demo-agent -n "$NAMESPACE" &> /dev/null; then
    fail "LanguageAgent demo-agent not found"
    return 1
  fi

  wait_for_resource languageagent demo-agent "$NAMESPACE"

  # Check pod with sidecar (should have 2 containers)
  log "Checking agent pod with sidecar..."
  wait_for_pod "app.kubernetes.io/name=demo-agent" "$NAMESPACE" "2/2"

  # Verify container count
  local container_count
  container_count=$(kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=demo-agent" -o jsonpath='{.items[0].spec.containers[*].name}' 2>/dev/null | wc -w || echo "0")

  if [ "$container_count" -eq 2 ]; then
    success "Agent pod has 2 containers (agent + tool sidecar)"
  else
    fail "Agent pod has $container_count containers, expected 2"
    return 1
  fi

  # Check workspace PVC
  if ! kubectl get pvc demo-agent-workspace -n "$NAMESPACE" &> /dev/null; then
    fail "Workspace PVC not found"
    return 1
  fi

  local pvc_status
  pvc_status=$(kubectl get pvc demo-agent-workspace -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

  if [ "$pvc_status" = "Bound" ]; then
    success "Workspace PVC is bound"
  else
    warning "Workspace PVC status: $pvc_status"
  fi

  # Check environment variables
  log "Checking agent environment variables..."
  local pod_name
  pod_name=$(kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=demo-agent" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

  if [ -n "$pod_name" ]; then
    if kubectl get pod "$pod_name" -n "$NAMESPACE" -o json | grep -q "MODEL_ENDPOINTS"; then
      success "Agent has MODEL_ENDPOINTS environment variable"
    else
      warning "MODEL_ENDPOINTS not found in agent environment"
    fi

    if kubectl get pod "$pod_name" -n "$NAMESPACE" -o json | grep -q "MCP_SERVERS"; then
      success "Agent has MCP_SERVERS environment variable"
    else
      warning "MCP_SERVERS not found in agent environment"
    fi
  fi
}

# Check pod logs for errors
check_logs() {
  log "Checking pod logs for errors..."

  # Check model pod
  local model_pod
  model_pod=$(kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=magistral-small-2509" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  if [ -n "$model_pod" ]; then
    if kubectl logs "$model_pod" -n "$NAMESPACE" --tail=50 2>&1 | grep -iq "error\|fatal\|panic"; then
      warning "Found errors in model pod logs"
      if [ "$VERBOSE" = true ]; then
        kubectl logs "$model_pod" -n "$NAMESPACE" --tail=50
      fi
    else
      success "Model pod logs look healthy"
    fi
  fi

  # Check agent pod (both containers)
  local agent_pod
  agent_pod=$(kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=demo-agent" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  if [ -n "$agent_pod" ]; then
    # Check agent container
    if kubectl logs "$agent_pod" -c agent -n "$NAMESPACE" --tail=50 2>&1 | grep -iq "error\|fatal\|panic"; then
      warning "Found errors in agent container logs"
      if [ "$VERBOSE" = true ]; then
        kubectl logs "$agent_pod" -c agent -n "$NAMESPACE" --tail=50
      fi
    else
      success "Agent container logs look healthy"
    fi

    # Check tool sidecar
    if kubectl logs "$agent_pod" -c tool-web-tool -n "$NAMESPACE" --tail=50 2>&1 | grep -iq "error\|fatal\|panic"; then
      warning "Found errors in tool sidecar logs"
      if [ "$VERBOSE" = true ]; then
        kubectl logs "$agent_pod" -c tool-web-tool -n "$NAMESPACE" --tail=50
      fi
    else
      success "Tool sidecar logs look healthy"
    fi
  fi
}

# Print summary
print_summary() {
  echo ""
  echo "========================================"
  echo "Verification Summary"
  echo "========================================"
  kubectl get languagecluster,languagemodel,languagetool,languageagent -n "$NAMESPACE" 2>/dev/null || true
  echo ""
  kubectl get pods,pvc,svc,networkpolicy -n "$NAMESPACE" 2>/dev/null || true
  echo "========================================"
}

# Main execution
main() {
  echo ""
  echo "========================================"
  echo "Language Operator E2E Verification"
  echo "========================================"
  echo ""

  # Change to script directory
  cd "$(dirname "$0")"

  if [ "$CLEANUP_ONLY" = true ]; then
    cleanup
    success "Cleanup completed"
    exit 0
  fi

  # Run verification
  check_prerequisites
  cleanup
  apply_manifests

  echo ""
  log "Starting verification (timeout: ${TIMEOUT}s)..."
  echo ""

  verify_cluster
  verify_model
  verify_tool
  verify_agent

  # Wait a bit for everything to stabilize
  log "Waiting for system to stabilize..."
  sleep 10

  check_logs
  print_summary

  echo ""
  success "<� All verification checks passed!"
  echo ""
  echo "To clean up, run: $0 --cleanup-only"
  echo ""
}

# Run main function
main "$@"
