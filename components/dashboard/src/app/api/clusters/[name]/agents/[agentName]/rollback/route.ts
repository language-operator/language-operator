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

export async function PUT(request: NextRequest, { params }: RouteParams) {
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
    const { versionName, lock = false } = body

    if (!versionName || typeof versionName !== 'string') {
      return NextResponse.json({ 
        error: 'Invalid request',
        message: 'versionName is required and must be a string'
      }, { status: 400 })
    }

    console.log(`Rolling back agent ${agentName} to version ${versionName} in namespace ${organization.namespace}`)

    // First, verify the target version exists and belongs to this agent
    let targetVersion: any = null
    try {
      const versionResponse = await k8sClient.getLanguageAgentVersion(organization.namespace, versionName)
      if ((versionResponse as any)?.body) {
        targetVersion = (versionResponse as any).body
      } else if ((versionResponse as any)?.data) {
        targetVersion = (versionResponse as any).data
      } else if (versionResponse) {
        targetVersion = versionResponse
      }
    } catch (versionError) {
      if (versionError instanceof Error && versionError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Version not found',
          message: `LanguageAgentVersion "${versionName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      throw versionError
    }

    // Verify the version belongs to this agent
    if (targetVersion.spec?.agentRef?.name !== agentName) {
      return NextResponse.json({ 
        error: 'Invalid version',
        message: `Version "${versionName}" does not belong to agent "${agentName}"`
      }, { status: 400 })
    }

    // Verify organization ownership
    const versionOrgLabel = targetVersion.metadata?.labels?.['langop.io/organization-id']
    if (versionOrgLabel && versionOrgLabel !== organization.id) {
      return NextResponse.json({ 
        error: 'Version not found',
        message: `LanguageAgentVersion "${versionName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

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

    // Update the agent's agentVersionRef
    const updatedAgent = {
      ...agent,
      spec: {
        ...agent.spec,
        agentVersionRef: {
          name: versionName,
          namespace: organization.namespace, // Ensure same namespace
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
        message: `Successfully rolled back agent "${agentName}" to version "${versionName}"${lock ? ' (locked)' : ''}`
      })

    } catch (updateError) {
      console.error('Error updating agent:', updateError)
      throw updateError
    }

  } catch (error) {
    console.error('Error rolling back agent:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to roll back agent',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}