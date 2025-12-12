import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { k8sClient } from '@/lib/k8s-client'
import { z } from 'zod'

const createOrganizationSchema = z.object({
  name: z.string().min(1).max(100),
  slug: z.string().min(1).max(50).regex(/^[a-z0-9-]+$/),
  namespace: z.string().min(1).max(63).regex(/^[a-z0-9-]+$/)
})

export async function GET() {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const organizations = await prisma.organization.findMany({
      where: {
        members: {
          some: {
            user: {
              email: session.user.email
            }
          }
        }
      },
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
        },
        _count: {
          select: {
            members: true,
            invites: true
          }
        }
      },
      orderBy: {
        name: 'asc'
      }
    })

    return NextResponse.json({ organizations })
  } catch (error) {
    console.error('Error fetching organizations:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const body = await request.json()
    const validatedData = createOrganizationSchema.parse(body)

    const user = await prisma.user.findUnique({
      where: { email: session.user.email }
    })

    if (!user) {
      return NextResponse.json({ error: 'User not found' }, { status: 404 })
    }

    // Check if slug is unique
    const existingOrg = await prisma.organization.findFirst({
      where: {
        OR: [
          { slug: validatedData.slug },
          { namespace: validatedData.namespace }
        ]
      }
    })

    if (existingOrg) {
      return NextResponse.json(
        { error: 'Organization slug or namespace already exists' },
        { status: 409 }
      )
    }

    // Create organization with user as owner
    const organization = await prisma.organization.create({
      data: {
        ...validatedData,
        plan: 'free', // Default to free plan for manually created orgs
        members: {
          create: {
            userId: user.id,
            role: 'owner'
          }
        }
      },
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

    // Create Kubernetes namespace with ResourceQuota for the organization
    try {
      await k8sClient.createOrganizationNamespace(validatedData.namespace, organization.id, 'free')
    } catch (err: any) {
      // If namespace creation fails, log but don't fail organization creation
      console.error('Failed to create organization namespace:', err.message)
    }

    return NextResponse.json({ organization }, { status: 201 })
  } catch (error) {
    console.error('Error creating organization:', error)
    
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