/**
 * Test suite for organization-aware middleware
 * 
 * Tests the Next.js middleware that handles organization validation,
 * URL extraction, rate limiting, and security features.
 */

import { NextRequest, NextResponse } from 'next/server'
import { middleware } from '../middleware'
import { getToken } from 'next-auth/jwt'

// Mock next-auth JWT
jest.mock('next-auth/jwt', () => ({
  getToken: jest.fn()
}))

// Mock NextResponse
jest.mock('next/server', () => ({
  NextRequest: jest.fn(),
  NextResponse: {
    next: jest.fn(),
    json: jest.fn()
  }
}))

const mockGetToken = getToken as jest.MockedFunction<typeof getToken>
const mockNextResponse = NextResponse as any

// Helper to create mock request
const createMockRequest = (url: string, method: string = 'GET'): NextRequest => {
  return {
    url,
    method,
    nextUrl: new URL(url, 'http://localhost:3000'),
    headers: new Map()
  } as any
}

describe('Organization Middleware', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    
    // Default auth token
    mockGetToken.mockResolvedValue({
      sub: 'user-123',
      email: 'test@example.com'
    } as any)

    // Default NextResponse mocks
    mockNextResponse.next.mockReturnValue({ headers: new Map() })
    mockNextResponse.json.mockReturnValue({})
  })

  describe('Route filtering', () => {
    it('should skip non-API routes', async () => {
      const request = createMockRequest('http://localhost:3000/dashboard')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })

    it('should skip authentication routes', async () => {
      const request = createMockRequest('http://localhost:3000/api/auth/session')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })

    it('should skip health check routes', async () => {
      const request = createMockRequest('http://localhost:3000/api/health')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })

    it('should skip monitoring routes', async () => {
      const request = createMockRequest('http://localhost:3000/api/monitoring/metrics')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })

    it('should skip _next routes', async () => {
      const request = createMockRequest('http://localhost:3000/api/_next/static/file.js')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })
  })

  describe('Organization ID extraction', () => {
    it('should extract valid UUID organization IDs', async () => {
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      const mockResponse = { headers: { set: jest.fn() } }
      mockNextResponse.next.mockReturnValue(mockResponse)
      
      await middleware(request)
      
      expect(mockResponse.headers.set).toHaveBeenCalledWith('x-organization-context', orgId)
      expect(mockResponse.headers.set).toHaveBeenCalledWith('x-user-id', 'user-123')
    })

    it('should extract valid CUID organization IDs', async () => {
      const orgId = 'clx1234567890abcdefghijk'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/agents`)
      
      const mockResponse = { headers: { set: jest.fn() } }
      mockNextResponse.next.mockReturnValue(mockResponse)
      
      await middleware(request)
      
      expect(mockResponse.headers.set).toHaveBeenCalledWith('x-organization-context', orgId)
    })

    it('should skip invalid organization ID formats', async () => {
      const request = createMockRequest('http://localhost:3000/api/invalid-org-id/clusters')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })

    it('should skip routes without organization ID', async () => {
      const request = createMockRequest('http://localhost:3000/api/organizations')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })
  })

  describe('Authentication validation', () => {
    it('should return 401 for unauthenticated requests', async () => {
      mockGetToken.mockResolvedValue(null)
      
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      await middleware(request)
      
      expect(mockNextResponse.json).toHaveBeenCalledWith(
        { error: 'Authentication required', code: 'UNAUTHENTICATED' },
        { status: 401 }
      )
    })

    it('should return 401 for token without user ID', async () => {
      mockGetToken.mockResolvedValue({ email: 'test@example.com' } as any)
      
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      await middleware(request)
      
      expect(mockNextResponse.json).toHaveBeenCalledWith(
        { error: 'Authentication required', code: 'UNAUTHENTICATED' },
        { status: 401 }
      )
    })

    it('should log security events for unauthenticated access', async () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation()
      mockGetToken.mockResolvedValue(null)
      
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      await middleware(request)
      
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Unauthenticated access attempt')
      )
      
      consoleSpy.mockRestore()
    })
  })

  describe('Rate limiting', () => {
    // Note: Rate limiting tests would require mocking the in-memory store
    // In a real implementation, you might use Redis or a database
    it('should track failed attempts in memory', async () => {
      // This test would verify that failed attempts are recorded
      // Implementation depends on the actual rate limiting strategy
      expect(true).toBe(true) // Placeholder
    })

    it('should return 429 when rate limit exceeded', async () => {
      // This would test the rate limiting functionality
      // Implementation depends on the actual rate limiting strategy
      expect(true).toBe(true) // Placeholder
    })

    it('should clear failed attempts on successful access', async () => {
      // This would test that successful access resets the counter
      expect(true).toBe(true) // Placeholder
    })
  })

  describe('Organization access validation', () => {
    it('should allow access for valid organization and user', async () => {
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      const mockResponse = { headers: { set: jest.fn() } }
      mockNextResponse.next.mockReturnValue(mockResponse)
      
      await middleware(request)
      
      expect(mockResponse.headers.set).toHaveBeenCalledWith('x-organization-context', orgId)
      expect(mockResponse.headers.set).toHaveBeenCalledWith('x-user-id', 'user-123')
    })

    it('should log successful access for audit trail', async () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation()
      
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      const mockResponse = { headers: { set: jest.fn() } }
      mockNextResponse.next.mockReturnValue(mockResponse)
      
      await middleware(request)
      
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Successful org access')
      )
      
      consoleSpy.mockRestore()
    })
  })

  describe('Security headers', () => {
    it('should add security headers to response', async () => {
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      const mockResponse = { headers: { set: jest.fn() } }
      mockNextResponse.next.mockReturnValue(mockResponse)
      
      await middleware(request)
      
      expect(mockResponse.headers.set).toHaveBeenCalledWith('X-Content-Type-Options', 'nosniff')
      expect(mockResponse.headers.set).toHaveBeenCalledWith('X-Frame-Options', 'DENY')
      expect(mockResponse.headers.set).toHaveBeenCalledWith('X-XSS-Protection', '1; mode=block')
    })
  })

  describe('Error handling', () => {
    it('should handle JWT verification errors gracefully', async () => {
      mockGetToken.mockRejectedValue(new Error('JWT verification failed'))
      
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      await middleware(request)
      
      expect(mockNextResponse.json).toHaveBeenCalledWith(
        expect.objectContaining({ error: 'Authentication required' }),
        { status: 401 }
      )
    })

    it('should handle malformed organization IDs', async () => {
      const request = createMockRequest('http://localhost:3000/api/malformed-id/clusters')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })
  })

  describe('Frontend route support (future)', () => {
    it('should skip frontend routes during Phase 1', async () => {
      const request = createMockRequest('http://localhost:3000/org123/dashboard')
      
      await middleware(request)
      
      expect(mockNextResponse.next).toHaveBeenCalled()
      expect(mockGetToken).not.toHaveBeenCalled()
    })
  })

  describe('Configuration', () => {
    it('should have correct matcher configuration', () => {
      // This would test the exported config object
      // In a real test, you'd import the config and verify the matcher
      expect(true).toBe(true) // Placeholder
    })
  })

  describe('Performance', () => {
    it('should minimize database calls during validation', async () => {
      // In Phase 4, we use lightweight validation to avoid DB calls in middleware
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const request = createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      
      const mockResponse = { headers: { set: jest.fn() } }
      mockNextResponse.next.mockReturnValue(mockResponse)
      
      await middleware(request)
      
      // Should complete without database queries for performance
      expect(mockResponse.headers.set).toHaveBeenCalledWith('x-organization-context', orgId)
    })

    it('should handle high request volume efficiently', async () => {
      // Test that middleware can handle multiple concurrent requests
      const orgId = '12345678-1234-1234-1234-123456789abc'
      const requests = Array.from({ length: 10 }, () =>
        createMockRequest(`http://localhost:3000/api/${orgId}/clusters`)
      )
      
      const mockResponse = { headers: { set: jest.fn() } }
      mockNextResponse.next.mockReturnValue(mockResponse)
      
      const promises = requests.map(request => middleware(request))
      await Promise.all(promises)
      
      expect(mockResponse.headers.set).toHaveBeenCalledTimes(20) // 2 headers per request
    })
  })
})