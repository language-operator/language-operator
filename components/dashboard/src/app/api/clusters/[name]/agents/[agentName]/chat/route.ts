import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { parseAgentMessage } from '@/lib/message-parser'
import { ChatMessage } from '@/types/chat'
import { serviceResolver } from '@/lib/service-resolver'

interface RouteParams {
  params: Promise<{
    name: string
    agentName: string
  }>
}

// Remove local interface - we'll import from types
// interface ChatMessage {
//   role: 'user' | 'assistant' | 'system'
//   content: string
//   timestamp: string
// }

interface ChatRequest {
  message: string
  conversation?: ChatMessage[]
}

// POST - Send a message to the agent
export async function POST(request: NextRequest, { params }: RouteParams) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    // Check permissions - user needs view access to chat with agents
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    const { message, conversation = [] }: ChatRequest = await request.json()

    if (!message?.trim()) {
      return NextResponse.json({ error: 'Message is required' }, { status: 400 })
    }

    console.log(`Chat message for agent ${agentName} in cluster ${clusterName}, namespace ${organization.namespace}`)

    // Get the agent to validate it exists and get its configuration
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

    // Check if agent is ready/running
    if (agentData.status?.phase !== 'Ready' && agentData.status?.phase !== 'Running') {
      return NextResponse.json({
        error: 'Agent is not available',
        message: `Agent "${agentName}" is currently ${agentData.status?.phase || 'unknown'}. Please wait for it to become ready.`
      }, { status: 503 })
    }

    // Get agent's networking configuration to find chat endpoint
    const networking = agentData.spec?.networking
    const port = networking?.port || 8080 // Service port for agent communication
    const serviceName = `${agentName}` // Service name matches agent name
    
    // Resolve agent endpoint based on environment (K8s vs Docker Compose)
    const agentEndpoint = serviceResolver.resolveAgentChatUrl(serviceName, organization.namespace, port)
    
    console.log(`Environment: ${JSON.stringify(serviceResolver.getEnvironmentInfo())}`)
    console.log(`Resolved agent endpoint: ${agentEndpoint}`)

    // Prepare the chat completion request
    const systemMessage = agentData.spec?.instructions 
      ? `You are ${agentName}. ${agentData.spec.instructions}`
      : `You are ${agentName}, a helpful AI assistant.`

    const messages = [
      { role: 'system', content: systemMessage },
      ...conversation,
      { role: 'user', content: message }
    ]

    // Make the request to the agent's chat completion endpoint
    const agentResponse = await fetch(agentEndpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      body: JSON.stringify({
        messages,
        max_tokens: 1000,
        temperature: 0.7,
        stream: false
      }),
      // Add timeout for agent responses
      signal: AbortSignal.timeout(60000) // 60 second timeout
    })

    if (!agentResponse.ok) {
      console.error(`Agent endpoint error: ${agentResponse.status} ${agentResponse.statusText}`)
      return NextResponse.json({
        error: 'Agent communication failed',
        message: `Failed to communicate with agent "${agentName}". The agent may be starting up or experiencing issues.`
      }, { status: 502 })
    }

    const agentData_response = await agentResponse.json()

    // Extract the assistant's response
    const assistantMessage = agentData_response.choices?.[0]?.message?.content || 
                             agentData_response.message || 
                             'I apologize, but I was unable to generate a response.'

    // Parse the response to separate thinking content from actual response
    const parsed = parseAgentMessage(assistantMessage)

    // Create the response message
    const responseMessage: ChatMessage = {
      id: (Date.now() + 1).toString(), // Generate unique ID
      role: 'assistant',
      content: assistantMessage, // Keep original content for backward compatibility
      timestamp: new Date().toISOString(),
      // Add parsed content for UI separation
      thinkingContent: parsed.thinkingContent,
      responseContent: parsed.responseContent,
      hasThinking: parsed.hasThinking
    }

    // Return the chat response
    return NextResponse.json({
      success: true,
      message: responseMessage,
      agentName,
      agentStatus: agentData.status?.phase,
      // Include agent metadata for UI
      agentInfo: {
        executionMode: agentData.spec?.executionMode,
        modelRefs: agentData.spec?.modelRefs || [],
        toolRefs: agentData.spec?.toolRefs || [],
        personaRefs: agentData.spec?.personaRefs || []
      }
    })

  } catch (error) {
    console.error('Error in agent chat:', error)
    
    if (error instanceof Error && error.name === 'AbortError') {
      return NextResponse.json({
        error: 'Request timeout',
        message: 'The agent took too long to respond. Please try again.'
      }, { status: 408 })
    }
    
    return NextResponse.json(
      { 
        error: 'Failed to chat with agent',
        message: error instanceof Error ? error.message : 'Unknown error occurred'
      },
      { status: 500 }
    )
  }
}

// GET - Get agent chat status and capabilities
export async function GET(request: NextRequest, { params }: RouteParams) {
  try {
    // Get user's selected organization
    const { user, organization } = await getUserOrganization(request)
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params

    // Get the agent
    const agent = await k8sClient.getLanguageAgent(organization.namespace, agentName)

    // Handle different response structures
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
        error: `Agent "${agentName}" not found`
      }, { status: 404 })
    }

    // Return agent chat capabilities and status
    return NextResponse.json({
      success: true,
      agentName,
      chatAvailable: agentData.status?.phase === 'Ready' || agentData.status?.phase === 'Running',
      status: agentData.status?.phase || 'Unknown',
      agentInfo: {
        instructions: agentData.spec?.instructions,
        executionMode: agentData.spec?.executionMode,
        modelRefs: agentData.spec?.modelRefs || [],
        toolRefs: agentData.spec?.toolRefs || [],
        personaRefs: agentData.spec?.personaRefs || [],
        networking: {
          port: agentData.spec?.networking?.port || 8080,
          enabled: !!agentData.spec?.networking
        }
      }
    })

  } catch (error) {
    console.error('Error getting agent chat status:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to get agent status',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}