/**
 * API Client with Organization Context
 * 
 * Supports both URL-based organization context (new) and header-based (legacy)
 * for seamless migration to RESTful organization URLs.
 */

import { useOrganizationStore } from '@/store/organization-store'

export const ORG_HEADER = 'x-organization-id'

/**
 * Build API URL with organization context embedded in URL
 * New approach: /api/:org_id/path instead of headers
 */
function buildOrgApiUrl(organizationId: string, path: string): string {
  // Remove leading slash if present
  const cleanPath = path.startsWith('/') ? path.slice(1) : path
  
  // Remove /api prefix if present (will be added back)
  const apiPath = cleanPath.startsWith('api/') ? cleanPath.slice(4) : cleanPath
  
  return `/api/${organizationId}/${apiPath}`
}

/**
 * Enhanced fetch with organization context
 * Supports both URL-based (new) and header-based (legacy) approaches
 */
export function fetchWithOrganization(
  input: RequestInfo | URL, 
  init?: RequestInit,
  options: {
    useUrlContext?: boolean  // Default: false for backward compatibility
    organizationId?: string  // Override organization ID
  } = {}
): Promise<Response> {
  const { activeOrganizationId } = useOrganizationStore.getState()
  const orgId = options.organizationId || activeOrganizationId
  
  // Convert input to string for URL manipulation
  const urlString = typeof input === 'string' ? input : input.toString()
  
  let finalUrl: string = urlString
  const headers = new Headers(init?.headers)
  
  if (orgId) {
    if (options.useUrlContext && urlString.startsWith('/api/') && !urlString.includes(`/api/${orgId}/`)) {
      // Convert to org-prefixed URL: /api/path → /api/:org_id/path
      const pathAfterApi = urlString.replace(/^\/api\//, '')
      finalUrl = buildOrgApiUrl(orgId, pathAfterApi)
    } else {
      // Legacy approach: add header for backward compatibility
      headers.set(ORG_HEADER, orgId)
    }
  }
  
  return fetch(finalUrl, {
    ...init,
    headers
  })
}

/**
 * Enhanced fetch specifically for org-prefixed URLs
 * Automatically uses URL-based organization context
 */
export function fetchWithOrgUrl(
  organizationId: string,
  path: string, 
  init?: RequestInit
): Promise<Response> {
  const url = buildOrgApiUrl(organizationId, path)
  
  return fetch(url, {
    ...init,
    headers: new Headers(init?.headers)
  })
}

/**
 * API client class for more complex scenarios
 * Now supports both URL-based and header-based organization context
 */
export class ApiClient {
  private baseUrl: string
  private defaultHeaders: HeadersInit
  private useUrlContext: boolean
  
  constructor(
    baseUrl: string = '/api', 
    defaultHeaders: HeadersInit = {},
    useUrlContext: boolean = false // Default to legacy mode for backward compatibility
  ) {
    this.baseUrl = baseUrl
    this.defaultHeaders = defaultHeaders
    this.useUrlContext = useUrlContext
  }
  
  private getHeaders(headers?: HeadersInit): HeadersInit {
    const { activeOrganizationId } = useOrganizationStore.getState()
    
    return {
      'Content-Type': 'application/json',
      ...this.defaultHeaders,
      ...headers,
      ...(activeOrganizationId && !this.useUrlContext && { [ORG_HEADER]: activeOrganizationId })
    }
  }
  
  private buildUrl(path: string): string {
    const { activeOrganizationId } = useOrganizationStore.getState()
    
    if (this.useUrlContext && activeOrganizationId) {
      // Convert to org-prefixed URL: /api/path → /api/:org_id/path
      const pathAfterApi = path.startsWith('/') ? path.slice(1) : path
      return buildOrgApiUrl(activeOrganizationId, pathAfterApi)
    }
    
    // Legacy mode: use baseUrl + path
    return `${this.baseUrl}${path}`
  }
  
  async get(path: string, options?: RequestInit) {
    return fetch(this.buildUrl(path), {
      method: 'GET',
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
  
  async post(path: string, data?: any, options?: RequestInit) {
    return fetch(this.buildUrl(path), {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
  
  async patch(path: string, data?: any, options?: RequestInit) {
    return fetch(this.buildUrl(path), {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
  
  async delete(path: string, options?: RequestInit) {
    return fetch(this.buildUrl(path), {
      method: 'DELETE',
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
}

// Default API client instance (legacy mode)
export const apiClient = new ApiClient()

// URL-based API client instance for new org-prefixed routes
export const orgApiClient = new ApiClient('/api', {}, true)

/**
 * Hook-friendly API client for React components
 * Enhanced to support both URL-based and header-based approaches
 */
export function useApiClient(useOrgUrls: boolean = false) {
  return {
    get: (path: string, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { method: 'GET', ...options }, { useUrlContext: useOrgUrls }),
    
    post: (path: string, data?: any, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { 
        method: 'POST', 
        body: data ? JSON.stringify(data) : undefined,
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        ...options 
      }, { useUrlContext: useOrgUrls }),
    
    patch: (path: string, data?: any, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { 
        method: 'PATCH', 
        body: data ? JSON.stringify(data) : undefined,
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        ...options 
      }, { useUrlContext: useOrgUrls }),
    
    delete: (path: string, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { method: 'DELETE', ...options }, { useUrlContext: useOrgUrls })
  }
}

/**
 * Hook specifically for org-prefixed URLs
 * Convenience wrapper for useApiClient(true)
 */
export function useOrgApiClient() {
  return useApiClient(true)
}