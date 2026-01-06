/**
 * Integration test suite for org-prefixed clusters API
 * 
 * Tests the new /api/[org_id]/clusters endpoints with real request flows,
 * middleware integration, and organization validation.
 */

import { NextRequest, NextResponse } from 'next/server'
import { GET, POST } from '../clusters/route'
import { getOrganizationContext } from '@/lib/organization-utils'

// Mock dependencies
jest.mock('@/lib/organization-utils')
jest.mock('@/lib/k8s-client')
jest.mock('@/lib/db')

const mockGetOrganizationContext = getOrganizationContext as jest.MockedFunction<typeof getOrganizationContext>
const mockDb = require('@/lib/db').db
const mockK8sClient = require('@/lib/k8s-client').k8sClient

// Mock Next.js request/response
const createMockRequest = (url: string, method: string = 'GET', body?: any): NextRequest => {
  return {
    url,
    method,
    headers: new Map(),
    json: jest.fn().mockResolvedValue(body),
    nextUrl: new URL(url, 'http://localhost:3000')
  } as any
}

const createMockParams = (org_id: string) => Promise.resolve({ org_id })

describe('/api/[org_id]/clusters Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    
    // Default organization context
    mockGetOrganizationContext.mockReturnValue({
      organizationId: 'test-org-123',
      userId: 'user-456'
    })

    // Default database mocks
    mockDb.organization.findUnique.mockResolvedValue({
      id: 'test-org-123',
      name: 'Test Organization',
      namespace: 'langop-test-org',
      plan: 'pro'
    })

    mockDb.user.findUnique.mockResolvedValue({
      id: 'user-456',
      email: 'test@example.com'
    })

    // Default Kubernetes client mocks
    mockK8sClient.listByOrganization.mockResolvedValue({
      body: { items: [] }
    })
  })

  describe('GET /api/[org_id]/clusters', () => {
    it('should successfully list clusters for valid organization', async () => {
      const mockClusters = [
        {
          metadata: { name: 'cluster1', namespace: 'langop-test-org' },
          spec: { domain: 'example.com' },
          status: { phase: 'Ready' }
        },
        {
          metadata: { name: 'cluster2', namespace: 'langop-test-org' },
          spec: { domain: 'test.com' },
          status: { phase: 'Pending' }
        }
      ]

      mockK8sClient.listByOrganization
        .mockResolvedValueOnce({ body: { items: mockClusters } })  // clusters
        .mockResolvedValueOnce({ body: { items: [] } })            // agents

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })
      const responseData = await response.json()

      expect(response.status).toBe(200)
      expect(responseData.success).toBe(true)
      expect(responseData.data).toHaveLength(2)
      expect(responseData.total).toBe(2)
      expect(responseData.page).toBe(1)
      expect(responseData.limit).toBe(50)
    })

    it('should handle pagination parameters', async () => {
      const mockClusters = Array.from({ length: 100 }, (_, i) => ({
        metadata: { name: `cluster${i}`, namespace: 'langop-test-org' },
        spec: { domain: `example${i}.com` },
        status: { phase: 'Ready' }
      }))

      mockK8sClient.listByOrganization
        .mockResolvedValueOnce({ body: { items: mockClusters } })
        .mockResolvedValueOnce({ body: { items: [] } })

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters?page=2&limit=25')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })
      const responseData = await response.json()

      expect(response.status).toBe(200)
      expect(responseData.data).toHaveLength(25) // Second page of 25
      expect(responseData.total).toBe(100)
      expect(responseData.page).toBe(2)
      expect(responseData.limit).toBe(25)
    })

    it('should apply search filters', async () => {
      const mockClusters = [
        {
          metadata: { name: 'production-cluster', namespace: 'langop-test-org' },
          spec: { domain: 'prod.example.com' },
          status: { phase: 'Ready' }
        },
        {
          metadata: { name: 'staging-cluster', namespace: 'langop-test-org' },
          spec: { domain: 'staging.example.com' },
          status: { phase: 'Ready' }
        }
      ]

      mockK8sClient.listByOrganization
        .mockResolvedValueOnce({ body: { items: mockClusters } })
        .mockResolvedValueOnce({ body: { items: [] } })

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters?search=production')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })
      const responseData = await response.json()

      expect(response.status).toBe(200)
      expect(responseData.data).toHaveLength(1)
      expect(responseData.data[0].metadata.name).toBe('production-cluster')
    })

    it('should include agent counts for clusters', async () => {
      const mockClusters = [
        {
          metadata: { name: 'cluster1', namespace: 'langop-test-org' },
          spec: { domain: 'example.com' },
          status: { phase: 'Ready' }
        }
      ]

      const mockAgents = [
        { spec: { clusterRef: 'cluster1' } },
        { spec: { clusterRef: 'cluster1' } },
        { spec: { clusterRef: 'other-cluster' } }
      ]

      mockK8sClient.listByOrganization
        .mockResolvedValueOnce({ body: { items: mockClusters } })
        .mockResolvedValueOnce({ body: { items: mockAgents } })

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })
      const responseData = await response.json()

      expect(response.status).toBe(200)
      expect(responseData.data[0].status.agentCount).toBe(2)
    })

    it('should return 400 for organization ID mismatch', async () => {
      mockGetOrganizationContext.mockReturnValue({
        organizationId: 'different-org',
        userId: 'user-456'
      })

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })

      expect(response.status).toBe(400)
    })

    it('should return 400 for missing organization context', async () => {
      mockGetOrganizationContext.mockReturnValue(null)

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })

      expect(response.status).toBe(400)
    })

    it('should return 404 for non-existent organization', async () => {
      mockDb.organization.findUnique.mockResolvedValue(null)

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })

      expect(response.status).toBe(404)
    })

    it('should gracefully handle Kubernetes errors', async () => {
      mockK8sClient.listByOrganization.mockRejectedValue(new Error('K8s API unavailable'))

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      const response = await GET(request, { params })
      const responseData = await response.json()

      // Should return empty list instead of error
      expect(response.status).toBe(200)
      expect(responseData.data).toEqual([])
      expect(responseData.total).toBe(0)
    })
  })

  describe('POST /api/[org_id]/clusters', () => {
    const mockClusterFormData = {
      name: 'new-cluster',
      domain: 'new.example.com',
      enableTLS: true,
      useCertManager: true,
      issuerName: 'letsencrypt-prod'
    }

    it('should successfully create cluster', async () => {
      const mockCreatedCluster = {
        metadata: {
          name: 'new-cluster',
          namespace: 'langop-test-org'
        },
        spec: mockClusterFormData
      }

      mockK8sClient.createLanguageCluster.mockResolvedValue({
        data: mockCreatedCluster
      })

      const request = createMockRequest(
        'http://localhost:3000/api/test-org-123/clusters',
        'POST',
        mockClusterFormData
      )
      const params = createMockParams('test-org-123')

      const response = await POST(request, { params })
      const responseData = await response.json()

      expect(response.status).toBe(200)
      expect(responseData.success).toBe(true)
      expect(responseData.data).toEqual(mockCreatedCluster)
      expect(responseData.message).toContain('new-cluster')

      expect(mockK8sClient.createLanguageCluster).toHaveBeenCalledWith(
        'langop-test-org',
        expect.objectContaining({
          metadata: expect.objectContaining({
            name: 'new-cluster',
            labels: expect.objectContaining({
              'langop.io/organization-id': 'test-org-123',
              'langop.io/created-by': 'user-456'
            })
          })
        })
      )
    })

    it('should include proper metadata and labels', async () => {
      mockK8sClient.createLanguageCluster.mockResolvedValue({ data: {} })

      const request = createMockRequest(
        'http://localhost:3000/api/test-org-123/clusters',
        'POST',
        mockClusterFormData
      )
      const params = createMockParams('test-org-123')

      await POST(request, { params })

      const createCall = mockK8sClient.createLanguageCluster.mock.calls[0]
      const cluster = createCall[1]

      expect(cluster.metadata.labels).toEqual({
        'langop.io/organization-id': 'test-org-123',
        'langop.io/created-by': 'user-456'
      })

      expect(cluster.metadata.annotations).toEqual({
        'langop.io/created-by-email': 'test@example.com',
        'langop.io/created-at': expect.any(String)
      })
    })

    it('should handle TLS configuration correctly', async () => {
      mockK8sClient.createLanguageCluster.mockResolvedValue({ data: {} })

      const request = createMockRequest(
        'http://localhost:3000/api/test-org-123/clusters',
        'POST',
        {
          name: 'tls-cluster',
          enableTLS: true,
          useCertManager: true,
          issuerName: 'letsencrypt-staging',
          issuerKind: 'Issuer',
          issuerGroup: 'cert-manager.io'
        }
      )
      const params = createMockParams('test-org-123')

      await POST(request, { params })

      const createCall = mockK8sClient.createLanguageCluster.mock.calls[0]
      const cluster = createCall[1]

      expect(cluster.spec.ingressConfig.tls).toEqual({
        enabled: true,
        issuerRef: {
          name: 'letsencrypt-staging',
          kind: 'Issuer',
          group: 'cert-manager.io'
        }
      })
    })

    it('should return 404 for non-existent user', async () => {
      mockDb.user.findUnique.mockResolvedValue(null)

      const request = createMockRequest(
        'http://localhost:3000/api/test-org-123/clusters',
        'POST',
        mockClusterFormData
      )
      const params = createMockParams('test-org-123')

      const response = await POST(request, { params })

      expect(response.status).toBe(404)
    })

    it('should handle organization validation errors', async () => {
      mockGetOrganizationContext.mockReturnValue({
        organizationId: 'different-org',
        userId: 'user-456'
      })

      const request = createMockRequest(
        'http://localhost:3000/api/test-org-123/clusters',
        'POST',
        mockClusterFormData
      )
      const params = createMockParams('test-org-123')

      const response = await POST(request, { params })

      expect(response.status).toBe(400)
    })

    it('should handle Kubernetes creation errors', async () => {
      mockK8sClient.createLanguageCluster.mockRejectedValue(
        new Error('Kubernetes API error')
      )

      const request = createMockRequest(
        'http://localhost:3000/api/test-org-123/clusters',
        'POST',
        mockClusterFormData
      )
      const params = createMockParams('test-org-123')

      const response = await POST(request, { params })

      expect(response.status).toBe(500)
    })
  })

  describe('Error handling and security', () => {
    it('should validate organization access on every request', async () => {
      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      await GET(request, { params })

      expect(mockGetOrganizationContext).toHaveBeenCalled()
      expect(mockDb.organization.findUnique).toHaveBeenCalledWith({
        where: { id: 'test-org-123' },
        select: { id: true, name: true, namespace: true, plan: true }
      })
    })

    it('should use organization namespace for Kubernetes operations', async () => {
      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      await GET(request, { params })

      expect(mockK8sClient.listByOrganization).toHaveBeenCalledWith(
        'clusters',
        'langop-test-org',
        'test-org-123'
      )
    })

    it('should include security logging in error responses', async () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation()

      mockGetOrganizationContext.mockReturnValue({
        organizationId: 'wrong-org',
        userId: 'user-456'
      })

      const request = createMockRequest('http://localhost:3000/api/test-org-123/clusters')
      const params = createMockParams('test-org-123')

      await GET(request, { params })

      // Should log the organization mismatch attempt
      expect(consoleSpy).toHaveBeenCalled()

      consoleSpy.mockRestore()
    })
  })
})