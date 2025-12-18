/**
 * Organization Context Utilities
 * 
 * Provides clean utilities for handling organization selection between
 * frontend and backend, replacing the broken memberships[0] pattern.
 */

import { NextRequest } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'

export const ORG_HEADER = 'x-organization-id'

/**
 * Get the user's selected organization from the request
 * This replaces the broken pattern: user.memberships[0].organization
 */
export async function getUserOrganization(request: NextRequest) {
  // Get session
  const session = await getServerSession(authOptions)
  if (!session?.user?.email) {
    throw new Error('Unauthorized: No valid session')
  }

  // Get user with their organization memberships
  const user = await db.user.findUnique({
    where: { email: session.user.email },
    include: { 
      memberships: { 
        include: { organization: true } 
      } 
    },
  })

  if (!user || user.memberships.length === 0) {
    throw new Error('No organization found: User has no organization memberships')
  }

  // Get the organization ID from the request header
  const requestedOrgId = request.headers.get(ORG_HEADER)
  
  if (requestedOrgId) {
    // Verify the user has access to the requested organization
    const requestedMembership = user.memberships.find(
      membership => membership.organization.id === requestedOrgId
    )
    
    if (requestedMembership) {
      return {
        user,
        organization: requestedMembership.organization,
        userRole: requestedMembership.role
      }
    }
    
    // User doesn't have access to the requested organization
    throw new Error(`Access denied: User is not a member of organization ${requestedOrgId}`)
  }
  
  // No organization specified - fall back to the first one
  // This maintains backward compatibility during the transition
  const fallbackMembership = user.memberships[0]
  
  console.warn('⚠️  No organization specified in request header. Using fallback:', fallbackMembership.organization.name)
  
  return {
    user,
    organization: fallbackMembership.organization,
    userRole: fallbackMembership.role
  }
}

/**
 * For endpoints that specifically need organization validation
 */
export async function requireUserOrganization(request: NextRequest) {
  try {
    return await getUserOrganization(request)
  } catch (error) {
    throw error
  }
}

/**
 * Get all organizations the user has access to
 */
export async function getUserOrganizations(request: NextRequest) {
  const session = await getServerSession(authOptions)
  if (!session?.user?.email) {
    throw new Error('Unauthorized: No valid session')
  }

  const user = await db.user.findUnique({
    where: { email: session.user.email },
    include: { 
      memberships: { 
        include: { organization: true } 
      } 
    },
  })

  if (!user) {
    throw new Error('User not found')
  }

  return {
    user,
    organizations: user.memberships.map(membership => ({
      organization: membership.organization,
      role: membership.role
    }))
  }
}