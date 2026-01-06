import { NextRequest, NextResponse } from 'next/server'
import { getOrganizationContext } from '@/lib/organization-utils'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { LanguageCluster, LanguageClusterListParams, LanguageClusterFormData } from '@/types/cluster'
import { 
  validateOrganizationContext, 
  createOrganizationErrorResponse,
  OrganizationNotFoundError,
  withOrganizationErrorHandler,
  createSuccessResponse
} from '@/lib/api-error-handler'

// GET /api/[org_id]/clusters - List all clusters for organization
export const GET = withOrganizationErrorHandler(async function(
  request: NextRequest,
  { params }: { params: Promise<{ org_id: string }> }
) {
  // Extract organization context from middleware
  const context = await getOrganizationContext()
  const resolvedParams = await params
  const urlOrgId = resolvedParams.org_id

  // Validate organization context with enhanced error handling
  validateOrganizationContext(urlOrgId, context?.organizationId, context?.userId)

  // Get organization details
  const organization = await db.organization.findUnique({
    where: { id: context!.organizationId },
    select: { id: true, name: true, namespace: true, plan: true }
  })

  if (!organization) {
    throw new OrganizationNotFoundError(context!.organizationId)
  }

    const url = new URL(request.url)
    const params_query: LanguageClusterListParams = {
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
      if (params_query.search) {
        const searchLower = params_query.search.toLowerCase()
        const nameMatch = cluster.metadata.name?.toLowerCase().includes(searchLower)
        const domainMatch = cluster.spec.domain?.toLowerCase().includes(searchLower)
        if (!nameMatch && !domainMatch) return false
      }
      
      if (params_query.phase && params_query.phase.length > 0) {
        if (!params_query.phase.includes(cluster.status?.phase || '')) return false
      }
      
      if (params_query.domain && cluster.spec.domain !== params_query.domain) return false
      
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
    const startIndex = ((params_query.page || 1) - 1) * (params_query.limit || 50)
    const endIndex = startIndex + (params_query.limit || 50)
    const paginatedClusters = clustersWithAgentCounts.slice(startIndex, endIndex)

  return createSuccessResponse(paginatedClusters, undefined, {
    total: clustersWithAgentCounts.length,
    page: params_query.page || 1,
    limit: params_query.limit || 50,
  })
})

// POST /api/[org_id]/clusters - Create a new cluster
export const POST = withOrganizationErrorHandler(async function(
  request: NextRequest,
  { params }: { params: Promise<{ org_id: string }> }
) {
  // Extract organization context from middleware
  const context = await getOrganizationContext()
  const resolvedParams = await params
  const urlOrgId = resolvedParams.org_id

  // Validate organization context with enhanced error handling
  validateOrganizationContext(urlOrgId, context?.organizationId, context?.userId)

  // Get organization details
  const organization = await db.organization.findUnique({
    where: { id: context!.organizationId },
    select: { id: true, name: true, namespace: true, plan: true }
  })

  if (!organization) {
    throw new OrganizationNotFoundError(context!.organizationId)
  }

    // Get user details
    const user = await db.user.findUnique({
      where: { id: context!.userId },
      select: { id: true, email: true }
    })

    if (!user || !user.email) {
      return NextResponse.json({ error: 'User not found' }, { status: 404 })
    }

    // Note: Permission check for 'create' is done in middleware

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
          'langop.io/created-by-email': user.email,
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

    return createSuccessResponse(response.data || response, `Cluster '${formData.name}' created successfully`)
})