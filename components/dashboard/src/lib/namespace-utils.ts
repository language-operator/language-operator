import { nanoid } from 'nanoid'

/**
 * Generates a secure, unique organization namespace using the pattern `org-{shortId}`
 * where shortId is a URL-safe 7-8 character string.
 * 
 * Examples: org-AzRfHys7, org-K4bDm2nP, org-V1StGXR8
 * 
 * @returns A namespace string following the pattern `org-[a-z0-9]{7,8}`
 */
export function generateOrganizationNamespace(): string {
  // Use nanoid for short, URL-safe identifiers
  // Generate 8 characters for good entropy while staying readable
  const shortId = nanoid(8)
    .replace(/_/g, '') // Remove underscores (nanoid can include them)
    .replace(/-/g, '') // Remove hyphens to avoid confusion with our separator
    .toLowerCase()
    .slice(0, 8) // Ensure exactly 8 characters
  
  return `org-${shortId}`
}

/**
 * Validates that a namespace follows the expected UUID-based organization pattern
 * 
 * @param namespace The namespace string to validate
 * @returns true if namespace matches pattern `org-[a-z0-9]{7,8}`
 */
export function validateNamespace(namespace: string): boolean {
  return /^org-[a-z0-9]{7,8}$/.test(namespace)
}

/**
 * Checks if a namespace is a legacy slug-based namespace (non-UUID)
 * Legacy namespaces would be anything that doesn't match our UUID pattern
 * 
 * @param namespace The namespace to check
 * @returns true if this appears to be a legacy slug-based namespace
 */
export function isLegacyNamespace(namespace: string): boolean {
  return !validateNamespace(namespace)
}

/**
 * Extracts the organization ID portion from a UUID-based namespace
 * 
 * @param namespace UUID-based namespace (e.g., "org-AzRfHys7")
 * @returns The short ID portion (e.g., "AzRfHys7") or null if invalid
 */
export function extractNamespaceId(namespace: string): string | null {
  const match = namespace.match(/^org-([a-z0-9]{7,8})$/)
  return match ? match[1] : null
}