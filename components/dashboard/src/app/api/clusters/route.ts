import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { LanguageCluster, LanguageClusterListParams, LanguageClusterFormData } from '@/types/cluster'
import { safeValidateLanguageCluster } from '@/lib/validation'

// GET /api/clusters - List all clusters for user's organization
export async function GET(request: NextRequest) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const url = new URL(request.url)
    const params: LanguageClusterListParams = {
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
      domain: url.searchParams.get('domain') || undefined,
    }

    // Fetch clusters from Kubernetes with robust response handling
    let clusters: LanguageCluster[] = []
    
    try {
      console.log(`Fetching clusters from namespace: ${organization.namespace}`)
      const response = await k8sClient.listByOrganization('clusters', organization.namespace, organization.id)
      
      // Handle different response structures from k8s client
      // Live K8s mode: { body: { items: [...] } }
      // Error fallback: { data: { items: [] } }
      const responseBody = (response as any)?.body
      const responseData = (response as any)?.data
      const responseItems = (response as any)?.items
      
      if (responseBody?.items && Array.isArray(responseBody.items)) {
        clusters = responseBody.items
        console.log(`Found ${clusters.length} clusters from Kubernetes API`)
      } else if (responseData?.items && Array.isArray(responseData.items)) {
        clusters = responseData.items
        console.log(`Found ${clusters.length} clusters from response data`)
      } else if (responseItems && Array.isArray(responseItems)) {
        clusters = responseItems
        console.log(`Found ${clusters.length} clusters from direct items`)
      } else {
        console.warn('No clusters array found in response structure:', Object.keys(response))
        clusters = []
      }
    } catch (k8sError) {
      console.error('Error fetching clusters from Kubernetes:', k8sError instanceof Error ? k8sError.message : String(k8sError))
      // Graceful degradation - return empty list instead of failing
      clusters = []
    }

    // Apply filtering
    let filteredClusters = clusters.filter((cluster: LanguageCluster) => {
      if (params.search) {
        const searchLower = params.search.toLowerCase()
        const nameMatch = cluster.metadata.name?.toLowerCase().includes(searchLower)
        const domainMatch = cluster.spec.domain?.toLowerCase().includes(searchLower)
        if (!nameMatch && !domainMatch) return false
      }
      
      if (params.phase && params.phase.length > 0) {
        if (!params.phase.includes(cluster.status?.phase || '')) return false
      }
      
      if (params.domain && cluster.spec.domain !== params.domain) return false
      
      return true
    })

    // Fetch all agents once to avoid N+1 query problem
    let agentCountsByCluster: Record<string, number> = {}
    
    try {
      const agentsResponse = await k8sClient.listByOrganization('agents', organization.namespace, organization.id)
      
      // Handle different response structures from k8s client
      const allAgents = (agentsResponse as any)?.body?.items || 
                       (agentsResponse as any)?.data?.items || 
                       (agentsResponse as any)?.items || 
                       []
      
      // Group agents by cluster
      agentCountsByCluster = allAgents.reduce((acc: Record<string, number>, agent: any) => {
        const clusterRef = agent.spec?.clusterRef
        if (clusterRef) {
          acc[clusterRef] = (acc[clusterRef] || 0) + 1
        }
        return acc
      }, {})
      
      console.log(`Agent counts by cluster:`, agentCountsByCluster)
    } catch (error) {
      console.error('Error fetching agents for cluster counts:', error)
      // agentCountsByCluster remains empty object, all clusters get 0 count
    }

    // Add agent counts to clusters
    const clustersWithAgentCounts = filteredClusters.map((cluster: LanguageCluster) => {
      const agentCount = agentCountsByCluster[cluster.metadata.name || ''] || 0
      
      return {
        ...cluster,
        status: {
          ...cluster.status,
          agentCount
        }
      }
    })

    // Sort and paginate
    const startIndex = ((params.page || 1) - 1) * (params.limit || 50)
    const endIndex = startIndex + (params.limit || 50)
    const paginatedClusters = clustersWithAgentCounts.slice(startIndex, endIndex)

    return NextResponse.json({
      success: true,
      data: paginatedClusters,
      total: clustersWithAgentCounts.length,
      page: params.page || 1,
      limit: params.limit || 50,
    })

  } catch (error) {
    console.error('Error fetching clusters:', error)
    return NextResponse.json({ error: 'Failed to fetch clusters' }, { status: 500 })
  }
}

// POST /api/clusters - Create a new cluster
export async function POST(request: NextRequest) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const formData: LanguageClusterFormData = await request.json()

    const cluster: LanguageCluster = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageCluster',
      metadata: {
        name: formData.name,
        namespace: organization.namespace,
        labels: {
          'langop.io/organization-id': organization.id,
          'langop.io/created-by': user.id,
        },
        annotations: {
          'langop.io/created-by-email': user.email!,
          'langop.io/created-at': new Date().toISOString(),
        },
      },
      spec: {
        ...(formData.domain && { domain: formData.domain }),
        ...(formData.gatewayName || formData.ingressClassName || formData.enableTLS) && {
          ingressConfig: {
            ...(formData.gatewayName && { 
              gatewayName: formData.gatewayName,
              gatewayNamespace: formData.gatewayNamespace,
            }),
            ...(formData.ingressClassName && { ingressClassName: formData.ingressClassName }),
            ...(formData.enableTLS) && {
              tls: {
                enabled: true,
                ...(formData.tlsSecretName && { secretName: formData.tlsSecretName }),
                ...(formData.useCertManager && formData.issuerName) && {
                  issuerRef: {
                    name: formData.issuerName,
                    kind: formData.issuerKind || 'ClusterIssuer',
                    group: formData.issuerGroup || 'cert-manager.io',
                  },
                },
              },
            },
          },
        },
      },
    }

    const response = await k8sClient.createLanguageCluster(organization.namespace, cluster)
    
    console.log(`User ${user.email} created LanguageCluster ${formData.name} in organization ${organization.name}`)
    console.log('K8s API response structure:', JSON.stringify(response, null, 2))

    return NextResponse.json({
      success: true,
      data: response.data || response, // Handle different response structures
    })

  } catch (error) {
    console.error('Error creating cluster:', error)
    return NextResponse.json({ error: 'Failed to create cluster' }, { status: 500 })
  }
}