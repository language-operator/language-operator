import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { z } from 'zod'

const updatePersonaSchema = z.object({
  spec: z.object({
    description: z.string().optional(),
    displayName: z.string().optional(),
    systemPrompt: z.string().optional(),
    tone: z.string().optional(),
    language: z.string().optional(),
    version: z.string().optional(),
    capabilities: z.array(z.string()).optional(),
    limitations: z.array(z.string()).optional(),
    instructions: z.array(z.string()).optional(),
    examples: z.array(z.object({
      input: z.string().min(1, "Example input is required"),
      output: z.string().min(1, "Example output is required"),
      context: z.string().optional(),
      tags: z.array(z.string()).optional(),
    })).optional(),
    constraints: z.array(z.string()).optional(),
    vocabulary: z.object({
      preferred: z.array(z.string()).optional(),
      forbidden: z.array(z.string()).optional(),
    }).optional(),
    responseFormat: z.object({
      structure: z.enum(['freeform', 'structured', 'json', 'markdown']).optional(),
      maxLength: z.number().int().min(1).max(10000).optional(),
      includeMetadata: z.boolean().optional(),
    }).optional(),
  })
})

// GET /api/personas/[name] - Get a specific persona
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
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
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    const namespace = organization.namespace

    // Parse and validate request body
    const body = await request.json()
    const validatedData = updatePersonaSchema.parse(body)

    // Get existing persona
    console.log('Fetching existing persona:', name, 'in namespace:', namespace)
    const existingPersona = await k8sClient.getLanguagePersona(namespace, name)
    if (!existingPersona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }
    console.log('Existing persona:', JSON.stringify(existingPersona, null, 2))

    // Build update payload
    const updatePayload = {
      metadata: {
        ...existingPersona.metadata,
        annotations: {
          ...existingPersona.metadata.annotations,
          'langop.io/updated-at': new Date().toISOString(),
          'langop.io/updated-by': user.email || 'unknown'
        }
      },
      spec: {
        ...existingPersona.spec,
        ...validatedData.spec,
      }
    }
    console.log('Update payload:', JSON.stringify(updatePayload, null, 2))

    // Update the persona
    const updatedPersona = await k8sClient.updateLanguagePersona(namespace, name, updatePayload)

    // Log the update for audit trail
    console.log(`Persona updated: ${name} by ${user.email} in ${namespace}`)

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
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    const namespace = organization.namespace

    // Check if persona exists
    const existingPersona = await k8sClient.getLanguagePersona(namespace, name)
    if (!existingPersona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }

    // Delete the persona
    await k8sClient.deleteLanguagePersona(namespace, name)

    // Log the deletion for audit trail
    console.log(`Persona deleted: ${name} by ${user.email} in ${namespace}`)

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting persona:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}