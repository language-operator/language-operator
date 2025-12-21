import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { k8sClient } from '@/lib/k8s-client'
import { requirePermission } from '@/lib/permissions'

// GET /api/organizations/[id]/quota - Get organization ResourceQuota usage
export async function GET(
  request: NextRequest, 
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const resolvedParams = await params
    const organizationId = resolvedParams.id

    // Get user and verify organization access
    const user = await db.user.findUnique({
      where: { email: session.user.email },
    })

    if (!user) {
      return NextResponse.json({ error: 'User not found' }, { status: 404 })
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organizationId, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Get organization details
    const organization = await db.organization.findUnique({
      where: { id: organizationId },
      select: { id: true, name: true, namespace: true, plan: true }
    })

    if (!organization) {
      return NextResponse.json({ error: 'Organization not found' }, { status: 404 })
    }

    // Get ResourceQuota usage from Kubernetes
    const quotaUsage = await k8sClient.getResourceQuotaUsage(organization.namespace)

    // Calculate warnings (80% threshold)
    const warnings: string[] = []
    Object.entries(quotaUsage.percentUsed).forEach(([resource, percent]) => {
      if (percent >= 80) {
        warnings.push(`${resource}: ${percent.toFixed(1)}% used`)
      }
    })

    return NextResponse.json({
      success: true,
      data: {
        organization: {
          id: organization.id,
          name: organization.name,
          namespace: organization.namespace,
          plan: organization.plan
        },
        quota: quotaUsage.quota,
        used: quotaUsage.used,
        available: quotaUsage.available,
        percentUsed: quotaUsage.percentUsed,
        warnings,
        isNearLimit: warnings.length > 0
      }
    })

  } catch (error) {
    console.error('Error fetching organization quota:', error)
    return NextResponse.json(
      { error: 'Failed to fetch organization quota' },
      { status: 500 }
    )
  }
}

// PUT /api/organizations/[id]/quota - Update organization quota (plan-based or custom)
export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const resolvedParams = await params
    const organizationId = resolvedParams.id
    const body = await request.json()
    const { plan, quotas } = body

    // Must provide either plan OR quotas
    if (!plan && !quotas) {
      return NextResponse.json({
        error: 'Either plan or quotas must be provided'
      }, { status: 400 })
    }

    // Get user and verify organization access
    const user = await db.user.findUnique({
      where: { email: session.user.email },
    })

    if (!user) {
      return NextResponse.json({ error: 'User not found' }, { status: 404 })
    }

    // Check permissions (billing management required for quota changes)
    const hasPermission = await requirePermission(user.id, organizationId, 'manage_billing')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Billing/quota management permissions required' }, { status: 403 })
    }

    // Get organization details
    const organization = await db.organization.findUnique({
      where: { id: organizationId },
      select: { id: true, name: true, namespace: true, plan: true }
    })

    if (!organization) {
      return NextResponse.json({ error: 'Organization not found' }, { status: 404 })
    }

    let newPlan: string

    if (quotas) {
      // Direct quota specification (new approach)
      newPlan = 'custom'

      // Validate quota specification
      const validation = k8sClient.validateQuotaSpec(quotas)
      if (!validation.valid) {
        return NextResponse.json({
          error: 'Invalid quota specification',
          details: validation.errors
        }, { status: 400 })
      }
    } else {
      // Plan-based (legacy approach)
      if (!['free', 'pro', 'enterprise'].includes(plan)) {
        return NextResponse.json({ error: 'Invalid plan' }, { status: 400 })
      }
      newPlan = plan
    }

    // Update database
    await db.organization.update({
      where: { id: organizationId },
      data: { plan: newPlan }
    })

    // Update Kubernetes ResourceQuota
    try {
      if (quotas) {
        // Custom quota specification
        await k8sClient.updateResourceQuotaWithCustomSpec(
          organization.namespace,
          quotas,
          organizationId
        )
      } else {
        // Plan-based quota
        await k8sClient.updateResourceQuota(organization.namespace, plan, organizationId)
      }
    } catch (k8sError: any) {
      console.error('Failed to update Kubernetes ResourceQuota:', k8sError.message)
      // Rollback database change
      await db.organization.update({
        where: { id: organizationId },
        data: { plan: organization.plan }
      })
      return NextResponse.json({
        error: 'Failed to update Kubernetes quotas',
        details: k8sError.message
      }, { status: 500 })
    }

    // Get updated quota usage
    const quotaUsage = await k8sClient.getResourceQuotaUsage(organization.namespace)

    return NextResponse.json({
      success: true,
      data: {
        organization: {
          ...organization,
          plan: newPlan
        },
        quota: quotaUsage.quota,
        used: quotaUsage.used,
        available: quotaUsage.available,
        percentUsed: quotaUsage.percentUsed
      }
    })

  } catch (error) {
    console.error('Error updating organization quota:', error)
    return NextResponse.json(
      { error: 'Failed to update organization quota' },
      { status: 500 }
    )
  }
}