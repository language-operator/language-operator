import { NextRequest, NextResponse } from 'next/server'
import { k8sClient } from '@/lib/k8s-client'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { createErrorResponse, createSuccessResponse, handleKubernetesOperation, validateClusterNameFormat, createAuthenticationRequiredError, createPermissionDeniedError } from '@/lib/api-error-handler'
import { validateClusterExists } from '@/lib/cluster-validation'

// GET /api/clusters/[name]/tools/[toolName] - Get a specific tool in a cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; toolName: string }> }
) {
  try {
    const { name: clusterName, toolName } = await params
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      throw createPermissionDeniedError('view tool', 'cluster-scoped tool', userRole)
    }

    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    // Get the specific tool
    const response = await handleKubernetesOperation(
      'get tool',
      k8sClient.getLanguageTool(organization.namespace, toolName)
    )
    
    if (!response) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }
    
    return createSuccessResponse(response, undefined, { cluster: clusterName })
    
  } catch (error) {
    console.error('Error fetching cluster tool:', error)
    return createErrorResponse(error, 'Failed to fetch tool')
  }
}

// PUT /api/clusters/[name]/tools/[toolName] - Update a specific tool in a cluster
export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; toolName: string }> }
) {
  try {
    const { name: clusterName, toolName } = await params
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      throw createPermissionDeniedError('edit tool', 'cluster-scoped tool', userRole)
    }

    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    // Parse request body
    let updateData
    try {
      updateData = await request.json()
    } catch {
      return createErrorResponse(new Error('Invalid JSON in request body'), 'Invalid JSON in request body')
    }

    // Get existing tool to validate it exists and belongs to the cluster
    const existingTool = await handleKubernetesOperation(
      'get existing tool',
      k8sClient.getLanguageTool(organization.namespace, toolName)
    )
    
    if (!existingTool) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }

    // Verify tool belongs to the specified cluster
    if (existingTool.spec.clusterRef !== clusterName) {
      return NextResponse.json({ 
        error: `Tool ${toolName} does not belong to cluster ${clusterName}` 
      }, { status: 400 })
    }

    // Build update payload focusing on editable fields
    const updatedTool = {
      ...existingTool,
      spec: {
        ...existingTool.spec,
        // Update container image if provided
        ...(updateData.image && { image: updateData.image }),
        
        // Update resource limits
        ...(updateData.resources && {
          resources: {
            requests: {
              cpu: updateData.resources.cpu || existingTool.spec.resources?.requests?.cpu || '100m',
              memory: updateData.resources.memory || existingTool.spec.resources?.requests?.memory || '128Mi'
            },
            limits: {
              cpu: updateData.resources.cpuLimit || existingTool.spec.resources?.limits?.cpu || '500m',
              memory: updateData.resources.memoryLimit || existingTool.spec.resources?.limits?.memory || '512Mi'
            }
          }
        }),
        
        // Update port configuration
        ...(updateData.port && {
          port: updateData.port
        }),
        
        // Update deployment mode
        ...(updateData.deploymentMode && {
          deploymentMode: updateData.deploymentMode
        }),
        
        // Update environment variables
        ...(updateData.envVars && {
          env: updateData.envVars.map((envVar: { key: string; value: string }) => ({
            name: envVar.key,
            value: envVar.value
          }))
        })
      }
    }

    // Update the tool via Kubernetes API using replace (not patch)
    const updatedResult = await handleKubernetesOperation(
      'update tool',
      k8sClient.replaceLanguageTool(organization.namespace, toolName, updatedTool)
    )

    // Log the update for audit trail
    console.log(`Tool updated: ${toolName} in cluster ${clusterName} by ${user.email} in namespace ${organization.namespace}`)

    return createSuccessResponse(updatedResult, undefined, { cluster: clusterName })
    
  } catch (error) {
    console.error('Error updating cluster tool:', error)
    return createErrorResponse(error, 'Failed to update tool')
  }
}

// DELETE /api/clusters/[name]/tools/[toolName] - Delete a specific tool in a cluster
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; toolName: string }> }
) {
  try {
    const { name: clusterName, toolName } = await params
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    const hasPermission = await requirePermission(user.id, organization.id, 'delete')
    if (!hasPermission) {
      throw createPermissionDeniedError('delete tool', 'cluster-scoped tool', userRole)
    }

    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    // Check if tool exists
    const existingTool = await handleKubernetesOperation(
      'get existing tool',
      k8sClient.getLanguageTool(organization.namespace, toolName)
    )
    
    if (!existingTool) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }

    // Delete the tool
    await handleKubernetesOperation(
      'delete tool',
      k8sClient.deleteLanguageTool(organization.namespace, toolName)
    )

    // Log the deletion for audit trail
    console.log(`Tool deleted: ${toolName} in cluster ${clusterName} by ${user.email} in namespace ${organization.namespace}`)

    return createSuccessResponse(null, undefined, { cluster: clusterName })
    
  } catch (error) {
    console.error('Error deleting cluster tool:', error)
    return createErrorResponse(error, 'Failed to delete tool')
  }
}