import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { config } from '@/lib/config'
import { z } from 'zod'
import crypto from 'crypto'

const createInviteSchema = z.object({
  email: z.string().email(),
  role: z.enum(['admin', 'editor', 'viewer'])
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

    const { hasAccess } = await checkOrganizationAccess(id, session.user.email, 'admin')
    if (!hasAccess) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    const invites = await prisma.organizationInvite.findMany({
      where: { 
        organizationId: id,
        expiresAt: {
          gt: new Date()
        }
      },
      orderBy: {
        createdAt: 'desc'
      }
    })

    return NextResponse.json({ invites })
  } catch (error) {
    console.error('Error fetching organization invites:', error)
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
    const validatedData = createInviteSchema.parse(body)

    // Check if user is already a member
    const existingMember = await prisma.organizationMember.findFirst({
      where: {
        organizationId: id,
        user: { email: validatedData.email }
      }
    })

    if (existingMember) {
      return NextResponse.json(
        { error: 'User is already a member of this organization' },
        { status: 409 }
      )
    }

    // Check for existing pending invite
    const existingInvite = await prisma.organizationInvite.findFirst({
      where: {
        organizationId: id,
        email: validatedData.email,
        expiresAt: {
          gt: new Date()
        }
      }
    })

    if (existingInvite) {
      return NextResponse.json(
        { error: 'Pending invite already exists for this email' },
        { status: 409 }
      )
    }

    // Generate secure token
    const token = crypto.randomBytes(32).toString('hex')
    
    // Set expiration to 7 days from now
    const expiresAt = new Date()
    expiresAt.setDate(expiresAt.getDate() + 7)

    const invite = await prisma.organizationInvite.create({
      data: {
        organizationId: id,
        email: validatedData.email,
        role: validatedData.role,
        token,
        expiresAt
      }
    })

    // TODO: Send invitation email here
    // You can integrate with your email service (SendGrid, SES, etc.)

    // Generate invitation URL
    const invitationUrl = `${config.dashboardUrl}/invites/${token}`

    return NextResponse.json({ 
      invite: {
        ...invite,
        invitationUrl
      }
    }, { status: 201 })
  } catch (error) {
    console.error('Error creating organization invite:', error)
    
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