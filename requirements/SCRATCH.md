# Agent Memory Bank

## Current Outstanding Issues (Nov 26, 2025)

### Priority 1: Validation Pipeline Foundation
- ✅ **Issue #51**: Synthesis pipeline ignores schema validation failures - **RESOLVED** (Nov 26) - Fixed inconsistent error handling
- ✅ **Issue #72**: Synthesis validator continues after validation failure - **RESOLVED** (Nov 26) - Fixed schema validation error collection

### Priority 2: Security & Configuration (Current Sprint)
- ✅ **Issue #74**: Registry whitelist bypass via operator-config version field - **RESOLVED** (Nov 26) - Added strict ConfigMap validation
- ✅ **Issue #52**: ConfigMap version validation allows zero/negative versions - **RESOLVED** (Nov 26) - Added CurrentVersion validation in learning controller

### Priority 3: Configuration & Validation Infrastructure (Ready - Nov 26)
- ✅ **Issue #71**: Learning controller status update failures - **RESOLVED** (Nov 26) - Fixed JSON serialization, ConfigMap size validation, and status data rotation
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY**
- 🎯 **Issue #68**: Telemetry adapter configuration validation failures - **READY**
- 🎯 **Issue #53**: IPv6 registry validation bracket handling failures - **READY**

### Lower Priority: Security Edge Cases (Backlog)
- **Issue #76**: IPv6 registry validation bypass (malformed addresses) 
- **Issue #65**: IPv6 registry parsing vulnerability
- **Issue #55**: Telemetry adapter endpoint validation panics

## Key Implementation Context

### Deployment Process
- ⚠️ **CANNOT** build/deploy operator locally from source
- Must push changes to origin → CI builds image → manual install via ~/workspace/system/manifests/language-operator
- Use `git push` workflow, not `make operator` or local builds

### Critical Guidelines
- **ConfigMap versioning**: Always preserve v1 (initial synthesis)
- **Gateway API**: ReferenceGrant auto-creation for cross-namespace refs
- **Testing**: Always run with race detection enabled
- **Code Style**: Use generic `ReconcileHelper[T]` pattern for new controllers

### Development Standards
- ❌ **NEVER** implement fake/stub algorithms or placeholder functions
- ✅ **ALWAYS** implement real, working algorithms from the start
- ✅ **TEST** all implementations thoroughly to ensure they work
- ✅ **PREFER** minimal working implementations over fake stubs

## Recent Completions Summary
- **Phase 1 Complete**: Core platform infrastructure delivered
- **Phase 2 Progress**: Resolved 18 critical issues (memory leaks, race conditions, security vulnerabilities)
- **Controller Optimization**: Extracted common pattern, eliminated ~150 lines duplicate code