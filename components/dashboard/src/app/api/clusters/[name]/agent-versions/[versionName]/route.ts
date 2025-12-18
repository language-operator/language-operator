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
    versionName: string
  }>
}

export async function GET(request: NextRequest, { params }: RouteParams) {
  const { name: clusterName, versionName } = await params
  
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    console.log(`Fetching LanguageAgentVersion ${versionName} from namespace ${organization.namespace}`)

    // Fetch the LanguageAgentVersion from Kubernetes
    let agentVersion: any = null
    
    try {
      const response = await k8sClient.getLanguageAgentVersion(organization.namespace, versionName)
      
      // Handle different response structures from k8s client
      if ((response as any)?.body) {
        agentVersion = (response as any).body
      } else if ((response as any)?.data) {
        agentVersion = (response as any).data
      } else if (response) {
        agentVersion = response
      }
    } catch (k8sError) {
      // If agent version not found, return 404
      if (k8sError instanceof Error && k8sError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Agent version not found',
          message: `LanguageAgentVersion "${versionName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      throw k8sError // Re-throw other k8s errors to be handled by outer catch
    }
    
    if (!agentVersion) {
      return NextResponse.json({ 
        error: 'Agent version not found',
        message: `LanguageAgentVersion "${versionName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    // Verify the agent version belongs to user's organization
    const agentVersionOrgLabel = agentVersion.metadata?.labels?.['langop.io/organization-id']
    if (agentVersionOrgLabel && agentVersionOrgLabel !== organization.id) {
      return NextResponse.json({ 
        error: 'Agent version not found',
        message: `LanguageAgentVersion "${versionName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    return NextResponse.json({
      data: agentVersion,
      message: 'Agent version retrieved successfully'
    })

  } catch (error) {
    console.error('Error fetching agent version:', error)
    
    // Handle specific error types
    if ((error as any)?.response?.statusCode === 404) {
      return NextResponse.json({ 
        error: 'Agent version not found',
        message: `LanguageAgentVersion "${versionName}" not found` 
      }, { status: 404 })
    }

    return NextResponse.json(
      { 
        error: 'Failed to fetch agent version',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}