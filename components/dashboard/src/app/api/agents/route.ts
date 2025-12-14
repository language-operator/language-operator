import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageAgent, LanguageAgentListParams, LanguageAgentFormData } from '@/types/agent'
import { validateLanguageAgent, safeValidateLanguageAgent } from '@/lib/validation'
import { ZodError } from 'zod'

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
    console.log('Fetching agents from namespace:', organization.namespace)
    const response = await k8sClient.listLanguageAgents(organization.namespace)
    console.log('K8s response type:', typeof response)
    console.log('K8s response keys:', Object.keys(response))
    console.log('Full K8s response:', JSON.stringify(response, null, 2))
    
    // Handle different response structures from k8s client
    // Live K8s mode: { items: [...] } (direct response)
    // Legacy modes: { body: { items: [...] } } or { data: { items: [] } }
    let agents = []
    try {
      const rawItems = response.items || response.body?.items || response.data?.items || []
      agents = Array.isArray(rawItems) ? rawItems : []
      console.log('Direct response check - response type:', typeof response)
      console.log('Raw items type:', typeof rawItems)
      console.log('Raw items length:', rawItems?.length)
      console.log('Extracted agents count:', agents.length)
      if (agents.length > 0) {
        console.log('First agent structure:', JSON.stringify(agents[0], null, 2))
      } else {
        console.log('No agents found - debugging full response:', JSON.stringify(response, null, 2))
      }
    } catch (k8sError) {
      console.error('Error extracting agents from K8s response:', k8sError)
      agents = []
    }

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
        if (!params.executionMode.includes(agent.spec.executionMode || 'autonomous')) return false
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

    // Parse and validate request body
    const body = await request.json()
    
    // Validate required fields as per CRD specification
    if (!body.name || !body.selectedModels || !Array.isArray(body.selectedModels) || body.selectedModels.length === 0) {
      return NextResponse.json(
        { error: 'Missing required fields: name and at least one selectedModel' },
        { status: 400 }
      )
    }

    const formData: LanguageAgentFormData = body

    // Convert form data to LanguageAgent CRD matching the Go specification
    const agentData: LanguageAgent = {
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
        // Required: Container image for the agent
        image: 'ghcr.io/langop/language-agent:latest',
        
        // Required: Model references 
        modelRefs: (formData.selectedModels || []).map(modelName => ({ 
          name: modelName,
          namespace: organization.namespace 
        })),
        
        // Optional fields
        ...(formData.clusterRef && { clusterRef: formData.clusterRef }),
        ...(formData.instructions && { instructions: formData.instructions }),
        executionMode: formData.executionMode || 'autonomous',
        replicas: formData.replicas || 1,
        // Tool references using CRD structure
        ...(formData.selectedTools && formData.selectedTools.length > 0 && {
          toolRefs: formData.selectedTools.map(toolName => ({ 
            name: toolName,
            namespace: organization.namespace 
          })),
        }),
        
        // Persona references using CRD structure  
        ...(formData.selectedPersona && formData.selectedPersona !== 'none' && {
          personaRefs: [{ 
            name: formData.selectedPersona,
            namespace: organization.namespace 
          }],
        }),
      },
    }

    // Validate the agent CRD structure
    const validationResult = safeValidateLanguageAgent(agentData)
    if (!validationResult.success) {
      return NextResponse.json(
        { 
          error: 'Invalid agent configuration',
          details: validationResult.error.issues.map(err => ({
            path: err.path.join('.'),
            message: err.message
          }))
        },
        { status: 400 }
      )
    }

    const agent = validationResult.data

    // Create agent in Kubernetes
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