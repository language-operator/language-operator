import { NextRequest, NextResponse } from 'next/server'
import { hash } from 'bcryptjs'
import { db } from '@/lib/db'
import { k8sClient } from '@/lib/k8s-client'
import { generateOrganizationNamespace } from '@/lib/namespace-utils'
import { isSignupsDisabled } from '@/lib/env'
import { isSignupAllowed } from '@/lib/invitation-utils'

export async function POST(req: NextRequest) {
  try {
    const { name, email, password, callbackUrl } = await req.json()

    // Check if signup is allowed
    const signupAllowed = await isSignupAllowed(isSignupsDisabled, callbackUrl)
    if (!signupAllowed) {
      return NextResponse.json(
        { error: 'Signup is currently disabled. Please use an invitation link to create an account.' },
        { status: 403 }
      )
    }

    // Validate input
    if (!name || !email || !password) {
      return NextResponse.json(
        { error: 'Missing required fields' },
        { status: 400 }
      )
    }

    if (password.length < 8) {
      return NextResponse.json(
        { error: 'Password must be at least 8 characters' },
        { status: 400 }
      )
    }

    // Check if user already exists
    const existingUser = await db.user.findUnique({
      where: { email },
    })

    if (existingUser) {
      return NextResponse.json(
        { error: 'User already exists' },
        { status: 400 }
      )
    }

    // Hash password
    const hashedPassword = await hash(password, 12)

    // Create user
    const user = await db.user.create({
      data: {
        name,
        email,
        password: hashedPassword,
      },
    })

    // Create default organization for the user
    const orgSlug = email.split('@')[0].toLowerCase().replace(/[^a-z0-9-]/g, '-')
    
    // Generate a unique UUID-based namespace
    let namespace: string
    let namespaceAttempts = 0
    const maxAttempts = 10

    do {
      namespace = generateOrganizationNamespace()
      const existingNamespace = await db.organization.findUnique({
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

    // Create organization
    const organization = await db.organization.create({
      data: {
        name: `${name}'s Organization`,
        slug: orgSlug,
        namespace,
        plan: 'free',
        members: {
          create: {
            userId: user.id,
            role: 'owner',
          },
        },
      },
    })

    // Create Kubernetes namespace with ResourceQuota for the organization
    try {
      await k8sClient.createOrganizationNamespace(namespace, organization.id, 'free')
      console.log(`Successfully created Kubernetes namespace: ${namespace}`)
    } catch (err: any) {
      // If namespace creation fails, we need to clean up the user and organization and fail signup
      console.error('Failed to create organization namespace:', err.message)
      
      // Clean up the created organization and user from database
      try {
        await db.organization.delete({
          where: { id: organization.id }
        })
        await db.user.delete({
          where: { id: user.id }
        })
      } catch (cleanupErr: any) {
        console.error('Failed to cleanup after namespace creation failure:', cleanupErr.message)
      }
      
      return NextResponse.json(
        { error: `Failed to create account infrastructure: ${err.message}` },
        { status: 500 }
      )
    }

    return NextResponse.json(
      {
        user: {
          id: user.id,
          name: user.name,
          email: user.email,
        },
        organization: {
          id: organization.id,
          name: organization.name,
          namespace,
        },
      },
      { status: 201 }
    )
  } catch (error: any) {
    console.error('Signup error:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}
