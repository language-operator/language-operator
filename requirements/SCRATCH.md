# Agent Memory Bank

## Current Focus Areas (Dec 14, 2025)

### Active Issues
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY**  
- **Issue #55**: Telemetry adapter endpoint validation panics - **BACKLOG**

### Dashboard Development (Recent)
- ✅ **Issue #133**: Agent tiles not clickable - cannot access detail pages with edit/delete options - **RESOLVED** (Dec 14) - Made agent tiles clickable with navigation to detail pages, added hover effects, keyboard accessibility, and proper ARIA labels for screen readers
- ✅ **Issue #135**: React warning: Tools list items missing unique 'key' props - **RESOLVED** (Dec 14) - Removed incorrect key prop from ToolCard component, React keys should only be set by parent components
- ✅ **Issue #136**: Tool detail pages crash with 'tools?.find is not a function' error - **RESOLVED** (Dec 14) - Fixed JavaScript crash AND implemented complete tool detail functionality with cluster-scoped API endpoint, proper data display, and working Edit/Delete buttons
- ✅ **Issue #137**: Tool tiles not clickable - cannot access detail pages for configuration/removal - **RESOLVED** (Dec 14) - Made tool tiles clickable with navigation to detail pages, added accessibility features and hover effects
- ✅ **Issue #140**: Persona creation fails with 500 Internal Server Error - **RESOLVED** (Dec 14) - Fixed cluster-scoped persona creation payload structure, removed nested 'spec' wrapper to match flat LanguagePersonaFormData interface
- ✅ **Issue #138**: Tools search only filters Available Tools, not Installed Tools - **RESOLVED** (Dec 14) - Fixed LanguageTool data structure mapping and added consistent search filtering for both sections
- ✅ **Issue #141**: Complete CRUD functionality for personas - **RESOLVED** (Dec 14) - Fixed API routes, form validation, error handling, and cluster-scoped operations
- ✅ **Issue #126**: Dashboard counts API returns incorrect cluster count (0 vs 1) - **RESOLVED** (Dec 14) - Fixed k8s client response parsing to handle both live and demo modes correctly
- ✅ **Issue #120**: Dashboard Quick Actions are non-functional - **RESOLVED** (Dec 14) - Implemented cluster-aware Quick Actions with modal selection, now routes to cluster-scoped creation pages
- ✅ **Issue #119**: Model edit form shows 'Create Model' instead of 'Update Model' button - **RESOLVED** (Dec 14) - Fixed button text conditional logic in ModelForm component
- ✅ **Issue #118**: Agents creation page returns 404 error - **RESOLVED** (Dec 14) - Already resolved by prior cluster-scoped routing work, verified working correctly
- ✅ **Issue #117**: Personas creation page returns 404 error - **RESOLVED** (Dec 14) - Already resolved by prior cluster-scoped routing work, verified working correctly  
- ✅ **Issue #116**: Tools creation page returns 404 error - **RESOLVED** (Dec 14) - Already resolved by prior cluster-scoped routing work, verified working correctly
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