# Agent Memory Bank

## Current Focus Areas (Dec 26, 2025)

### Active Issues
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY** 
- **Issue #55**: Telemetry adapter endpoint validation panics - **BACKLOG**

### Recently Completed (Dec 25-26)
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
- ❌ **NEVER**: `npm run build` directly (crashes system)

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