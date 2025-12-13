# Development Standards

## CRUD Implementation Standards

### Problem Statement
Agents consistently report "CRUD implementation complete" at 80% completion, missing critical cluster-scoped routing that causes user-facing 404 errors. This creates a verification gap where complex backend work succeeds but simple UI routing fails.

### Complete vs Partial Implementation

**❌ PARTIAL (80% - Often Falsely Reported as Complete):**
- ✅ API routes (`/api/{resource}/*`) 
- ✅ Form components (`{Resource}Form`)
- ✅ Global pages (`/{resource}/new`, `/{resource}/[id]`)
- ❌ **Missing cluster-scoped routes** (`/clusters/[name]/{resource}/*`)

**✅ COMPLETE (100% - Actual User-Facing Functionality):**
- ✅ All 8 routing patterns implemented
- ✅ Cluster-scoped workflows tested end-to-end
- ✅ No 404 errors on any user path

### The 8-Point Routing Standard

Every CRUD implementation MUST include all 8 routing files:

#### Global Routes (Secondary User Path)
1. `/{resource}/new` - Global create page
2. `/{resource}/[id]` - Global detail page  
3. `/{resource}/[id]/edit` - Global edit page

#### Cluster-Scoped Routes (Primary User Path)
4. `/clusters/[name]/{resource}` - Cluster resource list page
5. `/clusters/[name]/{resource}/new` - Cluster create page ⚠️ **Most often missed**
6. `/clusters/[name]/{resource}/[id]` - Cluster detail page ⚠️ **Most often missed**
7. `/clusters/[name]/{resource}/[id]/edit` - Cluster edit page ⚠️ **Most often missed**
8. `/clusters/[name]/{resource}/[list]` - Additional list view if needed

### Mandatory Testing Pattern

**Before marking any CRUD task as "complete":**

1. **TypeScript Compilation** - Must compile with zero errors
2. **Cluster Workflow Testing** - Must manually test these URLs:
   ```
   /clusters/[test-cluster]/tools/new
   /clusters/[test-cluster]/tools/[test-tool]
   /clusters/[test-cluster]/tools/[test-tool]/edit
   ```
3. **End-to-End User Journey** - Complete create → edit → delete workflow
4. **Cross-Reference Validation** - Related resources display correctly
5. **Permission Testing** - Organization namespace isolation works

### File Structure Requirements

**Next.js App Router Structure:**
```
src/app/
├── {resource}/
│   ├── new/page.tsx           # Global create
│   ├── [id]/page.tsx          # Global detail  
│   └── [id]/edit/page.tsx     # Global edit
└── clusters/
    └── [name]/
        └── {resource}/
            ├── page.tsx               # List (if needed)
            ├── new/page.tsx           # Cluster create ⚠️
            ├── [id]/page.tsx          # Cluster detail ⚠️
            └── [id]/edit/page.tsx     # Cluster edit ⚠️
```

### Component Reuse Pattern

**Leverage existing components:**
- Reuse `{Resource}Form` component in both global and cluster pages
- Extract cluster name from `useParams()` in cluster-scoped pages
- Pass cluster context to forms via props
- Follow established patterns from existing implementations

### Definition of Done Criteria

**A CRUD implementation is considered complete ONLY when:**

- ✅ All 8 routing files exist and load without 404 errors
- ✅ Complete user journey tested manually (create → view → edit → delete)
- ✅ TypeScript compiles with zero errors
- ✅ CI tests pass (lint, unit, integration)
- ✅ Resources persist correctly in Kubernetes cluster
- ✅ Organization namespace isolation enforced
- ✅ Cross-references to related resources work

### Common Anti-Patterns to Avoid

1. **"Backend is done, frontend will be easy"** - Often the routing is forgotten
2. **Testing individual components only** - Integration matters more than units
3. **Ignoring TypeScript errors** - Compilation errors block production deployment
4. **Skipping manual testing** - Automated tests don't catch routing issues
5. **False completion reports** - 80% is not 100%

### Verification Script

Use the automated verification script to check completion:

```bash
node scripts/verify-crud-completeness.js --resource=tools
```

This script validates all 8 routing patterns exist and identifies missing files.

## Quality Standards

### TypeScript Requirements
- **Strict mode compliance** - No `any` types without explicit annotation
- **Null safety** - Handle `undefined` and `null` explicitly  
- **Error boundaries** - Proper error handling in components
- **Type guards** - Validate external data structures

### Testing Requirements
- **Manual testing mandatory** - No CRUD task complete without hands-on verification
- **End-to-end focus** - Test complete user workflows, not just individual functions
- **Playwright available** - Use browser automation for systematic UI testing
- **CI integration** - All tests must pass before marking tasks complete

### Code Quality
- **Component reuse** - Don't duplicate forms, leverage existing patterns
- **Consistent error handling** - Follow established error message patterns
- **Accessibility compliance** - Use proper ARIA labels, keyboard navigation
- **Performance considerations** - Optimize API calls, implement proper loading states

This systematic approach ensures that CRUD implementations are genuinely complete and provide working functionality for end users.