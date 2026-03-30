# Agent Memory Bank

## Development Environment

### Deployment Rules
- **Operator**: CI pipeline only, no local Docker builds
- **Dashboard**: http://localhost:3000 — if not available, run `make dev-up` or `docker compose up`
- **Login**: `james@theryans.io` / `password123`
- **NEVER**: `npm run build` or `npm run dev` outside docker compose (port conflicts, memory bloat)
- **NEVER**: `components/dashboard/docker-compose.yml` (deprecated)

### Port Conflict Debugging
- **Symptom**: Dashboard starts on port 3001 instead of 3000
- **Cause**: Existing npm dev server occupies port 3000
- **Debug**: `netstat -tulpn | grep :3000`, `pgrep -f "npm run dev"`
- **Key Principle**: Investigate actual cause — don't invent explanations

### Testing Protocol
- Manual testing before commit (Playwright for UI changes)
- Verify CI builds pass via `gh run watch`
- Test cluster-scoped CRUD workflows: `/clusters/[name]/{resource}/new`
- **NEVER commit untested code**

## Architecture Patterns

### API Structure
- All routes cluster-scoped: `/api/clusters/[name]/...`
- k8s-client.ts handles demo/live mode: `{ body: { items: [...] } }` vs `{ data: { items: [] } }`
- TypeScript strict mode; use `error instanceof Error` for error handling

### Navigation
- Always use `getOrgUrl()` for internal paths — never hardcode `/settings/...` style paths
- Import `useOrganization` from `@/components/organization-provider`
- Broken pattern causes 404s and loses org context in URL

### NetworkPolicy Rules
- Egress rules must have both `ports` AND `to` fields
- Operator skips rules where `rule.To == nil`

### Worktree Branches
- When creating worktrees from origin/main, set upstream tracking explicitly:
  `git push origin HEAD:refs/heads/<branch>` to avoid accidentally pushing to main

## Completed Tech Debt (for reference)
- EventManager adoption: all 5 controllers migrated
- Gateway API removal (#298): dropped HTTPRoute/ReferenceGrant, renamed `ingressConfig→ingress`
- NetworkPeer selectors (#311, #334): Group/Service/NamespaceSelector/PodSelector wired
- Pre-commit hook worktree fix: `GIT_DIR` path confusion resolved
- CI path filtering for dashboard builds
- Docs audit #325-333: all closed (2026-03-29)
- NetworkRule.From ingress wiring (#310): closed
