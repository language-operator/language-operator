import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { getKubernetesClient } from '@/lib/kubernetes'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageTool, LanguageToolListParams, LanguageToolFormData } from '@/types/tool'
import { safeValidateLanguageTool } from '@/lib/validation'

// GET /api/tools - List all tools for user's organization
export async function GET(request: NextRequest) {
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

    const url = new URL(request.url)
    const params: LanguageToolListParams = {
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      type: url.searchParams.getAll('type') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
    }

    const k8sClient = getKubernetesClient()
    const response = await k8sClient.listLanguageTools(organization.namespace)
    const tools = (response.data as any)?.items || []

    // Apply filtering and sorting (similar to agents pattern)
    let filteredTools = tools.filter((tool: LanguageTool) => {
      if (params.search) {
        const searchLower = params.search.toLowerCase()
        const nameMatch = tool.metadata.name?.toLowerCase().includes(searchLower)
        const typeMatch = tool.spec.type?.toLowerCase().includes(searchLower)
        if (!nameMatch && !typeMatch) return false
      }
      
      if (params.type && params.type.length > 0) {
        if (!params.type.includes(tool.spec.type)) return false
      }
      
      if (params.phase && params.phase.length > 0) {
        if (!params.phase.includes(tool.status?.phase || '')) return false
      }
      
      return true
    })

    // Sort and paginate
    const startIndex = ((params.page || 1) - 1) * (params.limit || 50)
    const endIndex = startIndex + (params.limit || 50)
    const paginatedTools = filteredTools.slice(startIndex, endIndex)

    return NextResponse.json({
      success: true,
      data: paginatedTools,
      total: filteredTools.length,
      page: params.page || 1,
      limit: params.limit || 50,
    })

  } catch (error) {
    console.error('Error fetching tools:', error)
    return NextResponse.json({ error: 'Failed to fetch tools' }, { status: 500 })
  }
}

// POST /api/tools - Create a new tool
export async function POST(request: NextRequest) {
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
    
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const formData: LanguageToolFormData = await request.json()

    const tool: LanguageTool = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageTool',
      metadata: {
        name: formData.name,
        namespace: organization.namespace,
        labels: {
          'langop.io/organization': organization.id,
          'langop.io/created-by': user.id,
        },
        annotations: {
          'langop.io/created-by-email': user.email!,
          'langop.io/created-at': new Date().toISOString(),
        },
      },
      spec: {
        type: formData.type,
        ...(formData.image && { image: formData.image }),
        ...(formData.toolName && {
          schema: {
            name: formData.toolName,
            description: formData.description,
            version: formData.version,
          },
        }),
        ...(formData.replicas && {
          scaling: {
            replicas: formData.replicas,
            minReplicas: formData.minReplicas,
            maxReplicas: formData.maxReplicas,
          },
        }),
      },
    }

    const k8sClient = getKubernetesClient()
    const response = await k8sClient.createLanguageTool(organization.namespace, tool)
    
    console.log(`User ${user.email} created LanguageTool ${formData.name} in organization ${organization.name}`)

    return NextResponse.json({
      success: true,
      data: response.data,
    })

  } catch (error) {
    console.error('Error creating tool:', error)
    return NextResponse.json({ error: 'Failed to create tool' }, { status: 500 })
  }
}