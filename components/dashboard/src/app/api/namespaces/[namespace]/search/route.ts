import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { safeValidateNamespaceSearch } from '@/lib/validation'

// GET /api/namespaces/[namespace]/search - Search all resources in namespace
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
    const searchParams = {
      q: url.searchParams.get('q') || '',
      type: url.searchParams.get('type') as any,
      limit: parseInt(url.searchParams.get('limit') || '20'),
      organization: organization.id,
    }

    // Skip validation if query is empty (list all)
    if (searchParams.q) {
      const validation = safeValidateNamespaceSearch(searchParams)
      if (!validation.success) {
        return NextResponse.json({ 
          error: 'Invalid search parameters', 
          details: validation.error?.issues || 'Validation failed'
        }, { status: 400 })
      }
    }

    const query = searchParams.q
    const resourceType = searchParams.type
    const phase = url.searchParams.get('phase')
    const createdBy = url.searchParams.get('createdBy')
    const limit = searchParams.limit


    let results: any[] = []

    if (createdBy) {
      // Search by creator
      if (resourceType && ['agents', 'models', 'tools', 'personas', 'clusters'].includes(resourceType)) {
        const response = await k8sClient.listByCreator(
          resourceType as any, 
          resolvedParams.namespace, 
          createdBy
        )
        const items = (response.body as any)?.items || []
        results = items.map((item: any) => ({
          type: resourceType,
          name: item.metadata.name,
          namespace: item.metadata.namespace,
          resource: item,
        }))
      } else {
        // Search all resource types by creator
        const promises = ['agents', 'models', 'tools', 'personas', 'clusters'].map(type =>
          k8sClient.listByCreator(type as any, resolvedParams.namespace, createdBy)
        )
        const responses = await Promise.all(promises)
        responses.forEach((response, index) => {
          const items = (response.body as any)?.items || []
          const type = ['agents', 'models', 'tools', 'personas', 'clusters'][index]
          items.forEach((item: any) => {
            results.push({
              type,
              name: item.metadata.name,
              namespace: item.metadata.namespace,
              resource: item,
            })
          })
        })
      }
    } else if (phase) {
      // Search by phase
      if (resourceType && ['agents', 'models', 'tools', 'personas', 'clusters'].includes(resourceType)) {
        const response = await k8sClient.listByPhase(
          resourceType as any, 
          resolvedParams.namespace, 
          phase
        )
        const items = (response.body as any)?.items || []
        results = items.map((item: any) => ({
          type: resourceType,
          name: item.metadata.name,
          namespace: item.metadata.namespace,
          resource: item,
        }))
      } else {
        // Search all resource types by phase
        const promises = ['agents', 'models', 'tools', 'personas', 'clusters'].map(type =>
          k8sClient.listByPhase(type as any, resolvedParams.namespace, phase)
        )
        const responses = await Promise.all(promises)
        responses.forEach((response, index) => {
          const items = (response.body as any)?.items || []
          const type = ['agents', 'models', 'tools', 'personas', 'clusters'][index]
          items.forEach((item: any) => {
            results.push({
              type,
              name: item.metadata.name,
              namespace: item.metadata.namespace,
              resource: item,
            })
          })
        })
      }
    } else if (query) {
      // General text search
      results = await k8sClient.searchResources(resolvedParams.namespace, query, organization.id)
      
      // Filter by resource type if specified
      if (resourceType) {
        results = results.filter(result => result.type === resourceType)
      }
    } else {
      // List all resources in namespace
      results = await k8sClient.searchResources(resolvedParams.namespace, '', organization.id)
      
      // Filter by resource type if specified
      if (resourceType) {
        results = results.filter(result => result.type === resourceType)
      }
    }

    // Apply limit
    const limitedResults = results.slice(0, limit)

    // Group results by type for easier consumption
    const groupedResults = limitedResults.reduce((acc, result) => {
      if (!acc[result.type]) {
        acc[result.type] = []
      }
      acc[result.type].push(result.resource)
      return acc
    }, {} as Record<string, any[]>)

    return NextResponse.json({
      success: true,
      namespace: resolvedParams.namespace,
      query,
      total: results.length,
      returned: limitedResults.length,
      results: limitedResults,
      grouped: groupedResults,
      counts: {
        agents: groupedResults.agents?.length || 0,
        models: groupedResults.models?.length || 0,
        tools: groupedResults.tools?.length || 0,
        personas: groupedResults.personas?.length || 0,
        clusters: groupedResults.clusters?.length || 0,
      },
    })

  } catch (error) {
    console.error('Error searching namespace resources:', error)
    return NextResponse.json(
      { error: 'Failed to search namespace resources' },
      { status: 500 }
    )
  }
}