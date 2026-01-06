import { NextRequest, NextResponse } from 'next/server'
import { getOrganizationContext } from '@/lib/organization-utils'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'

// GET /api/[org_id]/dashboard/counts - Get resource counts for dashboard
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ org_id: string }> }
) {
  try {
    const resolvedParams = await params
    const urlOrgId = resolvedParams.org_id

    // Get session for user authentication
    const session = await getServerSession(authOptions)
    if (!session?.user?.id) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Try to get organization context from middleware first, fallback to manual lookup
    let organizationId: string | undefined
    let userId: string | undefined

    try {
      const context = await getOrganizationContext()
      if (context) {
        organizationId = context.organizationId
        userId = context.userId
      }
    } catch (error) {
      console.warn('Middleware context not available, using fallback')
    }

    // Fallback: use URL org ID and session user ID
    if (!organizationId || !userId) {
      organizationId = urlOrgId
      userId = session.user.id
    }

    // Verify URL org ID matches resolved organization ID
    if (urlOrgId !== organizationId) {
      return NextResponse.json({ 
        error: 'Organization ID mismatch' 
      }, { status: 400 })
    }

    // Get organization details and verify access
    const membership = await db.organizationMember.findUnique({
      where: {
        organizationId_userId: {
          organizationId: organizationId,
          userId: userId
        }
      },
      include: {
        organization: {
          select: { id: true, name: true, namespace: true, plan: true }
        }
      }
    })

    if (!membership) {
      return NextResponse.json({ error: 'Organization not found or access denied' }, { status: 404 })
    }

    const organization = membership.organization

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