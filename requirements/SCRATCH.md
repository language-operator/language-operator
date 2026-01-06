# Agent Memory Bank

## Active Work
- 🎯 **Issue #77**: Learning controller ConfigMap serialization failures - **READY**
- 🎯 **Issue #61**: Registry whitelist configuration drift - **READY** 
- **Issue #55**: Telemetry adapter endpoint validation panics - **BACKLOG**

## Recently Completed
- ✅ **Issue #242**: Organization edit page fixes - organization name editing and Usage tab error resolved
  - Fixed tab routing logic to separate General (name editing) and Usage Limits (quotas) content
  - Added organization name editing form with validation and save functionality  
  - Resolved Usage Limits tab error - now displays quota management without issues
  - Improved UX: removed Plan column, fixed Edit Organization dropdown navigation

## Development Environment

### Deployment Rules
- ⚠️ **Operator**: CI pipeline only, no local Docker builds
- **Dashboard**: ROOT directory `docker compose up postgres-dev dashboard-dev` → http://localhost:3000
- **Login**: "james@theryans.io" / "password123"  
- ❌ **NEVER**: components/dashboard/docker-compose.yml (deprecated)
- ❌ **NEVER**: `npm run build` locally (causes file watch issues & memory bloat)

### Port Conflict Debugging
- **Symptom**: Dashboard starts on port 3001 instead of 3000
- **Cause**: Existing npm dev servers occupy port 3000 (not Docker auto-incrementing)
- **Debug**: `netstat -tulpn | grep :3000` to identify blocking processes
- **Resolution**: `pgrep -f "npm run dev"` to find processes, ask for help killing them if permission denied
- **Key Principle**: Don't invent explanations for infrastructure behavior - investigate actual cause

### Testing Protocol
- ✅ Manual testing before commit (Playwright for UI changes)
- ✅ Verify CI builds pass via `gh run watch`
- ✅ Test cluster-scoped CRUD workflows: `/clusters/[name]/{resource}/new`
- ❌ **NEVER commit untested code**

## Architecture Patterns

### API Structure
- All routes cluster-scoped: `/api/clusters/[name]/...`
- k8s-client.ts handles demo/live mode differences: `{ body: { items: [...] } }` vs `{ data: { items: [] } }`
- Error handling: Use `error instanceof Error`, avoid implicit any
- TypeScript: Strict mode compliance required

### Navigation
- **Organization-scoped URLs**: Always use `getOrgUrl()` for paths like `/settings/organizations`
- **Never hardcode paths**: Breaks organization context, causes 404s
- **Pattern**: `router.push(getOrgUrl('/settings/organizations'))` not `router.push('/settings/organizations')`

### NetworkPolicy Rules
- **Egress requirements**: Must have both `ports` AND `to` fields
- **Operator behavior**: Skips rules with `rule.To == nil`
- **Common error**: Missing `to` field breaks external connectivity

## Common Issue Patterns

### 1. Navigation Bugs
- **Symptom**: 404 errors in organization settings, missing org context in URL
- **Root cause**: Hardcoded paths instead of `getOrgUrl()`
- **Fix**: Import `useOrganization` from `@/components/organization-provider`, use `getOrgUrl()`
- **Test**: Verify navigation maintains `/[org_id]/...` URL structure

### 2. TypeScript Errors
- **Pattern**: Null safety violations, implicit any usage
- **Solution**: Strict mode compliance, proper error type handling
- **Check**: `error instanceof Error` pattern for error handling

### 3. API Response Parsing
- **Issue**: Demo vs live Kubernetes response structure differences
- **Reference**: Check k8s-client.ts for proper handling patterns
- **Demo mode**: `{ body: { items: [...] } }`
- **Live mode**: `{ data: { items: [] } }`

### 4. Event Recording Duplication
- **Problem**: Repeated event recording code across controllers
- **Solution**: Use centralized `/src/pkg/events/manager.go` with standardized constants
- **Status**: 2/5 controllers migrated, 3 remaining (LanguageModel, LanguagePersona, LanguageCluster)

### 5. Infrastructure Debugging
- **Anti-pattern**: Inventing explanations for unexpected behavior
- **Correct approach**: Investigate actual system state first
- **Tools**: `netstat`, `pgrep`, `lsof` for process debugging
- **Escalation**: Ask for help with permissions rather than working around

## Critical Knowledge

### Real-First Development
- **Principle**: Always work with real ClickHouse data, never mock data shortcuts
- **Testing**: Use real telemetry adapters, not demo mode for final verification
- **Integration**: Test end-to-end with actual Kubernetes clusters

### Issue Investigation Thoroughness
- **Common mistake**: Fixing only the obvious symptom (e.g., Cancel button) 
- **Complete fix**: Test all related navigation paths (e.g., Back arrow, breadcrumbs)
- **Pattern**: UI issues often affect multiple components sharing the same broken pattern

### Technical Debt Tracking
- **EventManager adoption**: 3 controllers need migration to centralized events
- **Status Phase Constants**: Check for remaining hardcoded status strings
- **Organization-scoped navigation**: Audit settings pages for hardcoded paths