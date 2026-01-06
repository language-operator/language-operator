/**
 * Organization utilities for URL-based organization context
 * Supports the migration from localStorage to RESTful URLs
 */

import { headers } from 'next/headers'
import { db } from '@/lib/db'

export interface OrganizationContext {
  organizationId: string
  userId: string
  userRole?: string
}

/**
 * Extract organization context from request headers (set by middleware)
 * Used in API routes to get validated organization context
 */
export async function getOrganizationContext(): Promise<OrganizationContext | null> {
  try {
    const headersList = await headers()
    const organizationId = headersList.get('x-organization-context')
    const userId = headersList.get('x-user-id')

    if (!organizationId || !userId) {
      return null
    }

    return {
      organizationId,
      userId
    }
  } catch (error) {
    console.error('[OrganizationUtils] Failed to extract organization context:', error)
    return null
  }
}

/**
 * Validate organization access for a user
 * Checks if user is a member of the organization with appropriate role
 */
export async function validateOrganizationAccess(
  userId: string,
  organizationId: string
): Promise<{
  hasAccess: boolean
  userRole?: string
  organization?: any
}> {
  try {
    // Query organization membership
    const membership = await db.organizationMember.findUnique({
      where: {
        organizationId_userId: {
          organizationId,
          userId
        }
      },
      include: {
        organization: true
      }
    })

    if (!membership) {
      return { hasAccess: false }
    }

    return {
      hasAccess: true,
      userRole: membership.role,
      organization: membership.organization
    }
  } catch (error) {
    console.error('[OrganizationUtils] Failed to validate organization access:', error)
    return { hasAccess: false }
  }
}

/**
 * Check if user has specific permission for organization
 * Based on role hierarchy: owner > admin > editor > viewer
 */
export function hasOrganizationPermission(
  userRole: string,
  requiredPermission: 'view' | 'create' | 'edit' | 'delete' | 'manage_members' | 'manage_billing'
): boolean {
  const rolePermissions: Record<string, string[]> = {
    owner: ['view', 'create', 'edit', 'delete', 'manage_members', 'manage_billing'],
    admin: ['view', 'create', 'edit', 'delete', 'manage_members'],
    editor: ['view', 'create', 'edit', 'delete'],
    viewer: ['view']
  }

  const permissions = rolePermissions[userRole] || []
  return permissions.includes(requiredPermission)
}

/**
 * Get organization namespace for Kubernetes operations
 * Maps organization to its dedicated namespace
 */
export function getOrganizationNamespace(organizationId: string, orgSlug?: string): string {
  // Use organization slug if available, otherwise fallback to ID
  // Format: langop-{slug} or langop-{first-8-chars-of-id}
  if (orgSlug) {
    return `langop-${orgSlug}`
  }
  
  const shortId = organizationId.slice(0, 8)
  return `langop-${shortId}`
}

/**
 * Extract organization ID from URL path
 * Supports both API routes (/api/:org_id/*) and frontend routes (/:org_id/*)
 */
export function extractOrgIdFromPath(pathname: string): string | null {
  // Match /api/[org_id]/* pattern
  const apiMatch = pathname.match(/^\/api\/([^\/]+)\//)
  if (apiMatch) {
    return apiMatch[1]
  }

  // Match /[org_id]/* pattern for frontend routes
  const frontendMatch = pathname.match(/^\/([^\/]+)\//)
  if (frontendMatch) {
    const orgId = frontendMatch[1]
    // Skip known non-org routes
    if (['login', 'register', 'accept-invite', '_next', 'api', 'settings'].includes(orgId)) {
      return null
    }
    return orgId
  }

  return null
}

/**
 * Build API URL with organization context
 * Converts from header-based to URL-based organization context
 */
export function buildOrgApiUrl(organizationId: string, path: string): string {
  // Remove leading slash if present
  const cleanPath = path.startsWith('/') ? path.slice(1) : path
  
  // Remove /api prefix if present (will be added back)
  const apiPath = cleanPath.startsWith('api/') ? cleanPath.slice(4) : cleanPath
  
  return `/api/${organizationId}/${apiPath}`
}

/**
 * Legacy helper: Get organization ID from request headers
 * Supports backward compatibility during migration
 */
export async function getLegacyOrgId(): Promise<string | null> {
  try {
    const headersList = await headers()
    // Check new context first, then fall back to legacy header
    return headersList.get('x-organization-context') || 
           headersList.get('x-organization-id') || 
           null
  } catch (error) {
    console.error('[OrganizationUtils] Failed to extract legacy org ID:', error)
    return null
  }
}

/**
 * Error response helpers for organization-related failures
 */
export class OrganizationError extends Error {
  constructor(
    message: string,
    public code: 'ORGANIZATION_NOT_FOUND' | 'ACCESS_DENIED' | 'INVALID_ORGANIZATION_ID',
    public statusCode: number = 403
  ) {
    super(message)
    this.name = 'OrganizationError'
  }
}

export function createOrganizationErrorResponse(error: OrganizationError) {
  return Response.json(
    {
      error: error.message,
      code: error.code
    },
    { status: error.statusCode }
  )
}