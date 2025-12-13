# Agent Memory Bank

## Current Focus Areas (Dec 13, 2025)

### Active Issues
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY**  
- **Issue #55**: Telemetry adapter endpoint validation panics - **BACKLOG**

### Dashboard Development (Recent)
- ✅ **Issue #121**: Complete cluster-scoped CRUD routing for Tools, Personas, and Agents - **RESOLVED** (Dec 13) - Implemented 9 missing routing files for cluster-scoped resource creation/editing, fixed TypeScript compilation errors, manually tested with Playwright
- ✅ **Issue #115**: Settings page returns 404 error - **RESOLVED** (Dec 13) - Added missing page.tsx with redirect, fixed AuthenticatedLayout wrapper, updated Docker volume mounts
- ✅ **Issue #114**: Cluster listing page shows incorrect data and missing clusters - **RESOLVED** (Dec 13) - Fixed API response parsing for different k8s client structures (live vs demo mode)

### Recent Issue Patterns (Nov-Dec 2025)
- **TypeScript Compilation**: Multiple fixes for strict mode compliance, null safety, error type handling
- **Dashboard API Integration**: Response parsing inconsistencies between demo/live Kubernetes modes
- **PostCSS/Tailwind**: Version compatibility issues (v4 syntax on v3 tooling)
- **Docker Development**: Volume mount configuration for live file updates

## Critical Development Context

### Deployment Constraints
- ⚠️ **Operator deployment**: CI pipeline only, no local Docker builds
- **Workflow**: Push to origin → CI builds image → manual install via ~/workspace/system/manifests/language-operator
- **Dashboard**: Can run locally with `npm run dev` for frontend development

### Next.js Dashboard Architecture
- **API Routes**: `/api/clusters`, `/api/models`, etc. proxy to Kubernetes client
- **Response Handling**: Support both `{ body: { items: [...] } }` (live K8s) and `{ data: { items: [] } }` (demo mode)
- **Error Patterns**: Use `error instanceof Error` checks, avoid implicit any types
- **Testing**: Manual testing required, playwright available for UI testing

### Key Technical Patterns
- **k8s-client.ts**: Handles demo mode fallbacks, null safety for API objects
- **ReconcileHelper[T]**: Standard pattern for new controllers (Go backend)
- **TypeScript**: Strict mode compliance, explicit error handling
- **CSS**: Use Tailwind v3 syntax, proper PostCSS configuration

### Development Standards
- ❌ **NEVER** implement stub/fake algorithms
- ✅ **ALWAYS** test implementations manually before committing  
- ✅ **FIX** TypeScript errors before pushing
- ✅ **VERIFY** CI builds pass before considering issues resolved
- ✅ **CRUD COMPLETENESS** - Use 8-point routing checklist (see requirements/development-standards.md)
- ✅ **CLUSTER-SCOPED TESTING** - Manually verify `/clusters/[name]/{resource}/new` workflows
- ✅ **NO FALSE COMPLETION** - 80% backend work ≠ 100% complete (missing cluster routes cause 404s)

## Recently Resolved (Context for future issues)
- **Dashboard TypeScript compilation**: Multiple fixes for null safety, error handling
- **CSS build issues**: PostCSS plugin updates, Tailwind v3/v4 compatibility
- **API response parsing**: Consistent handling across demo/live K8s environments
- **Docker volume configuration**: Live reload in development environment

## Completed Phases
- **Phase 1**: Core platform infrastructure (Go backend, CRDs, controllers)
- **Phase 2**: 20+ critical issues resolved (security, lifecycle, validation)
- **Phase 3**: Dashboard development and TypeScript modernization (ongoing)