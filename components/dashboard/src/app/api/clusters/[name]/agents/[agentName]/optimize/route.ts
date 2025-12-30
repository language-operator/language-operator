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

export async function POST(request: NextRequest, { params }: RouteParams) {
  const { name: clusterName, agentName } = await params
  
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    console.log(`Triggering manual optimization for agent ${agentName} in namespace ${organization.namespace}`)

    // Get the current agent
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

    // Verify agent belongs to organization
    const agentOrgLabel = agent.metadata?.labels?.['langop.io/organization-id']
    if (agentOrgLabel && agentOrgLabel !== organization.id) {
      return NextResponse.json({ 
        error: 'Agent not found',
        message: `LanguageAgent "${agentName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    // Check if agent version is locked
    if (agent.spec?.agentVersionRef?.lock) {
      return NextResponse.json({ 
        error: 'Agent locked',
        message: `Agent "${agentName}" version is locked and cannot be optimized. Unlock the version first.`
      }, { status: 400 })
    }

    // Manual optimization bypasses the automatic learning threshold
    // Users can trigger optimization on demand regardless of execution count

    // Use annotations to trigger optimization since status is read-only
    // The learning controller will watch for these annotations
    const updatedAgent = {
      ...agent,
      metadata: {
        ...agent.metadata,
        annotations: {
          ...agent.metadata.annotations,
          'langop.io/manual-optimization-requested': new Date().toISOString(),
          'langop.io/optimization-trigger': 'manual',
          'langop.io/learning-request-pending': 'true'
        }
      }
    }

    // Apply the update to Kubernetes to set learningRequestPending = true
    // Since status updates require special handling, we'll update the full agent resource
    try {
      const updateResponse = await k8sClient.updateLanguageAgent(organization.namespace, agentName, updatedAgent)
      
      let updatedAgentResult: any = null
      if ((updateResponse as any)?.body) {
        updatedAgentResult = (updateResponse as any).body
      } else if ((updateResponse as any)?.data) {
        updatedAgentResult = (updateResponse as any).data
      } else if (updateResponse) {
        updatedAgentResult = updateResponse
      }

      return NextResponse.json({
        data: updatedAgentResult,
        message: `Successfully triggered manual optimization for agent "${agentName}". The learning controller will process this request.`,
        optimizationStatus: {
          triggered: true,
          triggerType: 'manual',
          timestamp: new Date().toISOString()
        }
      })

    } catch (updateError) {
      console.error('Error triggering agent optimization:', updateError)
      throw updateError
    }

  } catch (error) {
    console.error('Error triggering agent optimization:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to trigger optimization',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}