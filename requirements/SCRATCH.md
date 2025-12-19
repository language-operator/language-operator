# Agent Memory Bank

## Current Focus Areas (Dec 18, 2025)

### Active Issues
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY**  
- **Issue #55**: Telemetry adapter endpoint validation panics - **BACKLOG**

### Dashboard Development (Recent)
- ✅ **Issue #174**: Implement NetworkPolicy egress UI for LanguageModel editing - **RESOLVED** (Dec 18) - Created comprehensive Security tab in model editing interface with visual egress rule editor supporting DNS names, CIDR blocks, port configuration, and quick-add buttons for common providers (OpenAI, Anthropic, Azure OpenAI, AWS Bedrock). Added individual model API endpoint (/api/clusters/[name]/models/[modelName]), implemented proper data format transformation between UI (integers) and CRD (port objects), and ensured full data persistence with loading. Fixed TypeScript compilation errors and verified CI/CD pipeline success. Enables network security configuration for models in Cilium NetworkPolicy-enforced clusters.
- ✅ **Issue #173**: Deprecate legacy organization-scoped APIs in favor of cluster-scoped APIs - **RESOLVED** (Dec 18) - Removed legacy API routes (/api/agents, /api/models, /api/personas) and updated all frontend hooks to require clusterName parameter. Fixed 8 component files to pass cluster context. Achieved single API pattern, reduced 590 lines of duplicate code, eliminated developer confusion about which APIs to use. Manual testing verified cluster-scoped routes (/clusters/ba/agents, /clusters/ba/models) working correctly with CRUD operations functional.
- ✅ **Issue #171**: Add comprehensive error handling to cluster-scoped APIs - **RESOLVED** (Dec 18) - Created centralized error handling utilities (api-error-handler.ts) and cluster validation helpers (cluster-validation.ts). Updated all 5 cluster-scoped APIs with standardized error responses, proper HTTP status codes, input validation, and graceful Kubernetes API failure handling. Implemented consistent error format with debug context, cluster existence validation, and orphaned resource detection. TypeScript compilation successful, all acceptance criteria met.
- ✅ **Issue #170**: Standardize cluster filtering logic across all APIs - **RESOLVED** (Dec 18) - Created reusable filterByClusterRef utility function and standardized all cluster-scoped APIs to use only spec.clusterRef for filtering, eliminating inconsistent fallback logic. Updated 5 API routes (models, agents, tools, personas, counts) to use unified filtering mechanism. Manually tested filtering behavior, all CI tests passing. Eliminates maintenance burden of supporting multiple filtering approaches.
- ✅ **Issue #166**: Cluster Settings page returns 404 - **RESOLVED** (Dec 16) - Fixed Settings button on cluster detail pages to correctly link to existing edit page (/clusters/[name]/edit) instead of non-existent /settings route. Simple one-line change resolved user navigation issue, manually tested with Playwright, all CI tests passing.  
- ✅ **Issue #155**: Agent creation API fails with multiple 404 errors - **RESOLVED** (Dec 15) - Eliminated all 404 errors in agent creation workflow by implementing proper cluster-scoped API endpoints. Added POST handler to /api/clusters/[name]/agents/, updated React hooks (useCreateAgent, useTools) to use cluster-scoped URLs, and fixed agent creation form to pass cluster context. Tools now load correctly (5 tools vs "No tools available"), form validation works properly, and agent creation reaches server without 404 errors.
- ✅ **Issue #157**: YAML preview displays invalid resource configuration - **RESOLVED** (Dec 15) - Enhanced YAML preview validation UX with visual feedback: red styling for "YAML Preview" header when validation fails, disabled Create Agent button when errors present, and real-time onChange validation. Manually tested with both valid and invalid configurations
- ✅ **Issue #153**: Agent creation page causes application-wide build failure - **RESOLVED** (Dec 15) - Fixed Next.js 16/Turbopack module resolution for react-syntax-highlighter by adding transpilePackages configuration. Agent creation page now loads properly with full YAML syntax highlighting functionality instead of critical Module not found errors
- ✅ **Issue #148**: Organization selection state not persisting across page navigation - **RESOLVED** (Dec 15) - Enhanced organization store with initializeActiveOrganization method, updated useOrganizations hook to initialize after loading, and added organization loading in AuthenticatedLayout on app startup. Organization selection now properly persists across page refreshes and navigation
- ✅ **Issue #149**: Agent details page displays empty content with 404 errors - **RESOLVED** (Dec 14) - Created cluster-scoped agent details API endpoint and updated React hooks, agent details pages now display full content with working Edit/Delete functionality and eliminated console 404 errors
- ✅ **Issue #150**: Agent count inconsistencies across cluster dashboard and agents page - **RESOLVED** (Dec 14) - Automatically fixed by Issue #151 agent count improvements, cluster dashboard now correctly shows "2" agents matching the agents page display
- ✅ **Issue #151**: Clusters overview displays incorrect agent counts systemically - **RESOLVED** (Dec 14) - Enhanced clusters API to dynamically calculate agent counts by querying LanguageAgent resources instead of relying on unpopulated cluster.status.agentCount field, now shows accurate "Total Agents: 2" and "synth - 2 agents"  
- ✅ **Issue #152**: Recent Activity displays data from inaccessible clusters and namespaces - **RESOLVED** (Dec 14) - Created Kubernetes events-based Recent Activity API to replace hardcoded "production" references, added proper permission filtering by organization membership, implemented real-time activity data with loading/error states
- ✅ **Issue #145**: Tools page generates 404 errors and shows inconsistent data - **NOT REPRODUCIBLE** (Dec 14) - Investigated thoroughly but could not reproduce reported 404 errors or data inconsistencies. Tools page working correctly with proper API responses and consistent data display between dashboard and tools catalog
- ✅ **Issue #143**: Dashboard displays incorrect resource counts for cluster components - **RESOLVED** (Dec 14) - Created missing agents API endpoint, fixed agents page to use cluster-scoped API, updated organization labels for stale resources, and corrected docker-compose NEXTAUTH_URL port configuration
- ✅ **Issue #146**: Tool installation fails with 500 Internal Server Error - **RESOLVED** (Dec 14) - Fixed authentication pattern in tool installation API to use proper database lookup instead of non-existent session.activeOrganization.namespace, added proper permission checks and organization labeling, improved 409 error handling
- ✅ **Issue #147**: Personas count inconsistency between dashboard and personas page - **RESOLVED** (Dec 14) - Fixed dashboard counts API to filter by organization ID and updated cluster dashboard page to use useResourceCounts hook instead of hardcoded zeros, now shows accurate resource counts
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
- **Dashboard**: Use docker compose in components/dashboard.  It runs Postgres and exposes the app at port 3000, which has hot-reloading capabilities.  It is the preferred development server, and you can log in with "james@theryans.io" and "password123"

### Next.js Dashboard Architecture
- **API Routes**: `/api/clusters`, `/api/models`, etc. proxy to Kubernetes client
- **Response Handling**: Support both `{ body: { items: [...] } }` (live K8s) and `{ data: { items: [] } }` (demo mode)
- **Error Patterns**: Use `error instanceof Error` checks, avoid implicit any types
- **Testing**: Manual testing required, playwright available for UI testing

### Key Technical Patterns
- **k8s-client.ts**: Handles fallbacks, null safety for API objects
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