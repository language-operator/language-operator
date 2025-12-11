import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { getKubernetesClient } from '@/lib/kubernetes'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageCluster, LanguageClusterListParams, LanguageClusterFormData } from '@/types/cluster'

// GET /api/clusters - List all clusters for user's organization
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
    const params: LanguageClusterListParams = {
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
      domain: url.searchParams.get('domain') || undefined,
    }

    const k8sClient = getKubernetesClient()
    const response = await k8sClient.listLanguageClusters(organization.namespace)
    const clusters = (response.data as any)?.items || []

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

    // Sort and paginate
    const startIndex = ((params.page || 1) - 1) * (params.limit || 50)
    const endIndex = startIndex + (params.limit || 50)
    const paginatedClusters = filteredClusters.slice(startIndex, endIndex)

    return NextResponse.json({
      success: true,
      data: paginatedClusters,
      total: filteredClusters.length,
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

    const formData: LanguageClusterFormData = await request.json()

    const cluster: LanguageCluster = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageCluster',
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

    const k8sClient = getKubernetesClient()
    const response = await k8sClient.createLanguageCluster(organization.namespace, cluster)
    
    console.log(`User ${user.email} created LanguageCluster ${formData.name} in organization ${organization.name}`)

    return NextResponse.json({
      success: true,
      data: response.data,
    })

  } catch (error) {
    console.error('Error creating cluster:', error)
    return NextResponse.json({ error: 'Failed to create cluster' }, { status: 500 })
  }
}