import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { k8sClient } from '@/lib/k8s-client'
import { generateOrganizationNamespace } from '@/lib/namespace-utils'
import { z } from 'zod'

const createOrganizationSchema = z.object({
  name: z.string().min(1).max(100)
})

function generateSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '') // Remove invalid characters
    .replace(/\s+/g, '-') // Replace spaces with hyphens
    .replace(/-+/g, '-') // Replace multiple hyphens with single
    .replace(/^-|-$/g, '') // Remove leading/trailing hyphens
}

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

    // Generate slug from name
    let slug = generateSlug(validatedData.name)
    let slugAttempts = 0
    const maxSlugAttempts = 10

    // Ensure slug uniqueness
    while (slugAttempts < maxSlugAttempts) {
      const existingOrg = await prisma.organization.findFirst({
        where: { slug: slug }
      })

      if (!existingOrg) {
        break
      }

      // Add a number suffix if slug already exists
      slugAttempts++
      slug = `${generateSlug(validatedData.name)}-${slugAttempts}`
    }

    if (slugAttempts >= maxSlugAttempts) {
      return NextResponse.json(
        { error: 'Failed to generate unique organization identifier. Please try again.' },
        { status: 500 }
      )
    }

    // Generate a unique UUID-based namespace
    let namespace: string
    let namespaceAttempts = 0
    const maxAttempts = 10

    do {
      namespace = generateOrganizationNamespace()
      const existingNamespace = await prisma.organization.findUnique({
        where: { namespace }
      })
      
      if (!existingNamespace) {
        break
      }
      
      namespaceAttempts++
    } while (namespaceAttempts < maxAttempts)

    if (namespaceAttempts >= maxAttempts) {
      return NextResponse.json(
        { error: 'Failed to generate unique namespace. Please try again.' },
        { status: 500 }
      )
    }

    // Create organization with user as owner
    const organization = await prisma.organization.create({
      data: {
        name: validatedData.name,
        slug: slug,
        namespace: namespace,
        plan: 'free', // Default to free plan
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
      await k8sClient.createOrganizationNamespace(namespace, organization.id, organization.plan)
      console.log(`Successfully created Kubernetes namespace: ${namespace}`)
    } catch (err: any) {
      // If namespace creation fails, we need to clean up the organization and fail
      console.error('Failed to create organization namespace:', err.message)
      
      // Clean up the created organization from database
      try {
        await prisma.organization.delete({
          where: { id: organization.id }
        })
      } catch (cleanupErr: any) {
        console.error('Failed to cleanup organization after namespace creation failure:', cleanupErr.message)
      }
      
      return NextResponse.json(
        { error: `Failed to create organization infrastructure: ${err.message}` },
        { status: 500 }
      )
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