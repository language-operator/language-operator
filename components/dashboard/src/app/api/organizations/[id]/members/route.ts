import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { z } from 'zod'

const addMemberSchema = z.object({
  email: z.string().email(),
  role: z.enum(['admin', 'editor', 'viewer'])
})

const updateMemberSchema = z.object({
  role: z.enum(['owner', 'admin', 'editor', 'viewer'])
})

interface RouteParams {
  params: Promise<{ id: string }>
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

export async function GET(request: NextRequest, { params }: RouteParams) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const { id } = await params

    const { hasAccess } = await checkOrganizationAccess(id, session.user.email)
    if (!hasAccess) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    const members = await prisma.organizationMember.findMany({
      where: { organizationId: id },
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
      },
      orderBy: [
        { role: 'asc' },
        { createdAt: 'asc' }
      ]
    })

    return NextResponse.json({ members })
  } catch (error) {
    console.error('Error fetching organization members:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function POST(request: NextRequest, { params }: RouteParams) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const { id } = await params

    const { hasAccess } = await checkOrganizationAccess(id, session.user.email, 'admin')
    if (!hasAccess) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    const body = await request.json()
    const validatedData = addMemberSchema.parse(body)

    // Check if user exists
    const user = await prisma.user.findUnique({
      where: { email: validatedData.email }
    })

    if (!user) {
      return NextResponse.json({ error: 'User not found' }, { status: 404 })
    }

    // Check if user is already a member
    const existingMember = await prisma.organizationMember.findUnique({
      where: {
        organizationId_userId: {
          organizationId: id,
          userId: user.id
        }
      }
    })

    if (existingMember) {
      return NextResponse.json(
        { error: 'User is already a member of this organization' },
        { status: 409 }
      )
    }

    const member = await prisma.organizationMember.create({
      data: {
        organizationId: id,
        userId: user.id,
        role: validatedData.role
      },
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

    return NextResponse.json({ member }, { status: 201 })
  } catch (error) {
    console.error('Error adding organization member:', error)
    
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