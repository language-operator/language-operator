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
    versionName: string
  }>
}

export async function DELETE(request: NextRequest, { params }: RouteParams) {
  const { name: clusterName, agentName, versionName } = await params
  
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'delete')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    console.log(`Deleting LanguageAgentVersion ${versionName} for agent ${agentName} in namespace ${organization.namespace}`)

    // First, get the target version to verify it exists and belongs to this agent
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

    // Prevent deletion of v1 (base version)
    if (targetVersion.spec?.version === 1) {
      return NextResponse.json({ 
        error: 'Cannot delete base version',
        message: 'Version v1 cannot be deleted as it is the base version'
      }, { status: 400 })
    }

    // Get the current agent to check if we're deleting the current version
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

    const currentVersionRef = agent?.spec?.agentVersionRef
    const currentVersionName = currentVersionRef?.name
    const isCurrentVersion = currentVersionName === versionName

    // If current version is locked, prevent deletion
    if (isCurrentVersion && currentVersionRef?.lock) {
      return NextResponse.json({ 
        error: 'Cannot delete locked version',
        message: 'The current version is locked and cannot be deleted. Unlock it first.'
      }, { status: 400 })
    }

    let rollbackVersionName: string | null = null
    
    // If deleting current version, we need to find the previous version to rollback to
    if (isCurrentVersion) {
      // Get all versions for this agent to find the previous one
      let versionsResponse: any = null
      try {
        versionsResponse = await k8sClient.listLanguageAgentVersions(organization.namespace)
      } catch (k8sError) {
        console.error('Error listing agent versions:', k8sError)
        return NextResponse.json(
          { 
            error: 'Failed to list agent versions',
            message: 'Could not determine previous version for rollback'
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
        const belongsToAgent = version.spec?.agentRef?.name === agentName
        const versionOrgLabel = version.metadata?.labels?.['langop.io/organization-id']
        const belongsToOrg = !versionOrgLabel || versionOrgLabel === organization.id
        return belongsToAgent && belongsToOrg
      })

      // Sort by version number descending, excluding the version we're deleting
      const otherVersions = agentVersions
        .filter(version => version.metadata.name !== versionName)
        .sort((a, b) => {
          const versionA = a.spec?.version || 0
          const versionB = b.spec?.version || 0
          return versionB - versionA
        })

      if (otherVersions.length === 0) {
        return NextResponse.json({ 
          error: 'Cannot delete last version',
          message: 'Cannot delete the only remaining version of the agent'
        }, { status: 400 })
      }

      // Use the highest version number that's not the one being deleted
      rollbackVersionName = otherVersions[0].metadata.name
    }

    // If we need to rollback, do it first
    if (rollbackVersionName) {
      const updatedAgent = {
        ...agent,
        spec: {
          ...agent.spec,
          agentVersionRef: {
            name: rollbackVersionName,
            namespace: organization.namespace,
            lock: false // Don't lock after rollback
          }
        }
      }

      try {
        await k8sClient.updateLanguageAgent(organization.namespace, agentName, updatedAgent)
        console.log(`Rolled back agent ${agentName} to version ${rollbackVersionName} before deletion`)
      } catch (updateError) {
        console.error('Error rolling back agent:', updateError)
        return NextResponse.json(
          { 
            error: 'Failed to rollback agent',
            message: 'Could not rollback to previous version before deletion'
          },
          { status: 500 }
        )
      }
    }

    // Delete the LanguageAgentVersion CRD
    // Note: Associated ConfigMaps are managed by the operator's garbage collection
    try {
      await k8sClient.deleteLanguageAgentVersion(organization.namespace, versionName)
      console.log(`Deleted LanguageAgentVersion ${versionName}`)
    } catch (deleteError) {
      console.error('Error deleting LanguageAgentVersion:', deleteError)
      throw deleteError
    }

    const successMessage = rollbackVersionName 
      ? `Successfully deleted version "${versionName}" and rolled back agent "${agentName}" to previous version`
      : `Successfully deleted version "${versionName}"`

    console.log(`User ${user.email} deleted LanguageAgentVersion ${versionName} for agent ${agentName} in cluster ${clusterName} in organization ${organization.name}`)

    return NextResponse.json({
      success: true,
      message: successMessage,
      rolledBackTo: rollbackVersionName,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error deleting agent version:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to delete agent version',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}