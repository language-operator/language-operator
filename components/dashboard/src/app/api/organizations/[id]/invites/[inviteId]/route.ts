import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'

interface RouteParams {
  params: Promise<{ id: string; inviteId: string }>
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

export async function DELETE(request: NextRequest, { params }: RouteParams) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const { id, inviteId } = await params

    const { hasAccess } = await checkOrganizationAccess(id, session.user.email, 'admin')
    if (!hasAccess) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    // Verify the invite belongs to this organization
    const invite = await prisma.organizationInvite.findUnique({
      where: { id: inviteId }
    })

    if (!invite || invite.organizationId !== id) {
      return NextResponse.json({ error: 'Invite not found' }, { status: 404 })
    }

    await prisma.organizationInvite.delete({
      where: { id: inviteId }
    })

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting organization invite:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}