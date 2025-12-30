import { NextRequest, NextResponse } from 'next/server'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { z } from 'zod'

// Validation schema for PATCH request
const updateConversationSchema = z.object({
  title: z.string().min(1, 'Title is required').max(100, 'Title must be less than 100 characters').trim(),
})

// PATCH /api/conversations/[conversationId] - Update a specific conversation
export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ conversationId: string }> }
) {
  try {
    const { user, organization } = await getUserOrganization(request)

    const hasPermission = await requirePermission(
      user.id,
      organization.id,
      'edit'
    )
    if (!hasPermission) {
      return NextResponse.json(
        { error: 'Insufficient permissions to update conversations' },
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

    // Parse and validate request body
    const body = await request.json()
    const validation = updateConversationSchema.safeParse(body)
    
    if (!validation.success) {
      return NextResponse.json(
        { error: 'Invalid input', details: validation.error.issues },
        { status: 400 }
      )
    }

    const { title } = validation.data

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

    // Update the conversation title
    const updatedConversation = await db.conversation.update({
      where: {
        id: conversationId,
      },
      data: {
        title: title,
      },
      select: {
        id: true,
        title: true,
        agentName: true,
        clusterName: true,
        createdAt: true,
        updatedAt: true,
      },
    })

    return NextResponse.json({
      success: true,
      message: 'Conversation updated successfully',
      conversation: updatedConversation,
    })
  } catch (error) {
    console.error('Error updating conversation:', error)
    return NextResponse.json(
      { error: 'Failed to update conversation' },
      { status: 500 }
    )
  }
}

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