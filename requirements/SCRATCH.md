# Agent Memory Bank

## Current Outstanding Issues (Nov 26, 2025)

### Priority 1: Validation Pipeline Foundation
- ✅ **Issue #51**: Synthesis pipeline ignores schema validation failures - **RESOLVED** (Nov 26) - Fixed inconsistent error handling
- ✅ **Issue #72**: Synthesis validator continues after validation failure - **RESOLVED** (Nov 26) - Fixed schema validation error collection

### Priority 2: Security & Configuration (Current Sprint)
- ✅ **Issue #74**: Registry whitelist bypass via operator-config version field - **RESOLVED** (Nov 26) - Added strict ConfigMap validation
- ✅ **Issue #52**: ConfigMap version validation allows zero/negative versions - **RESOLVED** (Nov 26) - Added CurrentVersion validation in learning controller

### Priority 3: Resource Lifecycle Management (Nov 29)  
- ✅ **Issue #84**: Deleting clusters leaves orphans - **RESOLVED** (Nov 27) - Added finalizer cleanup for LanguageCluster controller
- ✅ **Issue #71**: Learning controller status update failures - **RESOLVED** (Nov 26) - Fixed JSON serialization, ConfigMap size validation, and status data rotation
- ✅ **Issue #68**: Telemetry adapter configuration validation failures - **RESOLVED** (Nov 26) - Added strict URL validation to prevent runtime failures
- ✅ **Issue #53**: IPv6 registry validation bracket handling failures - **RESOLVED** (Nov 26) - Invalid bug report, implementation is complete and working
- ✅ **Issue #79**: Remove unused utility functions from controllers - **RESOLVED** (Nov 26) - Removed MergeLabels() function, 10 lines dead code cleanup
- ✅ **Issue #87**: SigNoz API compatibility issue preventing learning system trace queries - **RESOLVED** (Nov 27) - Updated adapter for Query Builder v5 format, fixed SigNoz v0.103.0 compatibility
- ✅ **Issue #89**: DSL synthesis generates invalid Ruby syntax for task definitions - **RESOLVED** (Nov 29) - Fixed template syntax by adding parentheses around neural task arguments
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY**

### Lower Priority: Security Edge Cases (Backlog)
- ✅ **Issue #76**: IPv6 registry validation bypass (malformed addresses) - **RESOLVED** (Nov 26) - Fixed bracket validation in image registry parser
- ✅ **Issue #65**: IPv6 registry parsing vulnerability - **INVALID** (Nov 26) - Comprehensive security analysis confirmed no vulnerability exists
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
- **Phase 2 Progress**: Resolved 20 critical issues (memory leaks, race conditions, security vulnerabilities, DSL syntax)
- **DSL Synthesis Fix (Nov 29)**: Fixed critical Ruby syntax error in task definitions, unblocking agent synthesis pipeline
- **Resource Lifecycle (Nov 27)**: Fixed LanguageCluster orphaned resource cleanup with finalizer implementation
- **Controller Optimization**: Extracted common pattern, eliminated ~150 lines duplicate code
- **Dead Code Cleanup (Nov 26)**: Removed 398 lines of unused code from cmd/main.go, eliminated duplicate registry loading logic
- **Next Optimization**: Learning controller migration to ReconcileHelper pattern (15+ lines reduction, standardization)