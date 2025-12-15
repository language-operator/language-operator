import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { k8sClient } from '@/lib/k8s-client'
import { z } from 'zod'

const updateOrganizationSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  slug: z.string().min(1).max(50).regex(/^[a-z0-9-]+$/).optional(),
  namespace: z.string().min(1).max(63).regex(/^[a-z0-9-]+$/).optional(),
  plan: z.enum(['free', 'pro', 'enterprise']).optional()
})

interface RouteParams {
  params: Promise<{ id: string }>
}

async function checkOrganizationAccess(organizationId: string, userEmail: string, requiredRole?: string) {
  const membership = await prisma.organizationMember.findFirst({
    where: {
      organizationId,
      user: { email: userEmail }
    },
    include: {
      organization: true,
      user: true
    }
  })

  if (!membership) {
    return { hasAccess: false, membership: null }
  }

  // Check role hierarchy: owner > admin > editor > viewer
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

    const { hasAccess, membership } = await checkOrganizationAccess(id, session.user.email)
    if (!hasAccess) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    const organization = await prisma.organization.findUnique({
      where: { id },
      include: {
        members: {
          include: {
            user: {
              select: {
                id: true,
                name: true,
                email: true,
                image: true
              }
            }
          },
          orderBy: [
            { role: 'asc' }, // owners first
            { createdAt: 'asc' }
          ]
        },
        invites: {
          where: {
            expiresAt: {
              gt: new Date()
            }
          },
          orderBy: {
            createdAt: 'desc'
          }
        },
        _count: {
          select: {
            members: true,
            invites: true
          }
        }
      }
    })

    if (!organization) {
      return NextResponse.json({ error: 'Organization not found' }, { status: 404 })
    }

    return NextResponse.json({ 
      organization,
      userRole: membership?.role 
    })
  } catch (error) {
    console.error('Error fetching organization:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function PATCH(request: NextRequest, { params }: RouteParams) {
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
    const validatedData = updateOrganizationSchema.parse(body)

    // Check if slug or namespace conflicts with other organizations
    if (validatedData.slug || validatedData.namespace) {
      const conflictCheck = await prisma.organization.findFirst({
        where: {
          AND: [
            { id: { not: id } },
            {
              OR: [
                ...(validatedData.slug ? [{ slug: validatedData.slug }] : []),
                ...(validatedData.namespace ? [{ namespace: validatedData.namespace }] : [])
              ]
            }
          ]
        }
      })

      if (conflictCheck) {
        return NextResponse.json(
          { error: 'Organization slug or namespace already exists' },
          { status: 409 }
        )
      }
    }

    const organization = await prisma.organization.update({
      where: { id },
      data: validatedData,
      include: {
        members: {
          include: {
            user: {
              select: {
                id: true,
                name: true,
                email: true,
                image: true
              }
            }
          }
        }
      }
    })

    return NextResponse.json({ organization })
  } catch (error) {
    console.error('Error updating organization:', error)
    
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

    const { id } = await params

    const { hasAccess } = await checkOrganizationAccess(id, session.user.email, 'owner')
    if (!hasAccess) {
      return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
    }

    // Get organization details before deletion to clean up namespace
    const organization = await prisma.organization.findUnique({
      where: { id }
    })

    if (!organization) {
      return NextResponse.json({ error: 'Organization not found' }, { status: 404 })
    }

    // Delete organization (cascading deletes will handle members and invites)
    await prisma.organization.delete({
      where: { id }
    })

    // Clean up Kubernetes namespace
    try {
      await k8sClient.deleteOrganizationNamespace(organization.namespace)
    } catch (k8sError: any) {
      // Log but don't fail deletion if namespace cleanup fails
      console.warn(`Warning: Failed to delete namespace ${organization.namespace}:`, k8sError.message)
    }

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting organization:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}