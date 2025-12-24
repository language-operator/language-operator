import { NextRequest, NextResponse } from 'next/server'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

// GET /api/conversations/[conversationId]/messages - Get all messages for a conversation
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ conversationId: string }> }
) {
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

    const { conversationId } = await params

    // Verify conversation belongs to user's organization
    const conversation = await db.conversation.findFirst({
      where: {
        id: conversationId,
        organizationId: organization.id,
      },
    })

    if (!conversation) {
      return NextResponse.json(
        { error: 'Conversation not found' },
        { status: 404 }
      )
    }

    // Fetch messages
    const messages = await db.conversationMessage.findMany({
      where: {
        conversationId,
      },
      orderBy: {
        timestamp: 'asc',
      },
    })

    return NextResponse.json({
      success: true,
      messages: messages.map((msg) => ({
        id: msg.id,
        role: msg.role,
        content: msg.content,
        toolCalls: msg.toolCalls,
        metadata: msg.metadata,
        timestamp: msg.timestamp.toISOString(),
      })),
    })
  } catch (error) {
    console.error('Error fetching messages:', error)
    return NextResponse.json(
      { error: 'Failed to fetch messages' },
      { status: 500 }
    )
  }
}

// POST /api/conversations/[conversationId]/messages - Add a message to a conversation
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ conversationId: string }> }
) {
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

    const { conversationId } = await params
    const { role, content, toolCalls, metadata } = await request.json()

    if (!role || !content) {
      return NextResponse.json(
        { error: 'role and content are required' },
        { status: 400 }
      )
    }

    // Verify conversation belongs to user's organization
    const conversation = await db.conversation.findFirst({
      where: {
        id: conversationId,
        organizationId: organization.id,
      },
    })

    if (!conversation) {
      return NextResponse.json(
        { error: 'Conversation not found' },
        { status: 404 }
      )
    }

    // Create message
    const message = await db.conversationMessage.create({
      data: {
        conversationId,
        role,
        content,
        toolCalls: toolCalls || null,
        metadata: metadata || null,
      },
    })

    // Update conversation's updatedAt timestamp
    await db.conversation.update({
      where: { id: conversationId },
      data: { updatedAt: new Date() },
    })

    return NextResponse.json({
      success: true,
      message: {
        id: message.id,
        role: message.role,
        content: message.content,
        toolCalls: message.toolCalls,
        metadata: message.metadata,
        timestamp: message.timestamp.toISOString(),
      },
    })
  } catch (error) {
    console.error('Error creating message:', error)
    return NextResponse.json(
      { error: 'Failed to create message' },
      { status: 500 }
    )
  }
}
