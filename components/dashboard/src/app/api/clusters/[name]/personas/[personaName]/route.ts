import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { validateClusterExists } from '@/lib/cluster-validation'
import {
  createErrorResponse,
  createSuccessResponse,
  handleKubernetesOperation,
  validateClusterNameFormat,
  createAuthenticationRequiredError,
  createPermissionDeniedError
} from '@/lib/api-error-handler'

// GET /api/clusters/[name]/personas/[personaName] - Get a specific persona in a cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      throw createPermissionDeniedError('view persona', 'cluster-scoped personas', userRole)
    }

    const { name: clusterName, personaName } = await params

    // Validate cluster name format
    validateClusterNameFormat(clusterName)

    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    console.log(`Fetching persona ${personaName} for cluster ${clusterName} from namespace:`, organization.namespace)

    // Fetch the specific persona from Kubernetes
    const response = await handleKubernetesOperation(
      'get persona',
      k8sClient.getLanguagePersona(organization.namespace, personaName)
    )

    // Extract persona data from response
    let persona = null
    if (response.body && typeof response.body === 'object') {
      persona = response.body
    } else if (response.data && typeof response.data === 'object') {
      persona = response.data
    } else {
      persona = response
    }

    if (!persona) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' not found`),
        'Persona not found'
      )
    }

    // Verify the persona belongs to this cluster (check clusterRef)
    if (persona.spec?.clusterRef !== clusterName) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' does not belong to cluster '${clusterName}'`),
        'Persona not found in cluster'
      )
    }

    console.log(`Successfully fetched persona ${personaName} for cluster ${clusterName}`)

    return createSuccessResponse({ persona }, undefined, {
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching cluster persona:', error)
    return createErrorResponse(error, 'Failed to fetch cluster persona')
  }
}

// PATCH /api/clusters/[name]/personas/[personaName] - Update a specific persona in a cluster
export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      throw createPermissionDeniedError('edit persona', 'cluster-scoped personas', userRole)
    }

    const { name: clusterName, personaName } = await params

    // Validate cluster name format
    validateClusterNameFormat(clusterName)

    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    const body = await request.json()

    console.log(`Updating persona ${personaName} for cluster ${clusterName} in namespace:`, organization.namespace)

    // First, get the existing persona to merge with updates
    const existingResponse = await handleKubernetesOperation(
      'get persona for update',
      k8sClient.getLanguagePersona(organization.namespace, personaName)
    )

    let existingPersona = null
    if (existingResponse.body && typeof existingResponse.body === 'object') {
      existingPersona = existingResponse.body
    } else if (existingResponse.data && typeof existingResponse.data === 'object') {
      existingPersona = existingResponse.data
    } else {
      existingPersona = existingResponse
    }

    if (!existingPersona) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' not found`),
        'Persona not found'
      )
    }

    // Verify the persona belongs to this cluster
    if (existingPersona.spec?.clusterRef !== clusterName) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' does not belong to cluster '${clusterName}'`),
        'Persona not found in cluster'
      )
    }

    // Merge the updates with existing persona spec
    const updatedPersona = {
      ...existingPersona,
      spec: {
        ...existingPersona.spec,
        ...(body.displayName !== undefined && { displayName: body.displayName }),
        ...(body.description !== undefined && { description: body.description }),
        ...(body.systemPrompt !== undefined && { systemPrompt: body.systemPrompt }),
        ...(body.tone !== undefined && { tone: body.tone }),
        ...(body.language !== undefined && { language: body.language }),
        ...(body.instructions !== undefined && { instructions: body.instructions }),
        // Ensure clusterRef is preserved
        clusterRef: clusterName,
      }
    }

    // Update the persona in Kubernetes
    const response = await handleKubernetesOperation(
      'update persona',
      k8sClient.updateLanguagePersona(organization.namespace, personaName, updatedPersona)
    )

    console.log(`Successfully updated persona ${personaName} for cluster ${clusterName}`)

    return createSuccessResponse(response, 'Persona updated successfully')

  } catch (error) {
    console.error('Error updating cluster persona:', error)
    return createErrorResponse(error, 'Failed to update cluster persona')
  }
}

// DELETE /api/clusters/[name]/personas/[personaName] - Delete a specific persona in a cluster
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'delete')
    if (!hasPermission) {
      throw createPermissionDeniedError('delete persona', 'cluster-scoped personas', userRole)
    }

    const { name: clusterName, personaName } = await params

    // Validate cluster name format
    validateClusterNameFormat(clusterName)

    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    console.log(`Deleting persona ${personaName} for cluster ${clusterName} from namespace:`, organization.namespace)

    // First verify the persona belongs to this cluster
    const existingResponse = await handleKubernetesOperation(
      'get persona for deletion',
      k8sClient.getLanguagePersona(organization.namespace, personaName)
    )

    let existingPersona = null
    if (existingResponse.body && typeof existingResponse.body === 'object') {
      existingPersona = existingResponse.body
    } else if (existingResponse.data && typeof existingResponse.data === 'object') {
      existingPersona = existingResponse.data
    } else {
      existingPersona = existingResponse
    }

    if (!existingPersona) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' not found`),
        'Persona not found'
      )
    }

    // Verify the persona belongs to this cluster
    if (existingPersona.spec?.clusterRef !== clusterName) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' does not belong to cluster '${clusterName}'`),
        'Persona not found in cluster'
      )
    }

    // Delete the persona from Kubernetes
    const response = await handleKubernetesOperation(
      'delete persona',
      k8sClient.deleteLanguagePersona(organization.namespace, personaName)
    )

    console.log(`Successfully deleted persona ${personaName} for cluster ${clusterName}`)

    return createSuccessResponse(response, 'Persona deleted successfully')

  } catch (error) {
    console.error('Error deleting cluster persona:', error)
    return createErrorResponse(error, 'Failed to delete cluster persona')
  }
}
