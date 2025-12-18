import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { safeValidateNamespaceStats } from '@/lib/validation'

// GET /api/namespaces/[namespace]/stats - Get namespace resource statistics
export async function GET(request: NextRequest, { params }: { params: Promise<{ namespace: string }> }) {
  try {
    const resolvedParams = await params
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Validate namespace access
    if (resolvedParams.namespace !== organization.namespace) {
      return NextResponse.json({ error: 'Access denied to namespace' }, { status: 403 })
    }

    // Parse and validate query parameters
    const url = new URL(request.url)
    const statsParams = {
      timeRange: (url.searchParams.get('timeRange') as any) || '24h',
      granularity: (url.searchParams.get('granularity') as any) || 'hour',
      includeDetails: url.searchParams.get('includeDetails') === 'true',
    }

    const validation = safeValidateNamespaceStats(statsParams)
    if (!validation.success) {
      return NextResponse.json({ 
        error: 'Invalid stats parameters', 
        details: validation.error?.issues || 'Validation failed'
      }, { status: 400 })
    }


    // Get resource counts
    const resourceCounts = await k8sClient.getNamespaceResourceCounts(resolvedParams.namespace, organization.id)

    // Get detailed statistics for each resource type
    const [agentsResponse, modelsResponse, toolsResponse, personasResponse, clustersResponse] = await Promise.all([
      k8sClient.listByOrganization('agents', resolvedParams.namespace, organization.id),
      k8sClient.listByOrganization('models', resolvedParams.namespace, organization.id),
      k8sClient.listByOrganization('tools', resolvedParams.namespace, organization.id),
      k8sClient.listByOrganization('personas', resolvedParams.namespace, organization.id),
      k8sClient.listByOrganization('clusters', resolvedParams.namespace, organization.id),
    ])

    const agents = (agentsResponse.body as any)?.items || []
    const models = (modelsResponse.body as any)?.items || []
    const tools = (toolsResponse.body as any)?.items || []
    const personas = (personasResponse.body as any)?.items || []
    const clusters = (clustersResponse.body as any)?.items || []

    // Calculate phase statistics
    const agentPhases = agents.reduce((acc: any, agent: any) => {
      const phase = agent.status?.phase || 'Unknown'
      acc[phase] = (acc[phase] || 0) + 1
      return acc
    }, {})

    const modelPhases = models.reduce((acc: any, model: any) => {
      const phase = model.status?.phase || 'Unknown'
      acc[phase] = (acc[phase] || 0) + 1
      return acc
    }, {})

    const toolPhases = tools.reduce((acc: any, tool: any) => {
      const phase = tool.status?.phase || 'Unknown'
      acc[phase] = (acc[phase] || 0) + 1
      return acc
    }, {})

    const personaPhases = personas.reduce((acc: any, persona: any) => {
      const phase = persona.status?.phase || 'Available'
      acc[phase] = (acc[phase] || 0) + 1
      return acc
    }, {})

    const clusterPhases = clusters.reduce((acc: any, cluster: any) => {
      const phase = cluster.status?.phase || 'Unknown'
      acc[phase] = (acc[phase] || 0) + 1
      return acc
    }, {})

    // Calculate provider distribution for models
    const modelProviders = models.reduce((acc: any, model: any) => {
      const provider = model.spec?.provider || 'Unknown'
      acc[provider] = (acc[provider] || 0) + 1
      return acc
    }, {})

    // Calculate tool types distribution
    const toolTypes = tools.reduce((acc: any, tool: any) => {
      const type = tool.spec?.type || 'Unknown'
      acc[type] = (acc[type] || 0) + 1
      return acc
    }, {})

    // Calculate cluster health
    const healthyClusters = clusters.filter((cluster: any) => 
      cluster.status?.phase === 'Ready' && cluster.status?.ingress?.ready === true
    ).length

    // Calculate total replicas across agents
    const totalReplicas = agents.reduce((sum: number, agent: any) => {
      return sum + (agent.status?.replicas?.ready || 0)
    }, 0)

    // Get creation timeline (last 30 days)
    const thirtyDaysAgo = new Date()
    thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)

    const recentResources = [
      ...agents.filter((a: any) => new Date(a.metadata.creationTimestamp) > thirtyDaysAgo),
      ...models.filter((m: any) => new Date(m.metadata.creationTimestamp) > thirtyDaysAgo),
      ...tools.filter((t: any) => new Date(t.metadata.creationTimestamp) > thirtyDaysAgo),
      ...personas.filter((p: any) => new Date(p.metadata.creationTimestamp) > thirtyDaysAgo),
      ...clusters.filter((c: any) => new Date(c.metadata.creationTimestamp) > thirtyDaysAgo),
    ]

    const timeline = recentResources.reduce((acc: any, resource: any) => {
      const date = new Date(resource.metadata.creationTimestamp).toISOString().split('T')[0]
      acc[date] = (acc[date] || 0) + 1
      return acc
    }, {})

    return NextResponse.json({
      success: true,
      namespace: resolvedParams.namespace,
      organization: organization.name,
      timestamp: new Date().toISOString(),
      summary: resourceCounts,
      details: {
        agents: {
          total: resourceCounts.agents,
          phases: agentPhases,
          totalReplicas,
        },
        models: {
          total: resourceCounts.models,
          phases: modelPhases,
          providers: modelProviders,
        },
        tools: {
          total: resourceCounts.tools,
          phases: toolPhases,
          types: toolTypes,
        },
        personas: {
          total: resourceCounts.personas,
          phases: personaPhases,
        },
        clusters: {
          total: resourceCounts.clusters,
          phases: clusterPhases,
          healthy: healthyClusters,
        },
      },
      timeline,
      health: {
        overall: Math.round(
          ((agentPhases.Ready || 0) + 
           (modelPhases.Available || 0) + 
           (toolPhases.Available || 0) + 
           (personaPhases.Available || 0) + 
           (clusterPhases.Ready || 0)) / 
          Math.max(resourceCounts.agents + resourceCounts.models + resourceCounts.tools + resourceCounts.personas + resourceCounts.clusters, 1) * 100
        ),
        agents: agentPhases.Ready || 0,
        models: modelPhases.Available || 0,
        tools: toolPhases.Available || 0,
        personas: personaPhases.Available || 0,
        clusters: clusterPhases.Ready || 0,
      },
    })

  } catch (error) {
    console.error('Error fetching namespace stats:', error)
    return NextResponse.json(
      { error: 'Failed to fetch namespace statistics' },
      { status: 500 }
    )
  }
}