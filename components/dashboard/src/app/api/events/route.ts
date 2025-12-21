import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

// GET /api/events - Get Kubernetes events with optional filtering
export async function GET(request: NextRequest) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Parse query parameters
    const url = new URL(request.url)
    const limit = parseInt(url.searchParams.get('limit') || '20')
    const clusterName = url.searchParams.get('clusterName')
    const namespace = url.searchParams.get('namespace') || organization.namespace
    const resourceType = url.searchParams.get('resourceType')
    const resourceName = url.searchParams.get('resourceName')

    // Build field selector for filtering events
    let fieldSelector = ''
    const fieldSelectors: string[] = []

    // Filter by involved object if resource specified
    if (resourceName && resourceType) {
      const k8sKind = mapResourceTypeToK8sKind(resourceType)
      fieldSelectors.push(`involvedObject.kind=${k8sKind}`)
      fieldSelectors.push(`involvedObject.name=${resourceName}`)
    } else if (resourceType) {
      const k8sKind = mapResourceTypeToK8sKind(resourceType)
      fieldSelectors.push(`involvedObject.kind=${k8sKind}`)
    }

    if (fieldSelectors.length > 0) {
      fieldSelector = fieldSelectors.join(',')
    }

    // Determine target namespace for event fetching
    let targetNamespace = namespace

    // If cluster name is specified, try to find the cluster's namespace
    if (clusterName && !namespace) {
      try {
        const clusterResponse = await k8sClient.getLanguageCluster(organization.namespace, clusterName)
        const cluster = (clusterResponse as any)?.body || (clusterResponse as any)?.data || clusterResponse
        targetNamespace = cluster?.metadata?.namespace || organization.namespace
      } catch (error) {
        console.warn(`Could not find cluster ${clusterName}, using organization namespace`)
        targetNamespace = organization.namespace
      }
    }

    // Fetch events from Kubernetes
    // Note: We don't use labelSelector because Events don't inherit labels from involved resources
    const eventsResponse = await k8sClient.listEvents(targetNamespace, {
      limit: Math.min(limit * 2, 100), // Fetch more than requested to account for filtering
      fieldSelector: fieldSelector || undefined,
    })

    // Handle different response structures from k8s client
    const allEvents = (eventsResponse as any)?.body?.items || 
                     (eventsResponse as any)?.data?.items || 
                     (eventsResponse as any)?.items || 
                     []

    // Transform and filter K8s events
    let filteredEvents = allEvents
      .filter((event: any) => {
        const involvedObject = event.involvedObject
        if (!involvedObject) return false

        // Always include Language Operator resources
        if (involvedObject.apiVersion === 'langop.io/v1alpha1') {
          return true
        }

        // Include core K8s resources that are related to Language Operator
        const coreResourceTypes = ['Pod', 'Service', 'Deployment', 'ReplicaSet', 'Job', 'CronJob']
        if (involvedObject.apiVersion?.startsWith('v1') || involvedObject.apiVersion?.startsWith('apps/')) {
          if (coreResourceTypes.includes(involvedObject.kind)) {
            // Check if the resource has Language Operator labels
            const labels = event.involvedObject?.labels || {}
            return labels['langop.io/managed-by'] || labels['langop.io/organization-id']
          }
        }

        return false
      })
      .sort((a: any, b: any) => {
        // Sort by last timestamp (most recent first)
        const aTime = new Date(a.lastTimestamp || a.firstTimestamp || a.metadata.creationTimestamp).getTime()
        const bTime = new Date(b.lastTimestamp || b.firstTimestamp || b.metadata.creationTimestamp).getTime()
        return bTime - aTime
      })

    // Apply cluster filtering if specified
    if (clusterName) {
      // For cluster filtering, we need to fetch resources and check their spec.clusterRef
      // Build a set of resource names that belong to this cluster
      const clusterResourceNames = new Set<string>()

      try {
        // Fetch all Language Operator resources in the namespace
        const [agentsResp, modelsResp, toolsResp, personasResp] = await Promise.all([
          k8sClient.listLanguageAgents(targetNamespace).catch(() => ({ items: [] })),
          k8sClient.listLanguageModels(targetNamespace).catch(() => ({ items: [] })),
          k8sClient.listLanguageTools(targetNamespace).catch(() => ({ items: [] })),
          k8sClient.listLanguagePersonas(targetNamespace).catch(() => ({ items: [] })),
        ])

        // Extract items from responses (handle different response structures)
        const agents = (agentsResp as any)?.body?.items || (agentsResp as any)?.data?.items || (agentsResp as any)?.items || []
        const models = (modelsResp as any)?.body?.items || (modelsResp as any)?.data?.items || (modelsResp as any)?.items || []
        const tools = (toolsResp as any)?.body?.items || (toolsResp as any)?.data?.items || (toolsResp as any)?.items || []
        const personas = (personasResp as any)?.body?.items || (personasResp as any)?.data?.items || (personasResp as any)?.items || []

        // Add resource names that belong to the cluster
        const allResources = [...agents, ...models, ...tools, ...personas]
        allResources.forEach((resource: any) => {
          if (resource.spec?.clusterRef === clusterName) {
            clusterResourceNames.add(resource.metadata?.name)
          }
        })

        // Also add the cluster itself
        clusterResourceNames.add(clusterName)
      } catch (error) {
        console.error('Error fetching resources for cluster filtering:', error)
      }

      // Filter events to only include resources in this cluster
      filteredEvents = filteredEvents.filter((event: any) => {
        const involvedObject = event.involvedObject

        // Check if the involved object is in our cluster resources set
        if (involvedObject.apiVersion === 'langop.io/v1alpha1') {
          return clusterResourceNames.has(involvedObject.name)
        }

        // For core K8s resources (Pods, Services), we can't easily determine cluster membership
        // without fetching each resource and checking labels, so include all for now
        // This is acceptable since we're already filtering by namespace
        return true
      })
    }

    // Limit to requested number
    const limitedEvents = filteredEvents.slice(0, limit)

    // Transform events to our format
    const transformedEvents = limitedEvents.map((event: any) => {
      const involvedObject = event.involvedObject
      const timestamp = event.lastTimestamp || event.firstTimestamp || event.metadata.creationTimestamp
      
      // Map K8s resource to our resource types
      const resourceType = mapK8sKindToResourceType(involvedObject.kind)
      const resourceName = involvedObject.name
      const action = getActionFromEvent(event)
      const eventNamespace = involvedObject.namespace || targetNamespace

      return {
        id: event.metadata.uid || `${event.metadata.name}-${timestamp}`,
        type: resourceType,
        action,
        resourceName,
        namespace: eventNamespace,
        message: formatEventMessage(resourceType, resourceName, action, event),
        timestamp: new Date(timestamp).toISOString(),
        reason: event.reason,
        source: event.source?.component || 'kubernetes',
        eventType: event.type || 'Normal',
        count: event.count || 1,
        firstTimestamp: event.firstTimestamp ? new Date(event.firstTimestamp).toISOString() : undefined,
        lastTimestamp: event.lastTimestamp ? new Date(event.lastTimestamp).toISOString() : undefined,
        involvedObject: {
          kind: involvedObject.kind,
          name: involvedObject.name,
          namespace: involvedObject.namespace,
          uid: involvedObject.uid,
        }
      }
    })

    return NextResponse.json({
      success: true,
      data: transformedEvents,
      total: filteredEvents.length,
      namespace: targetNamespace,
      clusterName,
    })

  } catch (error) {
    console.error('Error fetching events:', error)
    return NextResponse.json(
      { 
        error: 'Failed to fetch events',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

// Helper function to map resource types to K8s kinds
function mapResourceTypeToK8sKind(resourceType: string): string {
  const mapping: Record<string, string> = {
    'agent': 'LanguageAgent',
    'model': 'LanguageModel',
    'tool': 'LanguageTool', 
    'persona': 'LanguagePersona',
    'cluster': 'LanguageCluster',
    'pod': 'Pod',
    'service': 'Service',
    'deployment': 'Deployment',
  }
  return mapping[resourceType.toLowerCase()] || resourceType
}

// Helper function to map K8s kinds to our resource types
function mapK8sKindToResourceType(kind: string): string {
  const mapping: Record<string, string> = {
    'LanguageAgent': 'agent',
    'LanguageModel': 'model',
    'LanguageTool': 'tool',
    'LanguagePersona': 'persona', 
    'LanguageCluster': 'cluster',
    'Pod': 'pod',
    'Service': 'service',
    'Deployment': 'deployment',
    'ReplicaSet': 'deployment',
    'Job': 'deployment',
    'CronJob': 'deployment',
  }
  return mapping[kind] || kind.toLowerCase()
}

// Helper function to determine action from K8s event
function getActionFromEvent(event: any): string {
  const reason = event.reason?.toLowerCase() || ''
  const type = event.type?.toLowerCase() || ''
  
  // Map common K8s event reasons to user-friendly actions
  if (reason.includes('created') || reason.includes('successfulcreate')) return 'created'
  if (reason.includes('updated') || reason.includes('update')) return 'updated'
  if (reason.includes('scaled') || reason.includes('scaling')) return 'scaled'
  if (reason.includes('failed') || reason.includes('error') || type === 'warning') return 'failed'
  if (reason.includes('ready') || reason.includes('reconciled')) return 'ready'
  if (reason.includes('started')) return 'started'
  if (reason.includes('stopped') || reason.includes('killed')) return 'stopped'
  if (reason.includes('pulling')) return 'pulling'
  if (reason.includes('pulled')) return 'pulled'
  
  // Default mapping based on event type
  if (type === 'normal') return 'normal'
  if (type === 'warning') return 'warning'
  
  return reason || 'updated'
}

// Helper function to format event messages
function formatEventMessage(resourceType: string, resourceName: string, action: string, event: any): string {
  const capitalizedType = resourceType.charAt(0).toUpperCase() + resourceType.slice(1)
  
  // If the event has a specific message, use it for more context
  if (event.message) {
    // For some events, the message is more descriptive
    if (event.message.length < 100) {
      return event.message
    }
  }
  
  // Otherwise, construct a user-friendly message
  switch (action) {
    case 'created':
      return `${capitalizedType} "${resourceName}" was created`
    case 'updated':
      return `${capitalizedType} "${resourceName}" was updated`
    case 'scaled':
      return `${capitalizedType} "${resourceName}" was scaled`
    case 'failed':
      return `${capitalizedType} "${resourceName}" failed`
    case 'ready':
      return `${capitalizedType} "${resourceName}" is ready`
    case 'started':
      return `${capitalizedType} "${resourceName}" started`
    case 'stopped':
      return `${capitalizedType} "${resourceName}" stopped`
    case 'warning':
      return `${capitalizedType} "${resourceName}" warning: ${event.reason}`
    default:
      return `${capitalizedType} "${resourceName}" ${action}`
  }
}