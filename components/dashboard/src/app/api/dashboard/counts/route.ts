import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

// GET /api/dashboard/counts - Get resource counts for dashboard
export async function GET(request: NextRequest) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Get resource counts from Kubernetes
    // Filter by organization ID to ensure consistent counts with individual resource pages
    const counts = await k8sClient.getNamespaceResourceCounts(
      organization.namespace,
      organization.id
    )

    // Get quota usage information
    let quotaUsage = null
    try {
      quotaUsage = await k8sClient.getResourceQuotaUsage(organization.namespace)
    } catch (quotaError) {
      console.error('Failed to fetch quota usage:', quotaError)
      // Continue without quota info
    }

    return NextResponse.json({
      success: true,
      data: {
        ...counts,
        quota: quotaUsage
      },
    })

  } catch (error) {
    console.error('Error fetching dashboard counts:', error)
    return NextResponse.json(
      { error: 'Failed to fetch dashboard counts' },
      { status: 500 }
    )
  }
}