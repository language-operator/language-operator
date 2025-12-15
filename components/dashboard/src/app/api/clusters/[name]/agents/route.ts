import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageAgent, LanguageAgentListParams } from '@/types/agent'

// GET /api/clusters/[name]/agents - List all agents for specific cluster
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
    const queryParams: LanguageAgentListParams = {
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
      executionMode: url.searchParams.getAll('executionMode') || undefined,
    }

    // Fetch all agents from organization namespace
    const response = await k8sClient.listLanguageAgents(organization.namespace)
    
    // Handle different response structures from k8s client
    const allAgents = (response as any)?.items || (response.data as any)?.items || (response.body as any)?.items || []

    // For now, show all agents from the organization namespace as available to any cluster
    // In the future, this can be refined to filter by actual cluster-agent relationships
    // when cluster-scoping is fully implemented in the CRD spec
    const clusterAgents = allAgents

    // Apply search filtering 
    let filteredAgents = clusterAgents.filter((agent: LanguageAgent) => {
      if (queryParams.search) {
        const searchLower = queryParams.search.toLowerCase()
        const nameMatch = agent.metadata.name?.toLowerCase().includes(searchLower)
        const instructionsMatch = agent.spec.instructions?.toLowerCase().includes(searchLower)
        const execModeMatch = agent.spec.executionMode?.toLowerCase().includes(searchLower)
        if (!nameMatch && !instructionsMatch && !execModeMatch) return false
      }
      
      if (queryParams.phase && queryParams.phase.length > 0) {
        if (!queryParams.phase.includes(agent.status?.phase || '')) return false
      }
      
      if (queryParams.executionMode && queryParams.executionMode.length > 0) {
        if (!queryParams.executionMode.includes(agent.spec.executionMode || '')) return false
      }
      
      return true
    })

    // Apply sorting
    filteredAgents.sort((a: LanguageAgent, b: LanguageAgent) => {
      const order = queryParams.sortOrder === 'desc' ? -1 : 1
      
      switch (queryParams.sortBy) {
        case 'name':
          return (a.metadata.name || '').localeCompare(b.metadata.name || '') * order
        case 'phase':
          return ((a.status?.phase || '').localeCompare(b.status?.phase || '')) * order
        case 'executions':
          const aExecs = a.status?.executionCount || 0
          const bExecs = b.status?.executionCount || 0
          return (bExecs - aExecs) * order
        case 'successRate':
          const aRate = parseFloat(a.status?.metrics?.successRate || '0')
          const bRate = parseFloat(b.status?.metrics?.successRate || '0')
          return (bRate - aRate) * order
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
    const paginatedAgents = filteredAgents.slice(startIndex, endIndex)

    return NextResponse.json({
      success: true,
      data: paginatedAgents,
      total: filteredAgents.length,
      page: queryParams.page || 1,
      limit: queryParams.limit || 50,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching cluster agents:', error)
    return NextResponse.json({ 
      error: 'Failed to fetch agents for cluster',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}

// POST /api/clusters/[name]/agents - Create new agent for specific cluster
export async function POST(
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
    
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName } = await params
    if (!clusterName) {
      return NextResponse.json({ error: 'Cluster name is required' }, { status: 400 })
    }

    const agentData = await request.json()
    
    // Transform LanguageAgentFormData to LanguageAgent CRD format
    const agentCrd: LanguageAgent = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageAgent',
      metadata: {
        name: agentData.name,
        namespace: organization.namespace,
        labels: {
          'langop.io/organization-id': organization.id,
        },
        annotations: {
          'langop.io/created-by-email': user.email,
        },
      },
      spec: {
        instructions: agentData.instructions,
        executionMode: agentData.executionMode || 'autonomous',
        replicas: agentData.replicas || 1,
        
        // Required fields based on existing agent structure
        image: 'ghcr.io/language-operator/agent:latest',
        imagePullPolicy: 'Always',
        clusterRef: clusterName,
        backoffLimit: 3,
        maxIterations: 50,
        timeout: '10m',
        restartPolicy: 'OnFailure',
        
        // Model references with namespace and role
        modelRefs: agentData.selectedModels?.map((name: string) => ({ 
          name,
          namespace: organization.namespace,
          role: 'primary' as const
        })) || [],
        
        // Tool references with namespace
        ...(agentData.selectedTools?.length > 0 && {
          toolRefs: agentData.selectedTools.map((name: string) => ({ 
            name,
            namespace: organization.namespace 
          })),
        }),
        
        // Persona references with namespace  
        ...(agentData.selectedPersona && agentData.selectedPersona !== 'none' && {
          personaRefs: [{ 
            name: agentData.selectedPersona,
            namespace: organization.namespace 
          }],
        }),
        
        // Default workspace configuration
        workspace: {
          enabled: true,
          size: '10Gi',
          accessMode: 'ReadWriteOnce',
          mountPath: '/workspace',
        },
        
        // Empty resources (let defaults apply)
        resources: {},
        
        // Agent version reference (auto-generated name)
        agentVersionRef: {
          name: `${agentData.name}-v1`,
          namespace: organization.namespace,
          lock: false,
        },
      },
    }

    // Create the agent using k8s client
    const result = await k8sClient.createLanguageAgent(organization.namespace, agentCrd)

    return NextResponse.json({
      success: true,
      data: result,
      message: `Agent "${agentData.name}" created successfully in cluster "${clusterName}"`,
    })

  } catch (error) {
    console.error('Error creating agent:', error)
    return NextResponse.json({ 
      error: 'Failed to create agent',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}