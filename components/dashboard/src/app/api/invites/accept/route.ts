import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { z } from 'zod'

const acceptInviteSchema = z.object({
  token: z.string().min(1)
})

export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const body = await request.json()
    const validatedData = acceptInviteSchema.parse(body)

    // Find the invite
    const invite = await prisma.organizationInvite.findUnique({
      where: { token: validatedData.token },
      include: {
        organization: true
      }
    })

    if (!invite) {
      return NextResponse.json({ error: 'Invalid invite token' }, { status: 404 })
    }

    // Check if invite has expired
    if (invite.expiresAt < new Date()) {
      // Clean up expired invite
      await prisma.organizationInvite.delete({
        where: { id: invite.id }
      })
      return NextResponse.json({ error: 'Invite has expired' }, { status: 410 })
    }

    // Check if invite is for the current user's email
    if (invite.email !== session.user.email) {
      return NextResponse.json(
        { error: 'Invite is not for your email address' },
        { status: 403 }
      )
    }

    // Get user
    const user = await prisma.user.findUnique({
      where: { email: session.user.email }
    })

    if (!user) {
      return NextResponse.json({ error: 'User not found' }, { status: 404 })
    }

    // Check if user is already a member
    const existingMember = await prisma.organizationMember.findUnique({
      where: {
        organizationId_userId: {
          organizationId: invite.organizationId,
          userId: user.id
        }
      }
    })

    if (existingMember) {
      // Clean up the invite and return success (user is already a member)
      await prisma.organizationInvite.delete({
        where: { id: invite.id }
      })
      return NextResponse.json({
        message: 'You are already a member of this organization',
        organization: invite.organization
      })
    }

    // Accept the invite - create membership and delete invite
    const [member] = await prisma.$transaction([
      prisma.organizationMember.create({
        data: {
          organizationId: invite.organizationId,
          userId: user.id,
          role: invite.role
        },
        include: {
          organization: true,
          user: {
            select: {
              id: true,
              name: true,
              email: true,
              image: true
            }
          }
        }
      }),
      prisma.organizationInvite.delete({
        where: { id: invite.id }
      })
    ])

    return NextResponse.json({
      message: 'Successfully joined organization',
      member,
      organization: invite.organization
    })
  } catch (error) {
    console.error('Error accepting organization invite:', error)
    
    if (error instanceof z.ZodError) {
      return NextResponse.json(
        { error: 'Invalid input', details: error.issues },
        { status: 400 }
      )
    }

    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}