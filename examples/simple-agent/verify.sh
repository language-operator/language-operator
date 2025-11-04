#!/bin/bash
# Language Operator Verification Script
#
# Verifies that all resources are properly created and running.
# Run deploy.sh first to deploy the resources.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="demo"
VERBOSE="${VERBOSE:-false}"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --verbose|-v)
      VERBOSE=true
      shift
      ;;
    --namespace|-n)
      NAMESPACE="$2"
      shift 2
      ;;
    --help|-h)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Verifies that simple-agent example is properly deployed."
      echo ""
      echo "Options:"
      echo "  --verbose, -v       Show detailed output"
      echo "  --namespace, -n     Namespace to check (default: demo)"
      echo "  --help, -h          Show this help message"
      echo ""
      echo "Examples:"
      echo "  $0                  # Verify deployment in 'demo' namespace"
      echo "  $0 --verbose        # Show detailed verification output"
      echo ""
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Run with --help for usage information"
      exit 1
      ;;
  esac
done

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_resource() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3

    if kubectl get "$resource_type" "$resource_name" -n "$namespace" &>/dev/null; then
        log_success "$resource_type/$resource_name exists"
        return 0
    else
        log_error "$resource_type/$resource_name not found"
        return 1
    fi
}

check_pod_ready() {
    local label=$1
    local namespace=$2
    local timeout=${3:-30}

    if kubectl wait --for=condition=ready pod -l "$label" -n "$namespace" --timeout="${timeout}s" &>/dev/null; then
        log_success "Pod with label $label is ready"
        return 0
    else
        log_warning "Pod with label $label not ready after ${timeout}s"
        return 1
    fi
}

check_crd_status() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    local condition_type=$4

    local status=$(kubectl get "$resource_type" "$resource_name" -n "$namespace" \
        -o jsonpath="{.status.conditions[?(@.type=='$condition_type')].status}" 2>/dev/null)

    if [[ "$status" == "True" ]]; then
        log_success "$resource_type/$resource_name condition $condition_type is True"
        return 0
    else
        log_warning "$resource_type/$resource_name condition $condition_type is $status"
        if [[ "$VERBOSE" == "true" ]]; then
            kubectl get "$resource_type" "$resource_name" -n "$namespace" -o jsonpath='{.status.conditions}' | jq .
        fi
        return 1
    fi
}

# Main verification
log_info "Starting verification for namespace: $NAMESPACE"
echo ""

FAILED=0

# Check namespace exists
log_info "Checking namespace..."
if kubectl get namespace "$NAMESPACE" &>/dev/null; then
    log_success "Namespace $NAMESPACE exists"
else
    log_error "Namespace $NAMESPACE not found. Run ./deploy.sh first."
    exit 1
fi
echo ""

# Check LanguageCluster
log_info "Checking LanguageCluster..."
check_resource "languagecluster" "demo" "$NAMESPACE" || ((FAILED++))
echo ""

# Check LanguagePersona
log_info "Checking LanguagePersona..."
check_resource "languagepersona" "helpful-assistant" "$NAMESPACE" || ((FAILED++))
echo ""

# Check LanguageModel
log_info "Checking LanguageModel..."
check_resource "languagemodel" "magistral-small-2509" "$NAMESPACE" || ((FAILED++))
check_crd_status "languagemodel" "magistral-small-2509" "$NAMESPACE" "Ready" || ((FAILED++))

log_info "Checking model pod..."
check_pod_ready "app=magistral-small-2509" "$NAMESPACE" 30 || ((FAILED++))
echo ""

# Check LanguageTool
log_info "Checking LanguageTool..."
check_resource "languagetool" "web-tool" "$NAMESPACE" || ((FAILED++))
echo ""

# Check LanguageAgent
log_info "Checking LanguageAgent..."
check_resource "languageagent" "demo-agent" "$NAMESPACE" || ((FAILED++))
check_crd_status "languageagent" "demo-agent" "$NAMESPACE" "Synthesized" || ((FAILED++))

log_info "Checking agent deployment..."
if kubectl get deployment demo-agent -n "$NAMESPACE" &>/dev/null; then
    log_success "Agent deployment exists"

    log_info "Checking agent pod..."
    check_pod_ready "app=demo-agent" "$NAMESPACE" 30 || ((FAILED++))

    # Check pod containers
    AGENT_POD=$(kubectl get pod -n "$NAMESPACE" -l app=demo-agent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [[ -n "$AGENT_POD" ]]; then
        READY=$(kubectl get pod "$AGENT_POD" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[*].ready}')
        if [[ "$READY" == *"false"* ]]; then
            log_warning "Some containers in $AGENT_POD are not ready"
            if [[ "$VERBOSE" == "true" ]]; then
                kubectl get pod "$AGENT_POD" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses}' | jq .
            fi
            ((FAILED++))
        else
            log_success "All containers in $AGENT_POD are ready"
        fi
    fi
else
    log_error "Agent deployment not found"
    ((FAILED++))
fi
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [[ $FAILED -eq 0 ]]; then
    log_success "All verification checks passed!"
    echo ""
    log_info "To view agent logs:"
    echo "  kubectl logs -f -n $NAMESPACE -l app=demo-agent -c agent"
    echo ""
    log_info "To check agent status:"
    echo "  kubectl get languageagent demo-agent -n $NAMESPACE -o yaml"
    exit 0
else
    log_error "$FAILED verification check(s) failed"
    echo ""
    log_info "For detailed status, run:"
    echo "  kubectl get all -n $NAMESPACE"
    echo "  kubectl get languageagent,languagemodel,languagetool -n $NAMESPACE"
    exit 1
fi
