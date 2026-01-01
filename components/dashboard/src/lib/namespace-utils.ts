import { nanoid } from 'nanoid'
import { getOrganizationNamespacePrefix } from './env'

/**
 * Generates a secure, unique organization namespace using a configurable prefix and short ID.
 * Default pattern: `language-operator-{shortId}` where shortId is a URL-safe 7-8 character string.
 * Prefix can be overridden with LANGOP_ORGANIZATION_NAMESPACE_PREFIX environment variable.
 * 
 * Examples: language-operator-AzRfHys7, language-operator-K4bDm2nP, custom-prefix-V1StGXR8
 * 
 * @returns A namespace string following the pattern `{prefix}[a-z0-9]{7,8}`
 */
export function generateOrganizationNamespace(): string {
  const prefix = getOrganizationNamespacePrefix()
  
  // Use nanoid for short, URL-safe identifiers
  // Generate 8 characters for good entropy while staying readable
  const shortId = nanoid(8)
    .replace(/_/g, '') // Remove underscores (nanoid can include them)
    .replace(/-/g, '') // Remove hyphens to avoid confusion with our separator
    .toLowerCase()
    .slice(0, 8) // Ensure exactly 8 characters
  
  return `${prefix}${shortId}`
}

/**
 * Validates that a namespace follows the expected UUID-based organization pattern
 * Uses the current configured prefix to validate the pattern.
 * 
 * @param namespace The namespace string to validate
 * @returns true if namespace matches pattern `{prefix}[a-z0-9]{7,8}`
 */
export function validateNamespace(namespace: string): boolean {
  const prefix = getOrganizationNamespacePrefix()
  const escapedPrefix = prefix.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // Escape regex special chars
  const pattern = new RegExp(`^${escapedPrefix}[a-z0-9]{7,8}$`)
  return pattern.test(namespace)
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
 * @param namespace UUID-based namespace (e.g., "language-operator-AzRfHys7")
 * @returns The short ID portion (e.g., "AzRfHys7") or null if invalid
 */
export function extractNamespaceId(namespace: string): string | null {
  const prefix = getOrganizationNamespacePrefix()
  const escapedPrefix = prefix.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // Escape regex special chars
  const pattern = new RegExp(`^${escapedPrefix}([a-z0-9]{7,8})$`)
  const match = namespace.match(pattern)
  return match ? match[1] : null
}