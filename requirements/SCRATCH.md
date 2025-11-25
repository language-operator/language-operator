# Agent Memory Bank

## Current Priority Status (Nov 24, 2025)

### 🚀 READY Issues (Priority Order)
*All critical infrastructure issues completed! Moving to UX polish phase.*

### 📋 Remaining Work
- **Issue #36** - UX improvements (DNS documentation)
- **Issues #25-26** - Advanced learning features (error-triggered, metrics)  
- **Issue #29** - DSL v1 release (final milestone)

## Key Context

**Recently Completed (Foundation Work):**
- ✅ Issues #18-23: Synthesis template consistency & ConfigMap versioning
- ✅ Issues #32-39: Gateway API improvements & production fixes
- ✅ Issue #43: Helm chart webhook configurations (Nov 24)
- ✅ Issue #45: Controller panic fix with workspace size validation (Nov 24)
- ✅ Issue #44: Cron validation for Schedule field (Nov 24) - multi-layer validation
- ✅ Issue #41: Status update error handling in controller (Nov 24) - user visibility fix
- ✅ Issue #42: IPv6 registry validation support (Nov 24) - networking compatibility fix
- ✅ Issue #40: Remove legacy synthesize command with misleading API key errors (Nov 25) - legacy cleanup
- ✅ Issue #46: Telemetry adapter interface for learning system (Nov 25) - foundation for organic functions
- ✅ Issue #24: Deployment updates for learned ConfigMaps (learning pipeline complete)
- ✅ All core infrastructure for DSL v1 synthesis pipeline

**Critical Infrastructure Dependencies:**
- ✅ Issue #45 (operator stability) → ✅ #44 (validation) → ✅ #41 (error handling) → ✅ #42 (IPv6 support) → ✅ #24 (learning) ✅
- **ALL CRITICAL INFRASTRUCTURE COMPLETE!** Core platform is production-ready
- Current focus: UX polish (#36) and advanced learning features (#25-26)  
- DSL v1 release ready after advanced learning features (#25-26) complete

**Key Implementation Notes:**
- ConfigMap versioning: Always preserve v1 (initial synthesis)
- Gateway API: ReferenceGrant auto-creation for cross-namespace refs  
- Webhook timing: Route readiness verification before URL population
- Performance: Gateway API detection cached with 5-minute TTL
- Workspace validation: Multi-layer defense (CRD + webhook + controller) prevents panics
- Telemetry adapter: Interface foundation ready for SigNoz/Jaeger implementations (#47/#48)

**Deployment Process:**
- ⚠️ **CANNOT** build/deploy operator locally from source
- Must push changes to origin → CI builds image → manual install via ~/workspace/system/manifests/language-operator
- Use `git push` workflow, not `make operator` or local builds