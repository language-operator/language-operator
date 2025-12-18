import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { filterByClusterRef } from '@/lib/cluster-utils'
import { validateClusterAccess, getClusterResourceCounts } from '@/lib/cluster-validation'
import { createErrorResponse, createSuccessResponse, validateClusterNameFormat, createAuthenticationRequiredError, createPermissionDeniedError } from '@/lib/api-error-handler'

// GET /api/clusters/[name]/counts - Get resource counts for a specific cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: clusterName } = await params

    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      throw createPermissionDeniedError('view cluster counts', 'cluster-scoped resource counts', userRole)
    }

    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster access with comprehensive validation
    await validateClusterAccess(organization.namespace, clusterName, organization.id, userRole)

    // Get cluster-specific resource counts from Kubernetes with error handling
    console.log(`Getting resource counts for cluster ${clusterName} in namespace ${organization.namespace}`)
    
    // Use the comprehensive cluster resource counting utility
    const clusterCounts = await getClusterResourceCounts(
      organization.namespace,
      clusterName,
      organization.id
    )

    return createSuccessResponse(clusterCounts)

  } catch (error) {
    console.error('Error fetching cluster resource counts:', error)
    return createErrorResponse(error, 'Failed to fetch cluster resource counts')
  }
}

