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

    // Parse request body
    const body = await request.json()
    const { lock } = body

    if (typeof lock !== 'boolean') {
      return NextResponse.json({ 
        error: 'Invalid request',
        message: 'lock field is required and must be a boolean'
      }, { status: 400 })
    }

    console.log(`${lock ? 'Locking' : 'Unlocking'} agent ${agentName} in namespace ${organization.namespace}`)

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

    // Check if agent has agentVersionRef
    if (!agent.spec?.agentVersionRef) {
      return NextResponse.json({ 
        error: 'No version reference',
        message: `Agent "${agentName}" does not have a version reference to lock/unlock`
      }, { status: 400 })
    }

    // Update the agent's agentVersionRef lock field
    const updatedAgent = {
      ...agent,
      spec: {
        ...agent.spec,
        agentVersionRef: {
          ...agent.spec.agentVersionRef,
          lock: lock
        }
      }
    }

    // Apply the update to Kubernetes
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
        message: `Successfully ${lock ? 'locked' : 'unlocked'} agent "${agentName}" version`
      })

    } catch (updateError) {
      console.error('Error updating agent lock status:', updateError)
      throw updateError
    }

  } catch (error) {
    console.error('Error updating agent lock status:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to update lock status',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}