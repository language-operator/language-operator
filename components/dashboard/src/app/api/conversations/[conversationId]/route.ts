import { NextRequest, NextResponse } from 'next/server'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

// DELETE /api/conversations/[conversationId] - Delete a specific conversation
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ conversationId: string }> }
) {
  try {
    const { user, organization } = await getUserOrganization(request)

    const hasPermission = await requirePermission(
      user.id,
      organization.id,
      'delete'
    )
    if (!hasPermission) {
      return NextResponse.json(
        { error: 'Insufficient permissions to delete conversations' },
        { status: 403 }
      )
    }

    const { conversationId } = await params

    if (!conversationId) {
      return NextResponse.json(
        { error: 'Conversation ID is required' },
        { status: 400 }
      )
    }

    // Verify the conversation exists and belongs to the user's organization
    const conversation = await db.conversation.findUnique({
      where: {
        id: conversationId,
      },
      select: {
        id: true,
        organizationId: true,
        agentName: true,
        title: true,
      },
    })

    if (!conversation) {
      return NextResponse.json(
        { error: 'Conversation not found' },
        { status: 404 }
      )
    }

    if (conversation.organizationId !== organization.id) {
      return NextResponse.json(
        { error: 'Conversation does not belong to your organization' },
        { status: 403 }
      )
    }

    // Delete the conversation (messages will be cascade deleted due to foreign key constraints)
    await db.conversation.delete({
      where: {
        id: conversationId,
      },
    })

    return NextResponse.json({
      success: true,
      message: 'Conversation deleted successfully',
      deletedConversation: {
        id: conversation.id,
        agentName: conversation.agentName,
        title: conversation.title,
      },
    })
  } catch (error) {
    console.error('Error deleting conversation:', error)
    return NextResponse.json(
      { error: 'Failed to delete conversation' },
      { status: 500 }
    )
  }
}