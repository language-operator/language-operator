/**
 * Test suite for API client with organization context
 * 
 * Tests the enhanced API client that supports both URL-based and header-based
 * organization context for backward compatibility during migration.
 */

import { 
  fetchWithOrganization,
  fetchWithOrgUrl,
  buildOrgApiUrl,
  ApiClient,
  useApiClient,
  useOrgApiClient
} from '../api-client'

// Mock the organization store
jest.mock('@/store/organization-store', () => ({
  useOrganizationStore: {
    getState: jest.fn(() => ({
      activeOrganizationId: 'test-org-123'
    }))
  }
}))

// Mock fetch globally
global.fetch = jest.fn()

const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>
const { useOrganizationStore } = require('@/store/organization-store')

describe('API Client with Organization Context', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ success: true })
    } as Response)
  })

  describe('buildOrgApiUrl', () => {
    it('should build correct org-prefixed URLs', () => {
      expect(buildOrgApiUrl('org123', 'clusters')).toBe('/api/org123/clusters')
      expect(buildOrgApiUrl('org456', 'agents')).toBe('/api/org456/agents')
    })

    it('should handle leading slashes in path', () => {
      expect(buildOrgApiUrl('org123', '/clusters')).toBe('/api/org123/clusters')
      expect(buildOrgApiUrl('org456', '/agents')).toBe('/api/org456/agents')
    })

    it('should handle paths with api prefix', () => {
      expect(buildOrgApiUrl('org123', 'api/clusters')).toBe('/api/org123/clusters')
      expect(buildOrgApiUrl('org456', '/api/agents')).toBe('/api/org456/agents')
    })

    it('should handle empty paths', () => {
      expect(buildOrgApiUrl('org123', '')).toBe('/api/org123/')
      expect(buildOrgApiUrl('org456', '/')).toBe('/api/org456/')
    })
  })

  describe('fetchWithOrganization', () => {
    it('should use header-based approach by default', async () => {
      await fetchWithOrganization('/api/clusters')

      expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
        headers: new Headers({
          'x-organization-id': 'test-org-123'
        })
      })
    })

    it('should use URL-based approach when useUrlContext is true', async () => {
      await fetchWithOrganization('/api/clusters', {}, { useUrlContext: true })

      expect(mockFetch).toHaveBeenCalledWith('/api/test-org-123/clusters', {
        headers: new Headers()
      })
    })

    it('should override organization ID when provided', async () => {
      await fetchWithOrganization('/api/clusters', {}, { organizationId: 'custom-org' })

      expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
        headers: new Headers({
          'x-organization-id': 'custom-org'
        })
      })
    })

    it('should combine URL context with custom organization ID', async () => {
      await fetchWithOrganization('/api/clusters', {}, { 
        useUrlContext: true,
        organizationId: 'custom-org'
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/custom-org/clusters', {
        headers: new Headers()
      })
    })

    it('should preserve existing headers', async () => {
      await fetchWithOrganization('/api/clusters', {
        headers: { 'Content-Type': 'application/json' }
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
        headers: new Headers({
          'Content-Type': 'application/json',
          'x-organization-id': 'test-org-123'
        })
      })
    })

    it('should handle no active organization', async () => {
      useOrganizationStore.getState.mockReturnValue({ activeOrganizationId: null })

      await fetchWithOrganization('/api/clusters')

      expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
        headers: new Headers()
      })
    })

    it('should not modify URL if already org-prefixed', async () => {
      await fetchWithOrganization('/api/test-org-123/clusters', {}, { useUrlContext: true })

      expect(mockFetch).toHaveBeenCalledWith('/api/test-org-123/clusters', {
        headers: new Headers()
      })
    })
  })

  describe('fetchWithOrgUrl', () => {
    it('should always use URL-based approach', async () => {
      await fetchWithOrgUrl('org456', 'clusters')

      expect(mockFetch).toHaveBeenCalledWith('/api/org456/clusters', {
        headers: new Headers()
      })
    })

    it('should preserve init options', async () => {
      await fetchWithOrgUrl('org456', 'clusters', {
        method: 'POST',
        body: '{"test": true}',
        headers: { 'Content-Type': 'application/json' }
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/org456/clusters', {
        method: 'POST',
        body: '{"test": true}',
        headers: new Headers({ 'Content-Type': 'application/json' })
      })
    })
  })

  describe('ApiClient class', () => {
    describe('legacy mode (header-based)', () => {
      const client = new ApiClient()

      it('should include organization header in requests', async () => {
        await client.get('/clusters')

        expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'x-organization-id': 'test-org-123'
          }
        })
      })

      it('should handle POST requests with data', async () => {
        const testData = { name: 'test-cluster' }
        
        await client.post('/clusters', testData)

        expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
          method: 'POST',
          body: JSON.stringify(testData),
          headers: {
            'Content-Type': 'application/json',
            'x-organization-id': 'test-org-123'
          }
        })
      })

      it('should not include org header when no active organization', async () => {
        useOrganizationStore.getState.mockReturnValue({ activeOrganizationId: null })

        await client.get('/clusters')

        expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json'
          }
        })
      })
    })

    describe('URL-based mode', () => {
      const client = new ApiClient('/api', {}, true)

      it('should use org-prefixed URLs', async () => {
        await client.get('/clusters')

        expect(mockFetch).toHaveBeenCalledWith('/api/test-org-123/clusters', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json'
          }
        })
      })

      it('should not include organization header', async () => {
        await client.post('/clusters', { name: 'test' })

        expect(mockFetch).toHaveBeenCalledWith('/api/test-org-123/clusters', {
          method: 'POST',
          body: JSON.stringify({ name: 'test' }),
          headers: {
            'Content-Type': 'application/json'
          }
        })
      })

      it('should fallback to base URL when no organization', async () => {
        useOrganizationStore.getState.mockReturnValue({ activeOrganizationId: null })

        await client.get('/clusters')

        expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json'
          }
        })
      })
    })
  })

  describe('useApiClient hook', () => {
    it('should use header-based approach by default', async () => {
      const client = useApiClient()
      
      await client.get('/clusters')

      expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
        method: 'GET',
        headers: new Headers({
          'x-organization-id': 'test-org-123'
        })
      })
    })

    it('should use URL-based approach when requested', async () => {
      const client = useApiClient(true)
      
      await client.get('/clusters')

      expect(mockFetch).toHaveBeenCalledWith('/api/test-org-123/clusters', {
        method: 'GET',
        headers: new Headers()
      })
    })

    it('should handle POST requests with proper headers', async () => {
      const client = useApiClient()
      const testData = { name: 'test-cluster' }
      
      await client.post('/clusters', testData)

      expect(mockFetch).toHaveBeenCalledWith('/api/clusters', {
        method: 'POST',
        body: JSON.stringify(testData),
        headers: {
          'Content-Type': 'application/json',
          'x-organization-id': 'test-org-123'
        }
      })
    })
  })

  describe('useOrgApiClient hook', () => {
    it('should always use URL-based approach', async () => {
      const client = useOrgApiClient()
      
      await client.get('/clusters')

      expect(mockFetch).toHaveBeenCalledWith('/api/test-org-123/clusters', {
        method: 'GET',
        headers: new Headers()
      })
    })

    it('should handle all HTTP methods with org URLs', async () => {
      const client = useOrgApiClient()
      
      await client.post('/clusters', { name: 'test' })
      await client.patch('/clusters/test', { domain: 'example.com' })
      await client.delete('/clusters/test')

      expect(mockFetch).toHaveBeenNthCalledWith(1, '/api/test-org-123/clusters', expect.objectContaining({
        method: 'POST'
      }))
      expect(mockFetch).toHaveBeenNthCalledWith(2, '/api/test-org-123/clusters/test', expect.objectContaining({
        method: 'PATCH'
      }))
      expect(mockFetch).toHaveBeenNthCalledWith(3, '/api/test-org-123/clusters/test', expect.objectContaining({
        method: 'DELETE'
      }))
    })
  })
})