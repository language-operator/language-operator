import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'

// GET /api/activity/recent - Get recent activity from Kubernetes events
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
    const limit = parseInt(url.searchParams.get('limit') || '10')

    // Fetch recent events from organization's namespace
    const eventsResponse = await k8sClient.listEvents(organization.namespace, {
      limit: Math.min(limit, 50), // Cap at 50 for performance
      labelSelector: `langop.io/organization-id=${organization.id}`,
      // Sort by creation timestamp descending (most recent first)
    })

    // Handle different response structures from k8s client
    const allEvents = (eventsResponse as any)?.body?.items || 
                     (eventsResponse as any)?.data?.items || 
                     (eventsResponse as any)?.items || 
                     []

    // Transform K8s events into activity format
    const activities = allEvents
      .filter((event: any) => {
        // Only show events related to Language Operator resources
        const involvedObject = event.involvedObject
        return involvedObject && 
               involvedObject.apiVersion === 'langop.io/v1alpha1' &&
               ['LanguageAgent', 'LanguageModel', 'LanguageTool', 'LanguagePersona', 'LanguageCluster'].includes(involvedObject.kind)
      })
      .sort((a: any, b: any) => {
        // Sort by last timestamp (most recent first)
        const aTime = new Date(a.lastTimestamp || a.firstTimestamp || a.metadata.creationTimestamp).getTime()
        const bTime = new Date(b.lastTimestamp || b.firstTimestamp || b.metadata.creationTimestamp).getTime()
        return bTime - aTime
      })
      .slice(0, limit) // Apply final limit
      .map((event: any) => {
        const involvedObject = event.involvedObject
        const timestamp = event.lastTimestamp || event.firstTimestamp || event.metadata.creationTimestamp
        
        // Map K8s event to user-friendly activity
        const resourceType = involvedObject.kind.replace('Language', '').toLowerCase()
        const resourceName = involvedObject.name
        const action = getActionFromEvent(event)
        const namespace = involvedObject.namespace || organization.namespace

        return {
          id: event.metadata.uid,
          type: resourceType, // 'agent', 'model', 'tool', 'persona', 'cluster'
          action, // 'created', 'updated', 'scaled', 'failed', 'ready'
          resourceName,
          namespace,
          message: formatActivityMessage(resourceType, resourceName, action, event),
          timestamp: new Date(timestamp).toISOString(),
          reason: event.reason,
          source: event.source?.component || 'kubernetes',
        }
      })

    return NextResponse.json({
      success: true,
      data: activities,
      total: activities.length,
      namespace: organization.namespace,
    })

  } catch (error) {
    console.error('Error fetching recent activity:', error)
    return NextResponse.json(
      { 
        error: 'Failed to fetch recent activity',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

// Helper function to determine action from K8s event
function getActionFromEvent(event: any): string {
  const reason = event.reason?.toLowerCase() || ''
  const type = event.type?.toLowerCase() || ''
  
  // Map common K8s event reasons to user-friendly actions
  if (reason.includes('created') || reason.includes('successfulcreate')) return 'created'
  if (reason.includes('updated') || reason.includes('update')) return 'updated'
  if (reason.includes('scaled') || reason.includes('scaling')) return 'scaled'
  if (reason.includes('failed') || type === 'warning') return 'failed'
  if (reason.includes('ready') || reason.includes('reconciled')) return 'ready'
  
  // Default mapping based on event type
  if (type === 'normal') return 'updated'
  
  return reason || 'updated'
}

// Helper function to format activity messages
function formatActivityMessage(resourceType: string, resourceName: string, action: string, event: any): string {
  const capitalizedType = resourceType.charAt(0).toUpperCase() + resourceType.slice(1)
  
  switch (action) {
    case 'created':
      return `${capitalizedType} "${resourceName}" created`
    case 'updated':
      return `${capitalizedType} "${resourceName}" configuration updated`
    case 'scaled':
      return `${capitalizedType} "${resourceName}" scaled`
    case 'failed':
      return `${capitalizedType} "${resourceName}" failed`
    case 'ready':
      return `${capitalizedType} "${resourceName}" is ready`
    default:
      return `${capitalizedType} "${resourceName}" ${action}`
  }
}