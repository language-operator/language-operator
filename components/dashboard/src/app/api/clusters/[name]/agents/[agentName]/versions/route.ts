import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

interface RouteParams {
  params: Promise<{
    name: string
    agentName: string
  }>
}

export async function GET(request: NextRequest, { params }: RouteParams) {
  const { name: clusterName, agentName } = await params
  
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    console.log(`Listing LanguageAgentVersions for agent ${agentName} in namespace ${organization.namespace}`)

    // First, get the agent to verify it exists and get the agentVersionRef
    let agent: any = null
    try {
      const agentResponse = await k8sClient.getLanguageAgent(organization.namespace, agentName)
      if ((agentResponse as any)?.body) {
        agent = (agentResponse as any).body
      } else if ((agentResponse as any)?.data) {
        agent = (agentResponse as any).data
      } else if (agentResponse) {
        agent = agentResponse
      }
    } catch (agentError) {
      if (agentError instanceof Error && agentError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Agent not found',
          message: `LanguageAgent "${agentName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      throw agentError
    }

    // Get all LanguageAgentVersions in the namespace
    let versionsResponse: any = null
    try {
      versionsResponse = await k8sClient.listLanguageAgentVersions(organization.namespace)
    } catch (k8sError) {
      console.error('Error listing agent versions:', k8sError)
      return NextResponse.json(
        { 
          error: 'Failed to list agent versions',
          message: k8sError instanceof Error ? k8sError.message : 'Unknown error'
        },
        { status: 500 }
      )
    }

    let allVersions: any[] = []
    
    // Handle different response structures from k8s client
    if ((versionsResponse as any)?.body?.items) {
      allVersions = (versionsResponse as any).body.items
    } else if ((versionsResponse as any)?.data?.items) {
      allVersions = (versionsResponse as any).data.items
    } else if ((versionsResponse as any)?.items) {
      allVersions = (versionsResponse as any).items
    } else if (Array.isArray(versionsResponse)) {
      allVersions = versionsResponse
    }

    // Filter versions that belong to this agent and organization
    const agentVersions = allVersions.filter(version => {
      // Check if version belongs to this agent
      const belongsToAgent = version.spec?.agentRef?.name === agentName
      
      // Check organization ownership
      const versionOrgLabel = version.metadata?.labels?.['langop.io/organization-id']
      const belongsToOrg = !versionOrgLabel || versionOrgLabel === organization.id
      
      return belongsToAgent && belongsToOrg
    })

    // Sort versions by version number (descending - newest first)
    agentVersions.sort((a, b) => {
      const versionA = a.spec?.version || 0
      const versionB = b.spec?.version || 0
      return versionB - versionA
    })

    // Determine current version
    const currentVersionRef = agent?.spec?.agentVersionRef
    const currentVersionName = currentVersionRef?.name

    // Add metadata to each version
    const enrichedVersions = agentVersions.map(version => ({
      ...version,
      isCurrent: version.metadata.name === currentVersionName,
      isLocked: currentVersionRef?.lock || false
    }))

    return NextResponse.json({
      data: enrichedVersions,
      currentVersion: currentVersionName,
      isLocked: currentVersionRef?.lock || false,
      message: `Found ${agentVersions.length} versions for agent "${agentName}"`
    })

  } catch (error) {
    console.error('Error listing agent versions:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to list agent versions',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}