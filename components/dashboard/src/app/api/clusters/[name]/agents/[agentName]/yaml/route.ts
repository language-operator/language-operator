import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import yaml from 'js-yaml'

// GET /api/clusters/[name]/agents/[agentName]/yaml - Get agent YAML
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
    let agent: any = null
    
    try {
      const response = await k8sClient.getLanguageAgent(organization.namespace, agentName)
      
      // Handle different response structures from k8s client
      if ((response as any)?.body) {
        agent = (response as any).body
      } else if ((response as any)?.data) {
        agent = (response as any).data
      } else if (response) {
        agent = response as any
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

    // Convert agent object to YAML
    const yamlContent = yaml.dump(agent, {
      indent: 2,
      lineWidth: 120,
      quotingType: '"',
      forceQuotes: false,
    })

    // Return as plain text with appropriate content-type
    return new NextResponse(yamlContent, {
      headers: {
        'Content-Type': 'text/yaml; charset=utf-8',
        'Cache-Control': 'no-cache',
      },
    })

  } catch (error) {
    console.error('Error fetching agent YAML:', error)
    return NextResponse.json({ 
      error: 'Failed to fetch agent YAML',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}