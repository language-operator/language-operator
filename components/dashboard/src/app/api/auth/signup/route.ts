import { NextRequest, NextResponse } from 'next/server'
import { hash } from 'bcryptjs'
import { db } from '@/lib/db'
import { k8sClient } from '@/lib/k8s-client'

export async function POST(req: NextRequest) {
  try {
    const { name, email, password } = await req.json()

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
    const namespace = `org-${orgSlug}`

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
    } catch (err: any) {
      // If namespace creation fails, log but don't fail signup
      // (namespace might already exist)
      console.error('Failed to create organization namespace:', err.message)
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
