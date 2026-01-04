# Agent Memory Bank

## Current Focus Areas (Dec 27, 2025)

### Active Issues
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY** 
- **Issue #55**: Telemetry adapter endpoint validation panics - **BACKLOG**

### Recently Completed (Jan 4, 2026)
- ✅ **Event Recording Centralization**: Successfully implemented centralized EventManager utility to eliminate 40+ duplicate event recording patterns across controllers. Created `/src/pkg/events/manager.go` with 25+ standardized event reason constants and type-safe recording methods. Refactored LanguageAgentReconciler and LanguageToolReconciler to use EventManager, reducing repetitive code and providing consistent event taxonomy. All tests pass and build successful. Remaining controllers (LanguageModel, LanguagePersona, LanguageCluster) can follow same pattern for complete migration.

### Recently Completed (Jan 2, 2026)
- ✅ **Issue #228**: Network policy issue with web tool - Root cause was incomplete egress configuration in web tool catalog template (missing `to` field in egress rules). Fixed deployed instance by patching LanguageTool with proper DNS targets (`*.duckduckgo.com`) and CIDR blocks (`0.0.0.0/0`). Verified NetworkPolicy now allows external HTTPS access and web search functionality restored. Opened upstream issue language-tools#8 for permanent catalog fix. Key lesson: egress rules need both `ports` and `to` fields, operator skips rules with `rule.To == nil`

### Recently Completed (Jan 1, 2026)
- ✅ **Issue #226**: Add support for disabling signups - Implemented `LANGOP_SIGNUPS_DISABLED` environment variable to control signup access. When enabled, signup page shows error message and redirects to login, API returns 403 for direct signups, and login page hides "Create Account" link. Invitation system continues to work normally with disabled signups. Added Helm chart configuration via `dashboard.features.signupsDisabled` value. Maintains backward compatibility with default disabled=false (commit pending)

### Previously Completed (Dec 30, 2025)  
- ✅ **Issue #224**: Agent code synthesizer produces poorly formatted Ruby code - Implemented pattern-based Ruby code formatter to fix LLM-generated code formatting issues. Added FormatRubyCode function with task definition parameter alignment fixes, empty hash normalization ({  } → {}), and excessive blank line removal. Integrated into both agent and task synthesis pipelines with comprehensive test coverage. Addresses the main formatting problems observed in ConfigMaps where task parameters had poor indentation and spacing (commit 2311d2e)

### Previously Completed (Dec 28, 2025)
- ✅ **Issue #223**: Learning controller not creating new LanguageAgentVersions after manual optimization - Fixed ClickHouse query logic where `service.name` attribute was incorrectly filtering on `SpanAttributes['service.name']` instead of the dedicated `ServiceName` column. Added RBAC annotations, enhanced error logging, and comprehensive unit tests. Verified with real telemetry data showing 1,982 traces now properly found for optimization (commit c34679c)
- ✅ **Issue #219**: Synthesis hash management bug causing infinite loops - Fixed root cause in `createInitialAgentVersion()` where LanguageAgentVersion v1 was returned with stale hash annotations when instructions changed. Implemented atomic hash + code updates to prevent both infinite synthesis loops and stale code usage. Added comprehensive unit tests covering infinite loop scenarios (commit 268f8ca)

### Previously Completed (Dec 27)
- ✅ **Issue #218**: Service selector routing to trigger pods - Fixed service selector to only route workspace API calls to agent pods, not trigger pods. Added `langop.io/component=agent` label to deployment pods and updated service selector accordingly (commits 3fa451f, 743447e)

### Previously Completed (Dec 25-26)
- ✅ **Issue #208**: Console auto-load last conversation - Implemented localStorage persistence to auto-restore last active conversation on Console page navigation (commit 08bdc16)
- ✅ **Issue #201**: Console workspace back button - Added ChevronLeft navigation from file viewer to file list with Escape key support (commit 36f6926)
- ✅ **Issue #202**: Console workspace file name casing - Added `style={{ textTransform: 'none' }}` to workspace-file-tree.tsx:220 (commit 8d0026d)
- ✅ **Issue #203**: Console conversation highlighting - Fixed conversationDbId matching, added dropdown menus, border styling (commits f303727, 636b7cf, 33a17fd, 951de3c)
- ✅ **Issue #207**: Markdown rendering in chat - Added react-markdown with role-based rendering (commit 28cfdae)
- ✅ **Issue #198**: Agent standby mode - Migrated scheduled agents from CronJob to Deployment+CronJobTrigger (commit e71d72d)
- ✅ **Issue #192**: Persona deletion UI - Complete delete workflow with confirmation dialogs (commits 2fb801f, b5ad83b)
- ✅ **Issue #191**: AI persona CRD validation - Fixed stale CRD deployment (commit 3efbc7f)
- ✅ **Issue #190**: AI persona generation - Fixed NetworkPolicy, model name resolution, form population (commits 31839b1, 184226c, d08425f)

### Critical Infrastructure Fixed (Dec 18-23)
- ✅ **Issue #187**: SSE watch stream crashes - Fixed ReadableStream controller safety with sse-watch-helper.ts (commits 5dd14fd, cc5ac35)
- ✅ **Issue #186**: Missing Kubernetes Events - Added EventRecorder to 4 controllers, 21 event types (commit d195c2c)  
- ✅ **Issue #178**: Log viewer UX issues - Fixed auto-scroll, added max-h-[60vh] container (commit 7d3c5da)
- ✅ **Issue #177**: Workspace pod lifecycle - Fixed 30min timeout handling, auto-cleanup (commit 2fb8112)
- ✅ **Issue #174**: NetworkPolicy UI - Added Security tab with egress rule editor (commit varies)
- ✅ **Issue #173**: API consolidation - Removed legacy organization-scoped APIs, standardized cluster-scoped (commit varies)
- ✅ **Issue #171**: Error handling - Created api-error-handler.ts, cluster-validation.ts utilities (commit varies)
- ✅ **Issue #170**: Cluster filtering - Unified filterByClusterRef across 5 API routes (commit varies)
### Dashboard Foundation Completed (Dec 13-16)
- ✅ **Major milestone**: Cluster-scoped CRUD routing - 25+ issues resolved
  - Fixed 404 errors across creation pages (#155, #118, #117, #116)
  - Implemented complete API endpoints for agents, tools, personas
  - Enhanced dashboard counts and resource management
  - Added proper authentication patterns and error handling
  - Resolved TypeScript compilation and build issues
  - Implemented clickable tiles, navigation, and accessibility features

## Critical Development Context

### Deployment Rules
- ⚠️ **Operator**: CI pipeline only, no local Docker builds
- **Dashboard**: ROOT directory `docker compose up` only → http://localhost:3000
- **Login**: "james@theryans.io" / "password123"  
- ❌ **NEVER**: components/dashboard/docker-compose.yml (deprecated)
- ❌ **NEVER**: `npm run build` locally (causes file watch issues & memory bloat that breaks dev environment)

### Key Patterns
- **k8s-client.ts**: Handles demo/live mode differences: `{ body: { items: [...] } }` vs `{ data: { items: [] } }`
- **Error handling**: Use `error instanceof Error`, avoid implicit any
- **TypeScript**: Strict mode compliance required
- **API structure**: All routes cluster-scoped `/api/clusters/[name]/...`

### Testing Requirements
- ✅ Manual testing before commit
- ✅ Verify CI builds pass
- ✅ Cluster-scoped CRUD workflows: `/clusters/[name]/{resource}/new`
- ✅ Playwright available for UI automation

### Common Issue Patterns
- **TypeScript**: Null safety, error type handling
- **API parsing**: Demo vs live Kubernetes response structures  
- **CSS**: Tailwind v3 syntax, PostCSS configuration
- **Routing**: Missing cluster-scoped endpoints → 404s