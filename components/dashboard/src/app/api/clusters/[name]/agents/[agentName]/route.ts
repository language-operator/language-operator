import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { LanguageAgent, LanguageAgentFormData } from '@/types/agent'

// GET /api/clusters/[name]/agents/[agentName] - Get specific agent details
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    if (!clusterName || !agentName) {
      return NextResponse.json({ error: 'Cluster name and agent name are required' }, { status: 400 })
    }

    // Fetch specific agent from organization namespace
    let agent: LanguageAgent | null = null
    
    try {
      const response = await k8sClient.getLanguageAgent(organization.namespace, agentName)
      
      // Handle different response structures from k8s client
      if ((response as any)?.body) {
        agent = (response as any).body
      } else if ((response as any)?.data) {
        agent = (response as any).data
      } else if (response) {
        agent = response as LanguageAgent
      }
    } catch (k8sError) {
      // If agent not found, return 404
      if (k8sError instanceof Error && k8sError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Agent not found',
          details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      
      console.error('Error fetching agent from Kubernetes:', k8sError)
      throw k8sError
    }

    if (!agent) {
      return NextResponse.json({ 
        error: 'Agent not found',
        details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    // Verify agent belongs to user's organization
    const agentOrgLabel = agent.metadata?.labels?.['langop.io/organization-id']
    if (agentOrgLabel && agentOrgLabel !== organization.id) {
      return NextResponse.json({ 
        error: 'Agent not found',
        details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    return NextResponse.json({
      success: true,
      data: agent,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching agent details:', error)
    return NextResponse.json({ 
      error: 'Failed to fetch agent details',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}

// PATCH /api/clusters/[name]/agents/[agentName] - Update specific agent
export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    if (!clusterName || !agentName) {
      return NextResponse.json({ error: 'Cluster name and agent name are required' }, { status: 400 })
    }

    const body: Partial<LanguageAgentFormData> = await request.json()

    // First, get the current agent to ensure it exists and get its current state
    let currentAgent: LanguageAgent | null = null
    try {
      const response = await k8sClient.getLanguageAgent(organization.namespace, agentName)
      
      if ((response as any)?.body) {
        currentAgent = (response as any).body
      } else if ((response as any)?.data) {
        currentAgent = (response as any).data
      } else if (response) {
        currentAgent = response as LanguageAgent
      }
    } catch (k8sError) {
      if (k8sError instanceof Error && k8sError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Agent not found',
          details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      throw k8sError
    }

    if (!currentAgent) {
      return NextResponse.json({ 
        error: 'Agent not found',
        details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    // Verify agent belongs to user's organization
    const agentOrgLabel = currentAgent.metadata?.labels?.['langop.io/organization-id']
    if (agentOrgLabel && agentOrgLabel !== organization.id) {
      return NextResponse.json({ 
        error: 'Agent not found',
        details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    // Build the updated agent spec
    const updatedSpec: any = { ...currentAgent.spec }

    // Update basic fields
    if (body.instructions !== undefined) {
      updatedSpec.instructions = body.instructions
    }

    // Update model references
    if (body.selectedModels) {
      updatedSpec.modelRefs = body.selectedModels.map(name => ({ name }))
    }

    // Update tool references
    if (body.selectedTools) {
      updatedSpec.toolRefs = body.selectedTools.map(name => ({ name }))
    }

    // Update persona references
    if (body.selectedPersona !== undefined) {
      if (body.selectedPersona && body.selectedPersona !== 'none') {
        updatedSpec.personaRefs = [{ name: body.selectedPersona }]
      } else {
        updatedSpec.personaRefs = []
      }
    }

    // Update resources
    if (body.cpuRequest || body.memoryRequest || body.cpuLimit || body.memoryLimit) {
      updatedSpec.resources = updatedSpec.resources || {}
      
      if (body.cpuRequest || body.memoryRequest) {
        updatedSpec.resources.requests = updatedSpec.resources.requests || {}
        if (body.cpuRequest) updatedSpec.resources.requests.cpu = body.cpuRequest
        if (body.memoryRequest) updatedSpec.resources.requests.memory = body.memoryRequest
      }
      
      if (body.cpuLimit || body.memoryLimit) {
        updatedSpec.resources.limits = updatedSpec.resources.limits || {}
        if (body.cpuLimit) updatedSpec.resources.limits.cpu = body.cpuLimit
        if (body.memoryLimit) updatedSpec.resources.limits.memory = body.memoryLimit
      }
    }

    // Update egress rules
    if (body.egressRules) {
      updatedSpec.egress = body.egressRules.map(rule => ({
        description: rule.description,
        ...(rule.dns && rule.dns.length > 0 || rule.cidr) && {
          to: {
            ...(rule.dns && rule.dns.length > 0 && { dns: rule.dns }),
            ...(rule.cidr && { cidr: rule.cidr }),
          },
        },
        ...(rule.ports && rule.ports.length > 0) && {
          ports: rule.ports,
        },
      }))
    }

    // Create the updated agent object
    const updatedAgent: LanguageAgent = {
      ...currentAgent,
      spec: updatedSpec,
    }

    // Update agent in Kubernetes
    try {
      const response = await k8sClient.updateLanguageAgent(organization.namespace, agentName, updatedAgent)
      
      console.log(`User ${user.email} updated LanguageAgent ${agentName} in cluster ${clusterName} in organization ${organization.name}`)
      
      return NextResponse.json({
        success: true,
        data: response,
        message: `Agent "${agentName}" updated successfully`,
        cluster: clusterName,
      })
    } catch (k8sError) {
      console.error('Error updating agent in Kubernetes:', k8sError)
      throw k8sError
    }

  } catch (error) {
    console.error('Error updating agent:', error)
    return NextResponse.json({ 
      error: 'Failed to update agent',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}

// DELETE /api/clusters/[name]/agents/[agentName] - Delete specific agent
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'delete')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    if (!clusterName || !agentName) {
      return NextResponse.json({ error: 'Cluster name and agent name are required' }, { status: 400 })
    }

    // Delete agent from organization namespace
    try {
      const response = await k8sClient.deleteLanguageAgent(organization.namespace, agentName)
      
      console.log(`User ${user.email} deleted LanguageAgent ${agentName} from cluster ${clusterName} in organization ${organization.name}`)
      
      return NextResponse.json({
        success: true,
        message: `Agent "${agentName}" deleted successfully`,
        cluster: clusterName,
      })
    } catch (k8sError) {
      // If agent not found, return 404
      if (k8sError instanceof Error && k8sError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Agent not found',
          details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      
      console.error('Error deleting agent from Kubernetes:', k8sError)
      throw k8sError
    }

  } catch (error) {
    console.error('Error deleting agent:', error)
    return NextResponse.json({ 
      error: 'Failed to delete agent',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}