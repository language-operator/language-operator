import { db } from '@/lib/db'

/**
 * Check if a URL contains a valid invitation callback
 */
export function hasInvitationCallback(callbackUrl: string | null): boolean {
  if (!callbackUrl) return false
  return /\/invites\/[^\/]+/.test(callbackUrl)
}

/**
 * Extract invitation token from callback URL
 */
export function extractInvitationToken(callbackUrl: string | null): string | null {
  if (!callbackUrl) return null
  const match = callbackUrl.match(/\/invites\/([^\/]+)/)
  return match ? match[1] : null
}

/**
 * Validate that an invitation token exists and is not expired
 */
export async function validateInvitationToken(token: string): Promise<boolean> {
  if (!token) return false
  
  try {
    const invite = await db.organizationInvite.findUnique({
      where: { token },
      select: { 
        expiresAt: true 
      }
    })
    
    if (!invite) return false
    
    // Check if invitation is expired
    if (invite.expiresAt < new Date()) return false
    
    return true
  } catch (error) {
    console.error('Error validating invitation token:', error)
    return false
  }
}

/**
 * Check if signup is allowed based on disabled state and invitation context
 */
export async function isSignupAllowed(
  isSignupsDisabled: boolean,
  callbackUrl: string | null = null
): Promise<boolean> {
  // If signups are not disabled, allow all signups
  if (!isSignupsDisabled) return true
  
  // If signups are disabled, require valid invitation
  if (!hasInvitationCallback(callbackUrl)) return false
  
  const token = extractInvitationToken(callbackUrl)
  if (!token) return false
  
  return await validateInvitationToken(token)
}