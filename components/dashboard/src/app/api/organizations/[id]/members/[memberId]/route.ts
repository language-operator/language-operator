import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { z } from 'zod'

const updateMemberSchema = z.object({
  role: z.enum(['owner', 'admin', 'editor', 'viewer'])
})

interface RouteParams {
  params: Promise<{ id: string; memberId: string }>
}

async function checkOrganizationAccess(organizationId: string, userEmail: string, requiredRole?: string) {
  const membership = await prisma.organizationMember.findFirst({
    where: {
      organizationId,
      user: { email: userEmail }
    }
  })

  if (!membership) {
    return { hasAccess: false, membership: null }
  }

  const roleHierarchy = { owner: 4, admin: 3, editor: 2, viewer: 1 }
  const userRole = roleHierarchy[membership.role as keyof typeof roleHierarchy]
  const requiredRoleLevel = requiredRole ? roleHierarchy[requiredRole as keyof typeof roleHierarchy] : 1

  return {
    hasAccess: userRole >= requiredRoleLevel,
    membership
  }
}

export async function PATCH(request: NextRequest, { params }: RouteParams) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const { id, memberId } = await params

    const { hasAccess, membership } = await checkOrganizationAccess(id, session.user.email, 'admin')
    if (!hasAccess || !membership) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    const body = await request.json()
    const validatedData = updateMemberSchema.parse(body)

    // Get the member being updated
    const targetMember = await prisma.organizationMember.findUnique({
      where: { id: memberId },
      include: {
        user: {
          select: {
            email: true
          }
        }
      }
    })

    if (!targetMember || targetMember.organizationId !== id) {
      return NextResponse.json({ error: 'Member not found' }, { status: 404 })
    }

    // Role change restrictions
    const roleHierarchy = { owner: 4, admin: 3, editor: 2, viewer: 1 }
    const currentUserRole = roleHierarchy[membership.role as keyof typeof roleHierarchy]
    const targetCurrentRole = roleHierarchy[targetMember.role as keyof typeof roleHierarchy]
    const targetNewRole = roleHierarchy[validatedData.role as keyof typeof roleHierarchy]

    // Can't modify someone with equal or higher role (unless you're owner)
    if (membership.role !== 'owner' && targetCurrentRole >= currentUserRole) {
      return NextResponse.json({ error: 'Cannot modify member with equal or higher role' }, { status: 403 })
    }

    // Can't grant a role higher than your own
    if (targetNewRole > currentUserRole) {
      return NextResponse.json({ error: 'Cannot grant role higher than your own' }, { status: 403 })
    }

    // Can't change your own role (prevents owners from accidentally demoting themselves)
    if (targetMember.user.email === session.user.email) {
      return NextResponse.json({ error: 'Cannot change your own role' }, { status: 403 })
    }

    // Only owners can create other owners
    if (validatedData.role === 'owner' && membership.role !== 'owner') {
      return NextResponse.json({ error: 'Only owners can create other owners' }, { status: 403 })
    }

    const updatedMember = await prisma.organizationMember.update({
      where: { id: memberId },
      data: { role: validatedData.role },
      include: {
        user: {
          select: {
            id: true,
            name: true,
            email: true,
            image: true,
            createdAt: true
          }
        }
      }
    })

    return NextResponse.json({ member: updatedMember })
  } catch (error) {
    console.error('Error updating organization member:', error)
    
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

export async function DELETE(request: NextRequest, { params }: RouteParams) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const { id, memberId } = await params

    const { hasAccess, membership } = await checkOrganizationAccess(id, session.user.email, 'admin')
    if (!hasAccess || !membership) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    // Get the member being removed
    const targetMember = await prisma.organizationMember.findUnique({
      where: { id: memberId },
      include: {
        user: {
          select: {
            email: true
          }
        }
      }
    })

    if (!targetMember || targetMember.organizationId !== id) {
      return NextResponse.json({ error: 'Member not found' }, { status: 404 })
    }

    // Role removal restrictions
    const roleHierarchy = { owner: 4, admin: 3, editor: 2, viewer: 1 }
    const currentUserRole = roleHierarchy[membership.role as keyof typeof roleHierarchy]
    const targetRole = roleHierarchy[targetMember.role as keyof typeof roleHierarchy]

    // Can't remove someone with equal or higher role (unless you're owner)
    if (membership.role !== 'owner' && targetRole >= currentUserRole) {
      return NextResponse.json({ error: 'Cannot remove member with equal or higher role' }, { status: 403 })
    }

    // Check if this is the last owner
    if (targetMember.role === 'owner') {
      const ownerCount = await prisma.organizationMember.count({
        where: {
          organizationId: id,
          role: 'owner'
        }
      })

      if (ownerCount <= 1) {
        return NextResponse.json(
          { error: 'Cannot remove the last owner of the organization' },
          { status: 403 }
        )
      }
    }

    await prisma.organizationMember.delete({
      where: { id: memberId }
    })

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error removing organization member:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}