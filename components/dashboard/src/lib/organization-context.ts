/**
 * Organization Context Utilities
 * 
 * Provides clean utilities for handling organization selection between
 * frontend and backend, replacing the broken memberships[0] pattern.
 * Includes caching to improve performance and reduce database queries.
 */

import { NextRequest } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'

// Cache for organization data to reduce database queries
interface CacheEntry {
  data: {
    user: any
    organization: any
    userRole: string
  }
  timestamp: number
  expiry: number
}

const organizationCache = new Map<string, CacheEntry>()
const CACHE_DURATION = 5 * 60 * 1000 // 5 minutes
const MAX_CACHE_SIZE = 1000 // Prevent memory leaks

export const ORG_HEADER = 'x-organization-id'

/**
 * Cache management utilities
 */
function getCacheKey(userEmail: string, orgId?: string): string {
  return `${userEmail}:${orgId || 'default'}`
}

function getCachedData(cacheKey: string): CacheEntry['data'] | null {
  const entry = organizationCache.get(cacheKey)
  if (!entry) return null
  
  const now = Date.now()
  if (now > entry.expiry) {
    organizationCache.delete(cacheKey)
    return null
  }
  
  return entry.data
}

function setCachedData(cacheKey: string, data: CacheEntry['data']): void {
  const now = Date.now()
  
  // Prevent memory leaks by limiting cache size
  if (organizationCache.size >= MAX_CACHE_SIZE) {
    // Remove oldest entries
    const entries = Array.from(organizationCache.entries())
    entries.sort((a, b) => a[1].timestamp - b[1].timestamp)
    const toRemove = entries.slice(0, Math.floor(MAX_CACHE_SIZE * 0.2)) // Remove 20%
    toRemove.forEach(([key]) => organizationCache.delete(key))
  }
  
  organizationCache.set(cacheKey, {
    data,
    timestamp: now,
    expiry: now + CACHE_DURATION
  })
}

/**
 * Clear cache for a specific user (useful for logout, role changes)
 */
export function clearUserCache(userEmail: string): void {
  const keysToDelete = Array.from(organizationCache.keys()).filter(key =>
    key.startsWith(`${userEmail}:`)
  )
  keysToDelete.forEach(key => organizationCache.delete(key))
}

/**
 * Get the user's selected organization from the request
 * This replaces the broken pattern: user.memberships[0].organization
 * Now includes intelligent caching to reduce database queries
 */
export async function getUserOrganization(request: NextRequest) {
  // Get session
  const session = await getServerSession(authOptions)
  if (!session?.user?.email) {
    throw new Error('Unauthorized: No valid session')
  }

  // Get the organization ID from the request header
  const requestedOrgId = request.headers.get(ORG_HEADER)
  const cacheKey = getCacheKey(session.user.email, requestedOrgId || undefined)
  
  // Check cache first
  const cachedData = getCachedData(cacheKey)
  if (cachedData) {
    return cachedData
  }

  // Get user with their organization memberships from database
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

  let result: { user: any, organization: any, userRole: string }
  
  if (requestedOrgId) {
    // Verify the user has access to the requested organization
    const requestedMembership = user.memberships.find(
      membership => membership.organization.id === requestedOrgId
    )
    
    if (!requestedMembership) {
      throw new Error(`Access denied: User is not a member of organization ${requestedOrgId}`)
    }
    
    result = {
      user,
      organization: requestedMembership.organization,
      userRole: requestedMembership.role
    }
  } else {
    // No organization specified - fall back to the first one
    // This maintains backward compatibility during the transition
    const fallbackMembership = user.memberships[0]
    
    console.warn('⚠️  No organization specified in request header. Using fallback:', fallbackMembership.organization.name)
    
    result = {
      user,
      organization: fallbackMembership.organization,
      userRole: fallbackMembership.role
    }
  }
  
  // Cache the result for future requests
  setCachedData(cacheKey, result)
  
  return result
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