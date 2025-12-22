import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

// GET /api/clusters/[name]/personas/[personaName] - Get a specific persona in a cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    const { name: clusterName, personaName } = await params
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    if (!clusterName || !personaName) {
      return NextResponse.json({ error: 'Cluster name and persona name are required' }, { status: 400 })
    }

    // Get the specific persona
    const persona = await k8sClient.getLanguagePersona(organization.namespace, personaName)

    if (!persona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }

    return NextResponse.json({
      persona,
      cluster: clusterName
    })

  } catch (error) {
    console.error('Error fetching cluster persona:', error)
    return NextResponse.json(
      {
        error: 'Failed to fetch persona',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

// PATCH /api/clusters/[name]/personas/[personaName] - Update a specific persona in a cluster
export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    const { name: clusterName, personaName } = await params
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    if (!clusterName || !personaName) {
      return NextResponse.json({ error: 'Cluster name and persona name are required' }, { status: 400 })
    }

    // Check if persona exists
    const existingPersona = await k8sClient.getLanguagePersona(organization.namespace, personaName)
    if (!existingPersona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }

    const body = await request.json()

    // Update the persona spec
    const updatedSpec = {
      ...existingPersona.spec,
      ...(body.displayName !== undefined && { displayName: body.displayName }),
      ...(body.description !== undefined && { description: body.description }),
      ...(body.systemPrompt !== undefined && { systemPrompt: body.systemPrompt }),
      ...(body.tone !== undefined && { tone: body.tone }),
      ...(body.language !== undefined && { language: body.language }),
      ...(body.instructions !== undefined && { instructions: body.instructions }),
    }

    const updatedPersona = {
      ...existingPersona,
      spec: updatedSpec
    }

    // Update the persona
    const result = await k8sClient.updateLanguagePersona(organization.namespace, personaName, updatedPersona)

    // Log the update for audit trail
    console.log(`Persona updated: ${personaName} in cluster ${clusterName} by ${user.email} in namespace ${organization.namespace}`)

    return NextResponse.json({
      persona: result,
      cluster: clusterName
    })

  } catch (error) {
    console.error('Error updating cluster persona:', error)
    return NextResponse.json(
      {
        error: 'Failed to update persona',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

// DELETE /api/clusters/[name]/personas/[personaName] - Delete a specific persona in a cluster
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    const { name: clusterName, personaName } = await params
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    const hasPermission = await requirePermission(user.id, organization.id, 'delete')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    if (!clusterName || !personaName) {
      return NextResponse.json({ error: 'Cluster name and persona name are required' }, { status: 400 })
    }

    // Check if persona exists
    const existingPersona = await k8sClient.getLanguagePersona(organization.namespace, personaName)
    if (!existingPersona) {
      return NextResponse.json({ error: 'Persona not found' }, { status: 404 })
    }

    // Delete the persona
    await k8sClient.deleteLanguagePersona(organization.namespace, personaName)

    // Log the deletion for audit trail
    console.log(`Persona deleted: ${personaName} in cluster ${clusterName} by ${user.email} in namespace ${organization.namespace}`)

    return NextResponse.json({
      success: true,
      cluster: clusterName
    })

  } catch (error) {
    console.error('Error deleting cluster persona:', error)
    return NextResponse.json(
      {
        error: 'Failed to delete persona',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}
