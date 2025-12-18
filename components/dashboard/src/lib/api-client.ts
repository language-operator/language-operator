/**
 * API Client with Organization Context
 * 
 * Provides fetch wrapper that automatically includes the selected organization
 * in request headers, ensuring backend APIs use the correct organization.
 */

import { useOrganizationStore } from '@/store/organization-store'

export const ORG_HEADER = 'x-organization-id'

/**
 * Enhanced fetch that automatically includes organization context
 */
export function fetchWithOrganization(
  input: RequestInfo | URL, 
  init?: RequestInit
): Promise<Response> {
  const { activeOrganizationId } = useOrganizationStore.getState()
  
  // Prepare headers
  const headers = new Headers(init?.headers)
  
  // Add organization header if we have an active organization
  if (activeOrganizationId) {
    headers.set(ORG_HEADER, activeOrganizationId)
  }
  
  // Make the request with organization context
  return fetch(input, {
    ...init,
    headers
  })
}

/**
 * API client class for more complex scenarios
 */
export class ApiClient {
  private baseUrl: string
  private defaultHeaders: HeadersInit
  
  constructor(baseUrl: string = '/api', defaultHeaders: HeadersInit = {}) {
    this.baseUrl = baseUrl
    this.defaultHeaders = defaultHeaders
  }
  
  private getHeaders(headers?: HeadersInit): HeadersInit {
    const { activeOrganizationId } = useOrganizationStore.getState()
    
    return {
      'Content-Type': 'application/json',
      ...this.defaultHeaders,
      ...headers,
      ...(activeOrganizationId && { [ORG_HEADER]: activeOrganizationId })
    }
  }
  
  async get(path: string, options?: RequestInit) {
    return fetch(`${this.baseUrl}${path}`, {
      method: 'GET',
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
  
  async post(path: string, data?: any, options?: RequestInit) {
    return fetch(`${this.baseUrl}${path}`, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
  
  async patch(path: string, data?: any, options?: RequestInit) {
    return fetch(`${this.baseUrl}${path}`, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
  
  async delete(path: string, options?: RequestInit) {
    return fetch(`${this.baseUrl}${path}`, {
      method: 'DELETE',
      ...options,
      headers: this.getHeaders(options?.headers)
    })
  }
}

// Default API client instance
export const apiClient = new ApiClient()

/**
 * Hook-friendly API client for React components
 */
export function useApiClient() {
  return {
    get: (path: string, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { method: 'GET', ...options }),
    
    post: (path: string, data?: any, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { 
        method: 'POST', 
        body: data ? JSON.stringify(data) : undefined,
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        ...options 
      }),
    
    patch: (path: string, data?: any, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { 
        method: 'PATCH', 
        body: data ? JSON.stringify(data) : undefined,
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        ...options 
      }),
    
    delete: (path: string, options?: RequestInit) => 
      fetchWithOrganization(`/api${path}`, { method: 'DELETE', ...options })
  }
}