import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'

interface RouteParams {
  params: Promise<{
    name: string
    agentName: string
  }>
}

export async function POST(request: NextRequest, { params }: RouteParams) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    // Check permissions - user needs edit access to manually run agents
    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params

    console.log(`Manual execution requested for agent ${agentName} in cluster ${clusterName}, namespace ${organization.namespace}`)

    // Get the agent to validate it exists and is scheduled
    const agent = await k8sClient.getLanguageAgent(organization.namespace, agentName)

    // Handle different response structures from k8s client  
    let agentData: any = null
    if ((agent as any)?.body) {
      agentData = (agent as any).body
    } else if ((agent as any)?.data) {
      agentData = (agent as any).data
    } else if (agent) {
      agentData = agent
    }

    if (!agentData) {
      return NextResponse.json({ 
        error: `Agent "${agentName}" not found`,
        message: `Agent "${agentName}" not found in namespace ${organization.namespace}`
      }, { status: 404 })
    }

    // Validate agent is scheduled execution mode
    if (agentData.spec?.executionMode !== 'scheduled') {
      return NextResponse.json({
        error: 'Agent is not scheduled',
        message: `Agent "${agentName}" has execution mode "${agentData.spec?.executionMode || 'unknown'}", only scheduled agents can be run manually`
      }, { status: 400 })
    }

    // Generate unique job name
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-').toLowerCase()
    const jobName = `${agentName}-manual-${timestamp}`

    // Create manual execution Job from the agent's CronJob
    const createdJob = await k8sClient.createJobFromCronJob(organization.namespace, agentName, jobName)
    
    console.log(`Manual execution Job created: ${jobName}`)

    // Return job information
    return NextResponse.json({
      success: true,
      jobName,
      agentName,
      namespace: organization.namespace,
      message: `Manual execution started for agent "${agentName}"`,
      job: {
        name: jobName,
        namespace: organization.namespace,
        createdAt: new Date().toISOString()
      }
    })

  } catch (error) {
    console.error('Error creating manual execution job:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to execute agent manually',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}