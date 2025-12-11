import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { getKubernetesClient } from '@/lib/kubernetes'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageAgent, LanguageAgentListParams, LanguageAgentFormData } from '@/types/agent'

// GET /api/agents - List all agents for user's organization
export async function GET(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Get user's current organization
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { 
        memberships: { 
          include: { organization: true } 
        } 
      },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    // Use the first organization (in a real app, you'd have org selection logic)
    const organization = user.memberships[0].organization
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Parse query parameters
    const url = new URL(request.url)
    const params: LanguageAgentListParams = {
      namespace: organization.namespace,
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
      executionMode: url.searchParams.getAll('executionMode') || undefined,
    }

    // Fetch agents from Kubernetes
    const k8sClient = getKubernetesClient()
    const response = await k8sClient.listLanguageAgents(organization.namespace)
    const agents = (response.data as any)?.items || []

    // Apply client-side filtering (in production, you'd want server-side filtering)
    let filteredAgents = agents.filter((agent: LanguageAgent) => {
      // Search filter
      if (params.search) {
        const searchLower = params.search.toLowerCase()
        const nameMatch = agent.metadata.name?.toLowerCase().includes(searchLower)
        const namespaceMatch = agent.metadata.namespace?.toLowerCase().includes(searchLower)
        if (!nameMatch && !namespaceMatch) return false
      }

      // Phase filter
      if (params.phase && params.phase.length > 0) {
        if (!params.phase.includes(agent.status?.phase || '')) return false
      }

      // Execution mode filter
      if (params.executionMode && params.executionMode.length > 0) {
        if (!params.executionMode.includes(agent.spec.executionMode)) return false
      }

      return true
    })

    // Sort agents
    filteredAgents.sort((a: LanguageAgent, b: LanguageAgent) => {
      let aValue: any, bValue: any
      
      switch (params.sortBy) {
        case 'name':
          aValue = a.metadata.name || ''
          bValue = b.metadata.name || ''
          break
        case 'namespace':
          aValue = a.metadata.namespace || ''
          bValue = b.metadata.namespace || ''
          break
        case 'phase':
          aValue = a.status?.phase || ''
          bValue = b.status?.phase || ''
          break
        case 'age':
          aValue = new Date(a.metadata.creationTimestamp || 0).getTime()
          bValue = new Date(b.metadata.creationTimestamp || 0).getTime()
          break
        case 'executions':
          aValue = a.status?.executionCount || 0
          bValue = b.status?.executionCount || 0
          break
        case 'successRate':
          aValue = parseFloat(a.status?.metrics?.successRate || '0')
          bValue = parseFloat(b.status?.metrics?.successRate || '0')
          break
        default:
          aValue = a.metadata.name || ''
          bValue = b.metadata.name || ''
      }

      if (params.sortOrder === 'desc') {
        return aValue < bValue ? 1 : aValue > bValue ? -1 : 0
      }
      return aValue > bValue ? 1 : aValue < bValue ? -1 : 0
    })

    // Pagination
    const startIndex = ((params.page || 1) - 1) * (params.limit || 50)
    const endIndex = startIndex + (params.limit || 50)
    const paginatedAgents = filteredAgents.slice(startIndex, endIndex)

    return NextResponse.json({
      success: true,
      data: paginatedAgents,
      total: filteredAgents.length,
      page: params.page || 1,
      limit: params.limit || 50,
    })

  } catch (error) {
    console.error('Error fetching agents:', error)
    return NextResponse.json(
      { error: 'Failed to fetch agents' },
      { status: 500 }
    )
  }
}

// POST /api/agents - Create a new agent
export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Get user's current organization
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { 
        memberships: { 
          include: { organization: true } 
        } 
      },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Parse request body
    const formData: LanguageAgentFormData = await request.json()

    // Convert form data to LanguageAgent CRD
    const agent: LanguageAgent = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageAgent',
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
        executionMode: formData.executionMode,
        replicas: formData.replicas,
        model: {
          name: formData.modelName,
          provider: formData.modelProvider,
          endpoint: formData.modelEndpoint,
          parameters: formData.modelParameters,
        },
        ...(formData.personaName && {
          persona: {
            name: formData.personaName,
            tone: formData.personaTone,
            instructions: formData.personaInstructions,
          },
        }),
        ...(formData.selectedTools.length > 0 && {
          tools: formData.selectedTools.map(toolName => ({ name: toolName })),
        }),
        ...(formData.cpuRequest || formData.memoryRequest || formData.cpuLimit || formData.memoryLimit) && {
          resources: {
            ...(formData.cpuRequest || formData.memoryRequest) && {
              requests: {
                ...(formData.cpuRequest && { cpu: formData.cpuRequest }),
                ...(formData.memoryRequest && { memory: formData.memoryRequest }),
              },
            },
            ...(formData.cpuLimit || formData.memoryLimit) && {
              limits: {
                ...(formData.cpuLimit && { cpu: formData.cpuLimit }),
                ...(formData.memoryLimit && { memory: formData.memoryLimit }),
              },
            },
          },
        },
        ...(formData.minReplicas || formData.maxReplicas || formData.targetCPUUtilization) && {
          scaling: {
            minReplicas: formData.minReplicas,
            maxReplicas: formData.maxReplicas,
            targetCPUUtilization: formData.targetCPUUtilization,
          },
        },
        ...(formData.nodeSelector && Object.keys(formData.nodeSelector).length > 0) && {
          nodeSelector: formData.nodeSelector,
        },
        ...(formData.tolerations && formData.tolerations.length > 0) && {
          tolerations: formData.tolerations,
        },
        ...(formData.enableIngress) && {
          networking: {
            ingress: {
              enabled: true,
              host: formData.ingressHost,
              path: formData.ingressPath || '/',
              tls: formData.enableTLS,
            },
          },
        },
      },
    }

    // Create agent in Kubernetes
    const k8sClient = getKubernetesClient()
    const response = await k8sClient.createLanguageAgent(organization.namespace, agent)
    const createdAgent = response.data as LanguageAgent

    // Audit log
    console.log(`User ${user.email} created LanguageAgent ${formData.name} in organization ${organization.name}`)

    return NextResponse.json({
      success: true,
      data: createdAgent,
    })

  } catch (error) {
    console.error('Error creating agent:', error)
    return NextResponse.json(
      { error: 'Failed to create agent' },
      { status: 500 }
    )
  }
}