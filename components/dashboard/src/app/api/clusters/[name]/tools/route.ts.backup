import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageTool, LanguageToolListParams } from '@/types/tool'

// GET /api/clusters/[name]/tools - List all tools for specific cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName } = await params
    if (!clusterName) {
      return NextResponse.json({ error: 'Cluster name is required' }, { status: 400 })
    }

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

    // Fetch all tools from organization namespace
    const response = await k8sClient.listLanguageTools(organization.namespace)
    
    // Handle different response structures from k8s client
    const allTools = (response as any)?.items || (response.data as any)?.items || (response.body as any)?.items || []

    // For now, show all tools from the organization namespace as available to any cluster
    // In the future, this can be refined to filter by actual cluster-tool relationships
    // when cluster-scoping is fully implemented in the CRD spec
    const clusterTools = allTools

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

    return NextResponse.json({
      success: true,
      data: paginatedTools,
      total: filteredTools.length,
      page: queryParams.page || 1,
      limit: queryParams.limit || 50,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching cluster tools:', error)
    return NextResponse.json({ 
      error: 'Failed to fetch tools for cluster',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}