import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'

// GET /api/dashboard/counts - Get resource counts for dashboard
export async function GET(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Get user's current organization
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { 
        memberships: { 
          include: { organization: true } 
        } 
      },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    // Use the first organization (in a real app, you'd have org selection logic)
    const organization = user.memberships[0].organization
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Get resource counts from Kubernetes
    // Note: Don't filter by organization ID to match the behavior of /api/clusters
    // which shows all clusters in the namespace regardless of organization labels
    const counts = await k8sClient.getNamespaceResourceCounts(
      organization.namespace
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