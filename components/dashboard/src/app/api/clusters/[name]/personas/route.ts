import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { filterByClusterRef } from '@/lib/cluster-utils'
import { LanguagePersona, LanguagePersonaListParams, LanguagePersonaFormData } from '@/types/persona'

// GET /api/clusters/[name]/personas - List personas for a specific cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName } = await params
    if (!clusterName) {
      return NextResponse.json({ error: 'Cluster name is required' }, { status: 400 })
    }

    const url = new URL(request.url)
    const queryParams: LanguagePersonaListParams = {
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      tone: url.searchParams.getAll('tone') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
    }

    // Fetch all personas from organization namespace
    const response = await k8sClient.listLanguagePersonas(organization.namespace)
    
    // Handle different response structures from k8s client
    const allPersonas = (response as any)?.items || (response.data as any)?.items || (response.body as any)?.items || []

    // Filter personas to only show those that belong to this specific cluster
    // Personas are cluster-scoped by having clusterRef set to the cluster name
    const clusterPersonas = filterByClusterRef(allPersonas, clusterName) as LanguagePersona[]

    // Apply search filtering 
    let filteredPersonas = clusterPersonas.filter((persona: LanguagePersona) => {
      if (queryParams.search) {
        const searchLower = queryParams.search.toLowerCase()
        const nameMatch = persona.metadata.name?.toLowerCase().includes(searchLower)
        const displayNameMatch = persona.spec.displayName?.toLowerCase().includes(searchLower)
        const descMatch = persona.spec.description?.toLowerCase().includes(searchLower)
        if (!nameMatch && !displayNameMatch && !descMatch) return false
      }
      
      if (queryParams.tone && queryParams.tone.length > 0) {
        if (!queryParams.tone.includes(persona.spec.tone || '')) return false
      }
      
      if (queryParams.phase && queryParams.phase.length > 0) {
        if (!queryParams.phase.includes(persona.status?.phase || '')) return false
      }
      
      return true
    })

    // Apply sorting
    filteredPersonas.sort((a: LanguagePersona, b: LanguagePersona) => {
      const order = queryParams.sortOrder === 'desc' ? -1 : 1
      
      switch (queryParams.sortBy) {
        case 'name':
          return (a.metadata.name || '').localeCompare(b.metadata.name || '') * order
        case 'displayName':
          return (a.spec.displayName || '').localeCompare(b.spec.displayName || '') * order
        case 'tone':
          return ((a.spec.tone || '').localeCompare(b.spec.tone || '')) * order
        case 'phase':
          return ((a.status?.phase || '').localeCompare(b.status?.phase || '')) * order
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
    const paginatedPersonas = filteredPersonas.slice(startIndex, endIndex)

    return NextResponse.json({
      success: true,
      data: paginatedPersonas,
      total: filteredPersonas.length,
      page: queryParams.page || 1,
      limit: queryParams.limit || 50,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching cluster personas:', error)
    return NextResponse.json({ 
      error: 'Failed to fetch personas for cluster',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}

// POST /api/clusters/[name]/personas - Create new persona for specific cluster
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName } = await params
    if (!clusterName) {
      return NextResponse.json({ error: 'Cluster name is required' }, { status: 400 })
    }

    const formData: LanguagePersonaFormData = await request.json()
    
    // Create persona with cluster reference
    const persona: LanguagePersona = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguagePersona',
      metadata: {
        name: formData.name,
        namespace: organization.namespace,
        labels: {
          'langop.io/organization-id': organization.id,
          'langop.io/cluster': clusterName,
        },
        annotations: {
          'langop.io/created-by-email': user.email,
          'langop.io/created-at': new Date().toISOString(),
        },
      },
      spec: {
        displayName: formData.displayName,
        description: formData.description,
        systemPrompt: formData.systemPrompt,
        clusterRef: clusterName,  // This makes it cluster-scoped
        ...(formData.tone && { tone: formData.tone }),
        ...(formData.language && { language: formData.language }),
        ...(formData.version && { version: formData.version }),
        ...(formData.capabilities && formData.capabilities.length > 0 && { capabilities: formData.capabilities }),
        ...(formData.limitations && formData.limitations.length > 0 && { limitations: formData.limitations }),
        ...(formData.instructions && formData.instructions.length > 0 && { instructions: formData.instructions }),
        ...(formData.examples && formData.examples.length > 0 && {
          examples: formData.examples.map(ex => ({
            input: ex.input,
            output: ex.output,
            ...(ex.context && { context: ex.context }),
            ...(ex.tags && { tags: ex.tags }),
          })),
        }),
      },
    }

    // Create the persona using k8s client
    const result = await k8sClient.createLanguagePersona(organization.namespace, persona)

    return NextResponse.json({
      success: true,
      data: result,
      message: `Persona "${formData.name}" created successfully in cluster "${clusterName}"`,
    })

  } catch (error) {
    console.error('Error creating persona:', error)
    return NextResponse.json({ 
      error: 'Failed to create persona',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}