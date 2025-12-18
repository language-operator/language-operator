import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth/next'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { z } from 'zod'
import bcrypt from 'bcryptjs'

const createUserSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  email: z.string().email('Invalid email address'),
  password: z.string().min(6, 'Password must be at least 6 characters'),
})

export async function GET(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // First, get the current user's organization memberships to determine which organizations they can see users for
    const currentUserMemberships = await prisma.organizationMember.findMany({
      where: {
        user: {
          email: session.user.email
        }
      },
      select: {
        organizationId: true
      }
    })

    if (currentUserMemberships.length === 0) {
      return NextResponse.json({ error: 'User not found in any organization' }, { status: 403 })
    }

    const organizationIds = currentUserMemberships.map(m => m.organizationId)

    // Get only users that belong to the same organizations as the current user
    const users = await prisma.user.findMany({
      where: {
        memberships: {
          some: {
            organizationId: {
              in: organizationIds
            }
          }
        }
      },
      select: {
        id: true,
        name: true,
        email: true,
        image: true,
        createdAt: true,
        updatedAt: true,
        memberships: {
          where: {
            organizationId: {
              in: organizationIds
            }
          },
          include: {
            organization: {
              select: {
                id: true,
                name: true,
                slug: true,
                namespace: true
              }
            }
          }
        },
        sessions: {
          select: {
            expires: true
          },
          orderBy: {
            expires: 'desc'
          },
          take: 1
        }
      },
      orderBy: {
        createdAt: 'desc'
      }
    })

    // Transform the data to include status and last seen
    const usersWithStatus = users.map(user => {
      const lastSession = user.sessions[0]
      const isActive = lastSession && lastSession.expires > new Date()
      
      return {
        id: user.id,
        name: user.name || 'Unnamed User',
        email: user.email,
        image: user.image,
        status: isActive ? 'active' : 'inactive',
        lastSeen: lastSession ? lastSession.expires : user.createdAt,
        createdAt: user.createdAt,
        updatedAt: user.updatedAt,
        memberships: user.memberships.map(membership => ({
          organizationId: membership.organization.id,
          organizationName: membership.organization.name,
          role: membership.role
        }))
      }
    })

    return NextResponse.json(usersWithStatus)
  } catch (error) {
    console.error('Error fetching users:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const body = await request.json()
    const validatedData = createUserSchema.parse(body)

    // Hash the password
    const hashedPassword = await bcrypt.hash(validatedData.password, 10)

    // Get the current user's organization membership to determine which org to add the new user to
    const currentUserMembership = await prisma.organizationMember.findFirst({
      where: { 
        userId: session.user.id 
      },
      include: {
        organization: true
      }
    })

    if (!currentUserMembership) {
      return NextResponse.json({ 
        error: 'You must be a member of an organization to add users' 
      }, { status: 400 })
    }

    // Check if user already exists
    const existingUser = await prisma.user.findUnique({
      where: { email: validatedData.email }
    })

    if (existingUser) {
      return NextResponse.json({ 
        error: 'User with this email already exists' 
      }, { status: 400 })
    }

    // Create user and organization membership in a transaction
    const result = await prisma.$transaction(async (tx) => {
      // Create the user
      const newUser = await tx.user.create({
        data: {
          name: validatedData.name,
          email: validatedData.email,
          // Note: NextAuth typically doesn't use a password field directly
          // You may need to adjust this based on your auth setup
          // For now, we'll store it as a custom field
        }
      })

      // Add user to the current organization with 'viewer' role by default
      await tx.organizationMember.create({
        data: {
          userId: newUser.id,
          organizationId: currentUserMembership.organizationId,
          role: 'viewer'
        }
      })

      return newUser
    })

    // Return the created user with membership info
    const userWithMembership = await prisma.user.findUnique({
      where: { id: result.id },
      select: {
        id: true,
        name: true,
        email: true,
        image: true,
        createdAt: true,
        updatedAt: true,
        memberships: {
          include: {
            organization: {
              select: {
                id: true,
                name: true
              }
            }
          }
        }
      }
    })

    if (!userWithMembership) {
      return NextResponse.json({ error: 'User created but not found' }, { status: 500 })
    }

    // Transform response
    const userData = {
      id: userWithMembership.id,
      name: userWithMembership.name || 'Unnamed User',
      email: userWithMembership.email,
      image: userWithMembership.image,
      status: 'inactive', // New users start as inactive until they log in
      lastSeen: userWithMembership.createdAt,
      createdAt: userWithMembership.createdAt,
      updatedAt: userWithMembership.updatedAt,
      memberships: userWithMembership.memberships.map(membership => ({
        organizationId: membership.organization.id,
        organizationName: membership.organization.name,
        role: membership.role
      }))
    }

    return NextResponse.json({ 
      user: userData,
      password: validatedData.password // Return the plain password for display
    }, { status: 201 })

  } catch (error) {
    if (error instanceof z.ZodError) {
      return NextResponse.json(
        { error: 'Invalid input', details: error.issues },
        { status: 400 }
      )
    }

    console.error('Error creating user:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}