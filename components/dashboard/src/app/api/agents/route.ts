import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { k8sClient } from '@/lib/k8s-client'

export async function GET(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } }
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Parse query parameters
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') || '1')
    const limit = parseInt(url.searchParams.get('limit') || '50')
    const search = url.searchParams.get('search') || ''
    const sortBy = url.searchParams.get('sortBy') || 'name'
    const sortOrder = url.searchParams.get('sortOrder') || 'asc'
    const phases = url.searchParams.getAll('phase')

    let allAgents: any[] = []
    
    try {
      // Get all agents across all clusters for this organization
      const labelSelector = `langop.io/organization-id=${organization.id}`
      
      const response = await k8sClient.listLanguageAgents(organization.namespace, {
        labelSelector,
        limit: 1000 // Get a large number to filter locally
      })

      // Handle different response structures from k8s client
      allAgents = (response as any)?.items || (response as any)?.body?.items || (response as any)?.data?.items || []
    } catch (k8sError) {
      console.error('Error fetching agents from Kubernetes:', k8sError instanceof Error ? k8sError.message : String(k8sError))
      // Graceful degradation - return empty list instead of failing
      allAgents = []
    }
    
    let filteredAgents = allAgents

    // Apply search filter
    if (search) {
      const searchLower = search.toLowerCase()
      filteredAgents = filteredAgents.filter((agent: any) =>
        agent.metadata?.name?.toLowerCase().includes(searchLower) ||
        agent.spec?.description?.toLowerCase().includes(searchLower)
      )
    }

    // Apply phase filter
    if (phases.length > 0) {
      filteredAgents = filteredAgents.filter((agent: any) =>
        phases.includes(agent.status?.phase || 'Unknown')
      )
    }

    // Sort
    filteredAgents.sort((a: any, b: any) => {
      let aVal, bVal
      
      switch (sortBy) {
        case 'name':
          aVal = a.metadata?.name || ''
          bVal = b.metadata?.name || ''
          break
        case 'phase':
          aVal = a.status?.phase || 'Unknown'
          bVal = b.status?.phase || 'Unknown'
          break
        case 'created':
          aVal = new Date(a.metadata?.creationTimestamp || 0)
          bVal = new Date(b.metadata?.creationTimestamp || 0)
          break
        default:
          aVal = a.metadata?.name || ''
          bVal = b.metadata?.name || ''
      }

      if (sortOrder === 'desc') {
        return aVal > bVal ? -1 : aVal < bVal ? 1 : 0
      } else {
        return aVal < bVal ? -1 : aVal > bVal ? 1 : 0
      }
    })

    // Paginate
    const total = filteredAgents.length
    const startIndex = (page - 1) * limit
    const endIndex = startIndex + limit
    const paginatedAgents = filteredAgents.slice(startIndex, endIndex)

    return NextResponse.json({
      data: paginatedAgents,
      pagination: {
        page,
        limit,
        total,
        totalPages: Math.ceil(total / limit)
      }
    })

  } catch (error) {
    console.error('Error fetching agents:', error)
    return NextResponse.json(
      { error: `Failed to fetch agents: ${error instanceof Error ? error.message : 'Unknown error'}` },
      { status: 500 }
    )
  }
}