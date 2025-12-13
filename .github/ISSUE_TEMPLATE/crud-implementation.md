---
name: CRUD Implementation
about: Complete CRUD implementation for Kubernetes resources with systematic verification
title: 'Implement CRUD for [Resource Type]'
labels: enhancement, ready
assignees: ''
---

## CRUD Implementation Task

**Resource Type**: [e.g., LanguageAgent, LanguageModel, LanguageTool, LanguagePersona, LanguageCluster]

**Description**: Brief description of the resource and its functionality

## CRUD Implementation Checklist

### Backend (API Layer)
- [ ] **GET /api/{resource}** - List resources with auth/permissions/filtering
- [ ] **POST /api/{resource}** - Create resource with validation and K8s integration
- [ ] **GET /api/{resource}/[id]** - Get individual resource details
- [ ] **PUT /api/{resource}/[id]** - Update resource with validation
- [ ] **DELETE /api/{resource}/[id]** - Delete resource with confirmation

### Frontend (Component Layer) 
- [ ] **ResourceForm** - Complete form component with validation, error handling, all fields
- [ ] **ResourceList** - List component with filtering, search, pagination, status indicators
- [ ] **ResourceDetail** - Detail view component with tabs (Overview, Configuration, Metrics)

### Routing (Page Layer) - **Critical: Often Missed**
Global routes:
- [ ] **/{resource}/new** - Global create page
- [ ] **/{resource}/[id]** - Global detail page
- [ ] **/{resource}/[id]/edit** - Global edit page

⚠️ **Cluster-scoped routes (PRIMARY USER PATH):**
- [ ] **/clusters/[name]/{resource}** - Cluster resource list page
- [ ] **/clusters/[name]/{resource}/new** - Cluster create page ⚠️ **Often missed**
- [ ] **/clusters/[name]/{resource}/[id]** - Cluster detail page ⚠️ **Often missed**  
- [ ] **/clusters/[name]/{resource}/[id]/edit** - Cluster edit page ⚠️ **Often missed**

### End-to-End Verification (MANDATORY)
Must test the complete user journey:
- [ ] **Create from cluster page** - Navigate to cluster → click "Create {Resource}" → form submits successfully
- [ ] **Edit from cluster page** - Navigate to cluster → click resource → edit → saves successfully
- [ ] **Delete resource** - Confirmation dialog → successful deletion → returns to list
- [ ] **No 404 errors** - All links work, no broken routes
- [ ] **Cross-references work** - Related resources display correctly (e.g., Agent shows assigned Models/Tools)
- [ ] **Permissions enforced** - User can only access resources in their organization namespace
- [ ] **Data persistence** - Created resources appear in K8s cluster (`kubectl get {resource}`)

### Testing Requirements

**CRITICAL: Test cluster-scoped workflows** - This is the primary user path. 

Test these URLs manually:
```
/clusters/[test-cluster-name]/{resource}/new
/clusters/[test-cluster-name]/{resource}/[test-resource-name]  
/clusters/[test-cluster-name]/{resource}/[test-resource-name]/edit
```

**TypeScript compilation must pass** - Fix all compilation errors before committing

**Integration testing** - Use Playwright to verify complete user workflows

## Definition of Done

- ✅ All 8 checklist routing items complete (no missing files)
- ✅ All end-to-end verification items pass
- ✅ TypeScript compiles with no errors
- ✅ CI tests pass (linting, unit tests, integration tests)
- ✅ Manual testing confirms cluster-scoped workflows work
- ✅ Created resources persist in Kubernetes cluster

## Common Pitfalls to Avoid

1. **False completion at 80%** - Don't report "complete" when only global routes exist
2. **Missing cluster-scoped routes** - The 3 cluster routes are the most critical for users
3. **Untested implementations** - Always manually test the complete user journey
4. **TypeScript errors ignored** - Must compile cleanly before committing
5. **Skipped end-to-end testing** - Individual components working ≠ complete workflow working

## Success Criteria

User can navigate to any cluster page and successfully create, view, edit, and delete resources without encountering any 404 errors or broken functionality.