import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { filterByClusterRef } from '@/lib/cluster-utils'
import { validateClusterExists, validateResourceBelongsToCluster } from '@/lib/cluster-validation'
import { createErrorResponse, createSuccessResponse, handleKubernetesOperation, validateClusterNameFormat, createAuthenticationRequiredError, createPermissionDeniedError } from '@/lib/api-error-handler'
import { LanguageTool, LanguageToolListParams } from '@/types/tool'

// GET /api/clusters/[name]/tools - List all tools for specific cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      throw createPermissionDeniedError('view tools', 'cluster-scoped tools', userRole)
    }

    const { name: clusterName } = await params
    
    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    const url = new URL(request.url)
    const queryParams: LanguageToolListParams = {
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      type: url.searchParams.getAll('type') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
    }

    // Fetch all tools from organization namespace with proper error handling
    const response = await handleKubernetesOperation(
      'list tools',
      k8sClient.listLanguageTools(organization.namespace)
    )
    
    // Handle different response structures from k8s client
    const allTools = (response as any)?.items || (response.data as any)?.items || (response.body as any)?.items || []

    // Filter tools to only show those that belong to this specific cluster
    // Uses validation to handle orphaned resources gracefully
    const clusterTools = validateResourceBelongsToCluster(
      allTools, 
      clusterName, 
      { allowOrphanedResources: true }
    ) as LanguageTool[]

    // Apply search filtering 
    let filteredTools = clusterTools.filter((tool: LanguageTool) => {
      if (queryParams.search) {
        const searchLower = queryParams.search.toLowerCase()
        const nameMatch = tool.metadata.name?.toLowerCase().includes(searchLower)
        const typeMatch = tool.spec.type?.toLowerCase().includes(searchLower)
        const descMatch = tool.metadata?.annotations?.['langop.io/description']?.toLowerCase().includes(searchLower)
        if (!nameMatch && !typeMatch && !descMatch) return false
      }
      
      if (queryParams.type && queryParams.type.length > 0) {
        if (!queryParams.type.includes(tool.spec.type)) return false
      }
      
      if (queryParams.phase && queryParams.phase.length > 0) {
        if (!queryParams.phase.includes(tool.status?.phase || '')) return false
      }
      
      return true
    })

    // Apply sorting
    filteredTools.sort((a: LanguageTool, b: LanguageTool) => {
      const order = queryParams.sortOrder === 'desc' ? -1 : 1
      
      switch (queryParams.sortBy) {
        case 'name':
          return (a.metadata.name || '').localeCompare(b.metadata.name || '') * order
        case 'type':
          return (a.spec.type || '').localeCompare(b.spec.type || '') * order
        case 'phase':
          return ((a.status?.phase || '').localeCompare(b.status?.phase || '')) * order
        case 'age':
          const aTime = a.metadata.creationTimestamp ? new Date(a.metadata.creationTimestamp).getTime() : 0
          const bTime = b.metadata.creationTimestamp ? new Date(b.metadata.creationTimestamp).getTime() : 0
          return (bTime - aTime) * order // Default newest first, reverse for oldest first
        default:
          return 0
      }
    })

    // Apply pagination
    const startIndex = ((queryParams.page || 1) - 1) * (queryParams.limit || 50)
    const endIndex = startIndex + (queryParams.limit || 50)
    const paginatedTools = filteredTools.slice(startIndex, endIndex)

    return createSuccessResponse(paginatedTools, undefined, {
      total: filteredTools.length,
      page: queryParams.page || 1,
      limit: queryParams.limit || 50,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching cluster tools:', error)
    return createErrorResponse(error, 'Failed to fetch tools for cluster')
  }
}