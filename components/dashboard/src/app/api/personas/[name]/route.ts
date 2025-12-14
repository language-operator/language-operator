import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { z } from 'zod'

const updatePersonaSchema = z.object({
  role: z.string().optional(),
  description: z.string().optional(),
  spec: z.object({
    role: z.string().optional(),
    systemPrompt: z.string().min(20).optional(),
    traits: z.array(z.string()).min(1).optional(),
    examples: z.array(z.object({
      input: z.string(),
      output: z.string()
    })).optional(),
    parameters: z.object({
      temperature: z.number().min(0).max(2).optional(),
      maxTokens: z.number().int().min(1).max(8192).optional(),
    }).optional(),
    enabled: z.boolean().optional(),
    requireApproval: z.boolean().optional(),
  }).optional()
})

// GET /api/personas/[name] - Get a specific persona
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized - no organization' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace
    const persona = await k8sClient.getLanguagePersona(namespace, name)
    
    if (!persona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }
    
    return NextResponse.json({ persona })
  } catch (error) {
    console.error('Error fetching persona:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

// PATCH /api/personas/[name] - Update a specific persona
export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized - no organization' }, { status: 401 })
    }

    const body = await request.json()
    const validatedData = updatePersonaSchema.parse(body)
    
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace

    // Get existing persona
    const existingPersona = await k8sClient.getLanguagePersona(namespace, name)
    if (!existingPersona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }

    // Update the persona
    const updatedPersona = await k8sClient.updateLanguagePersona(name, namespace, {
      metadata: {
        ...existingPersona.metadata,
        annotations: {
          ...existingPersona.metadata.annotations,
          'langop.io/updated-at': new Date().toISOString(),
          'langop.io/updated-by': session.user.email || 'unknown'
        }
      },
      spec: {
        ...existingPersona.spec,
        ...validatedData.spec,
        role: validatedData.role || validatedData.spec?.role || existingPersona.spec.role,
      }
    })

    // Log the update for audit trail
    console.log(`Persona updated: ${name} by ${session.user.email} in ${namespace}`)

    return NextResponse.json({ persona: updatedPersona })
  } catch (error) {
    console.error('Error updating persona:', error)
    
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

// DELETE /api/personas/[name] - Delete a specific persona
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized - no organization' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace

    // Check if persona exists
    const existingPersona = await k8sClient.getLanguagePersona(namespace, name)
    if (!existingPersona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }

    // Delete the persona
    await k8sClient.deleteLanguagePersona(name, namespace)

    // Log the deletion for audit trail
    console.log(`Persona deleted: ${name} by ${session.user.email} in ${namespace}`)

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting persona:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}