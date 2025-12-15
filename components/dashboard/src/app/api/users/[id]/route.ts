import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth/next'
import { authOptions } from '@/lib/auth'
import { prisma } from '@/lib/prisma'
import { z } from 'zod'

const updateUserSchema = z.object({
  name: z.string().optional(),
  email: z.string().email().optional(),
  memberships: z.array(z.object({
    organizationId: z.string(),
    organizationName: z.string(),
    role: z.enum(['owner', 'admin', 'editor', 'viewer'])
  })).optional(),
  updateMemberships: z.boolean().optional()
})

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> | { id: string } }
) {
  try {
    const session = await getServerSession(authOptions)
    
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Await params if it's a Promise (Next.js 15+)
    const resolvedParams = await params
    const userId = resolvedParams.id

    const user = await prisma.user.findUnique({
      where: { id: userId },
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
      }
    })

    if (!user) {
      return NextResponse.json({ error: 'User not found' }, { status: 404 })
    }

    // Transform the data
    const lastSession = user.sessions[0]
    const isActive = lastSession && lastSession.expires > new Date()
    
    const userData = {
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

    return NextResponse.json(userData)
  } catch (error) {
    console.error('Error fetching user:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> | { id: string } }
) {
  try {
    const session = await getServerSession(authOptions)
    
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Await params if it's a Promise (Next.js 15+)
    const resolvedParams = await params
    const userId = resolvedParams.id
    
    console.log('User ID from params:', userId)

    const body = await request.json()
    console.log('PATCH Request Body:', JSON.stringify(body, null, 2))
    
    const validatedData = updateUserSchema.parse(body)
    console.log('Validated Data:', JSON.stringify(validatedData, null, 2))

    // Update user basic info
    const updateData: any = {}
    if (validatedData.name) updateData.name = validatedData.name
    if (validatedData.email) updateData.email = validatedData.email
    console.log('Update Data:', JSON.stringify(updateData, null, 2))

    // Start transaction
    const result = await prisma.$transaction(async (tx) => {
      console.log('Starting transaction for user:', userId)
      
      // Update user basic information
      const updatedUser = await tx.user.update({
        where: { id: userId },
        data: updateData,
        select: {
          id: true,
          name: true,
          email: true,
          image: true,
          updatedAt: true
        }
      })
      console.log('User updated successfully:', updatedUser)

      // Handle memberships only if explicitly requested
      if (validatedData.updateMemberships && validatedData.memberships !== undefined) {
        console.log('Processing memberships:', validatedData.memberships.length)
        
        // Remove all existing memberships
        const deleteResult = await tx.organizationMember.deleteMany({
          where: { userId: userId }
        })
        console.log('Deleted existing memberships:', deleteResult.count)

        // Add new memberships
        if (validatedData.memberships.length > 0) {
          const membershipData = validatedData.memberships.map(membership => ({
            userId: userId,
            organizationId: membership.organizationId,
            role: membership.role
          }))
          console.log('Creating new memberships:', membershipData)
          
          await tx.organizationMember.createMany({
            data: membershipData
          })
          console.log('New memberships created successfully')
        }
      } else {
        console.log('Skipping membership updates - only updating basic user info')
      }

      return updatedUser
    })

    // Fetch updated user with memberships
    const fullUser = await prisma.user.findUnique({
      where: { id: userId },
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
      }
    })

    if (!fullUser) {
      return NextResponse.json({ error: 'User not found after update' }, { status: 404 })
    }

    // Transform response
    const lastSession = fullUser.sessions[0]
    const isActive = lastSession && lastSession.expires > new Date()
    
    const userData = {
      id: fullUser.id,
      name: fullUser.name || 'Unnamed User',
      email: fullUser.email,
      image: fullUser.image,
      status: isActive ? 'active' : 'inactive',
      lastSeen: lastSession ? lastSession.expires : fullUser.createdAt,
      createdAt: fullUser.createdAt,
      updatedAt: fullUser.updatedAt,
      memberships: fullUser.memberships.map(membership => ({
        organizationId: membership.organization.id,
        organizationName: membership.organization.name,
        role: membership.role
      }))
    }

    return NextResponse.json(userData)
  } catch (error) {
    if (error instanceof z.ZodError) {
      console.error('Zod validation error:', error.issues)
      return NextResponse.json(
        { error: 'Invalid input', details: error.issues },
        { status: 400 }
      )
    }

    console.error('Error updating user:', error)
    console.error('Error stack:', error instanceof Error ? error.stack : 'No stack available')
    console.error('Error name:', error instanceof Error ? error.name : 'Unknown error')
    console.error('Error message:', error instanceof Error ? error.message : String(error))
    
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}