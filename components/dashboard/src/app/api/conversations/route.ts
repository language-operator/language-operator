import { NextRequest, NextResponse } from 'next/server'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

// GET /api/conversations - List all conversations for user's organization
export async function GET(request: NextRequest) {
  try {
    const { user, organization } = await getUserOrganization(request)

    const hasPermission = await requirePermission(
      user.id,
      organization.id,
      'view'
    )
    if (!hasPermission) {
      return NextResponse.json(
        { error: 'Insufficient permissions' },
        { status: 403 }
      )
    }

    // Fetch conversations from database
    const conversations = await db.conversation.findMany({
      where: {
        organizationId: organization.id,
      },
      orderBy: {
        updatedAt: 'desc',
      },
      include: {
        _count: {
          select: {
            messages: true,
          },
        },
      },
    })

    // Transform to include message count
    const conversationsWithCounts = conversations.map((conv) => ({
      id: conv.id,
      agentName: conv.agentName,
      clusterName: conv.clusterName,
      title: conv.title,
      createdAt: conv.createdAt.toISOString(),
      updatedAt: conv.updatedAt.toISOString(),
      messageCount: conv._count.messages,
    }))

    return NextResponse.json({
      success: true,
      conversations: conversationsWithCounts,
    })
  } catch (error) {
    console.error('Error fetching conversations:', error)
    return NextResponse.json(
      { error: 'Failed to fetch conversations' },
      { status: 500 }
    )
  }
}

// POST /api/conversations - Create a new conversation
export async function POST(request: NextRequest) {
  try {
    const { user, organization } = await getUserOrganization(request)

    const hasPermission = await requirePermission(
      user.id,
      organization.id,
      'create'
    )
    if (!hasPermission) {
      return NextResponse.json(
        { error: 'Insufficient permissions' },
        { status: 403 }
      )
    }

    const { agentName, clusterName, title } = await request.json()

    if (!agentName || !clusterName) {
      return NextResponse.json(
        { error: 'agentName and clusterName are required' },
        { status: 400 }
      )
    }

    // Create conversation
    const conversation = await db.conversation.create({
      data: {
        organizationId: organization.id,
        agentName,
        clusterName,
        title,
      },
    })

    return NextResponse.json({
      success: true,
      conversation: {
        id: conversation.id,
        agentName: conversation.agentName,
        clusterName: conversation.clusterName,
        title: conversation.title,
        createdAt: conversation.createdAt.toISOString(),
        updatedAt: conversation.updatedAt.toISOString(),
      },
    })
  } catch (error) {
    console.error('Error creating conversation:', error)
    return NextResponse.json(
      { error: 'Failed to create conversation' },
      { status: 500 }
    )
  }
}
