import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

// GET /api/clusters/[name]/counts - Get resource counts for a specific cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: clusterName } = await params

    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Note: Cluster validation removed - let the Kubernetes API handle cluster existence
    // The main cluster dashboard already validates cluster access properly

    // Get cluster-specific resource counts from Kubernetes
    console.log(`Getting resource counts for cluster ${clusterName} in namespace ${organization.namespace}`)
    
    try {
      // Get all resources in the namespace
      const counts = await k8sClient.getNamespaceResourceCounts(
        organization.namespace,
        organization.id
      )

      // Filter counts to only include resources that belong to this cluster
      const clusterCounts = await getClusterSpecificCounts(
        organization.namespace, 
        clusterName, 
        counts
      )

      return NextResponse.json({
        success: true,
        data: clusterCounts,
      })

    } catch (k8sError) {
      console.error('Error fetching cluster resource counts:', k8sError)
      return NextResponse.json(
        { error: 'Failed to fetch cluster resource counts' },
        { status: 500 }
      )
    }

  } catch (error) {
    console.error('Error in cluster counts endpoint:', error)
    return NextResponse.json(
      { error: 'Failed to fetch cluster counts' },
      { status: 500 }
    )
  }
}

async function getClusterSpecificCounts(namespace: string, clusterName: string, allCounts: any) {
  try {
    // For models, agents, tools, and personas - count only those that belong to this cluster
    // Resources belong to a cluster if they have the 'langop.io/cluster' label set to the cluster name
    // If no cluster label is present, they're available to all clusters
    
    const clusterCounts = {
      models: 0,
      agents: 0,
      tools: 0,
      personas: 0,
      clusters: allCounts.clusters || 0, // Clusters are not cluster-scoped
    }

    // Count models for this cluster
    try {
      const modelsResponse = await k8sClient.listLanguageModels(namespace)
      let models = []
      
      if (modelsResponse.body && typeof modelsResponse.body === 'object') {
        models = (modelsResponse.body as any)?.items || []
      } else if (modelsResponse.data && typeof modelsResponse.data === 'object') {
        models = (modelsResponse.data as any)?.items || []
      } else if (Array.isArray(modelsResponse)) {
        models = modelsResponse
      } else if ((modelsResponse as any)?.items) {
        models = (modelsResponse as any).items
      }

      clusterCounts.models = models.filter((model: any) => {
        // Check for clusterRef in spec (preferred) or fallback to label
        if (model.spec?.clusterRef) {
          return model.spec.clusterRef === clusterName
        }
        const clusterLabel = model.metadata?.labels?.['langop.io/cluster']
        return clusterLabel === clusterName
      }).length

    } catch (modelsError) {
      console.error('Error counting models:', modelsError)
    }

    // Count agents for this cluster
    try {
      const agentsResponse = await k8sClient.listLanguageAgents(namespace)
      let agents = []
      
      if (agentsResponse.body && typeof agentsResponse.body === 'object') {
        agents = (agentsResponse.body as any)?.items || []
      } else if (agentsResponse.data && typeof agentsResponse.data === 'object') {
        agents = (agentsResponse.data as any)?.items || []
      } else if (Array.isArray(agentsResponse)) {
        agents = agentsResponse
      } else if ((agentsResponse as any)?.items) {
        agents = (agentsResponse as any).items
      }

      clusterCounts.agents = agents.filter((agent: any) => {
        return agent.spec?.clusterRef === clusterName
      }).length

    } catch (agentsError) {
      console.error('Error counting agents:', agentsError)
    }

    // Count tools for this cluster
    try {
      const toolsResponse = await k8sClient.listLanguageTools(namespace)
      let tools = []
      
      if (toolsResponse.body && typeof toolsResponse.body === 'object') {
        tools = (toolsResponse.body as any)?.items || []
      } else if (toolsResponse.data && typeof toolsResponse.data === 'object') {
        tools = (toolsResponse.data as any)?.items || []
      } else if (Array.isArray(toolsResponse)) {
        tools = toolsResponse
      } else if ((toolsResponse as any)?.items) {
        tools = (toolsResponse as any).items
      }

      clusterCounts.tools = tools.filter((tool: any) => {
        return tool.spec?.clusterRef === clusterName
      }).length

    } catch (toolsError) {
      console.error('Error counting tools:', toolsError)
    }

    // Count personas for this cluster
    try {
      const personasResponse = await k8sClient.listLanguagePersonas(namespace)
      let personas = []
      
      if (personasResponse.body && typeof personasResponse.body === 'object') {
        personas = (personasResponse.body as any)?.items || []
      } else if (personasResponse.data && typeof personasResponse.data === 'object') {
        personas = (personasResponse.data as any)?.items || []
      } else if (Array.isArray(personasResponse)) {
        personas = personasResponse
      } else if ((personasResponse as any)?.items) {
        personas = (personasResponse as any).items
      }

      clusterCounts.personas = personas.filter((persona: any) => {
        return persona.spec?.clusterRef === clusterName
      }).length

    } catch (personasError) {
      console.error('Error counting personas:', personasError)
    }

    console.log(`Cluster ${clusterName} counts:`, clusterCounts)
    return clusterCounts

  } catch (error) {
    console.error('Error calculating cluster-specific counts:', error)
    throw error
  }
}