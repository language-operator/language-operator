/**
 * Test suite for enhanced API error handling
 * 
 * Tests the organization-aware error handling with security logging,
 * standardized error codes, and validation helpers.
 */

import { 
  OrganizationNotFoundError,
  OrganizationMismatchError,
  OrganizationAccessDeniedError,
  InvalidOrganizationIdError,
  OrganizationContextMissingError,
  RateLimitError,
  InsufficientPermissionsError,
  validateOrganizationContext,
  withOrganizationErrorHandler,
  createOrganizationErrorResponse,
  API_ERROR_CODES
} from '../api-error-handler'
import { NextResponse } from 'next/server'

// Mock NextResponse
jest.mock('next/server', () => ({
  NextResponse: {
    json: jest.fn()
  }
}))

// Mock console methods to test logging
const mockConsole = {
  warn: jest.fn(),
  error: jest.fn(),
  log: jest.fn()
}

// Replace console methods
Object.assign(console, mockConsole)

describe('API Error Handler', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockConsole.warn.mockClear()
    mockConsole.error.mockClear()
    mockConsole.log.mockClear()
  })

  describe('Organization-specific error classes', () => {
    describe('OrganizationNotFoundError', () => {
      it('should create error with correct properties', () => {
        const error = new OrganizationNotFoundError('org123')

        expect(error.message).toBe("Organization 'org123' not found")
        expect(error.code).toBe('ORGANIZATION_NOT_FOUND')
        expect(error.statusCode).toBe(404)
        expect(error.context).toEqual({ organizationId: 'org123' })
      })
    })

    describe('OrganizationMismatchError', () => {
      it('should create error with context information', () => {
        const error = new OrganizationMismatchError('url-org', 'context-org')

        expect(error.message).toBe('Organization ID mismatch')
        expect(error.code).toBe('ORGANIZATION_MISMATCH')
        expect(error.statusCode).toBe(400)
        expect(error.context).toEqual({ 
          urlOrgId: 'url-org', 
          contextOrgId: 'context-org' 
        })
      })
    })

    describe('OrganizationAccessDeniedError', () => {
      it('should create access denied error', () => {
        const error = new OrganizationAccessDeniedError('org123', 'user456')

        expect(error.message).toBe("Access denied to organization 'org123'")
        expect(error.code).toBe('ORGANIZATION_ACCESS_DENIED')
        expect(error.statusCode).toBe(403)
        expect(error.context).toEqual({ 
          organizationId: 'org123', 
          userId: 'user456' 
        })
      })
    })

    describe('InvalidOrganizationIdError', () => {
      it('should create validation error for invalid org ID', () => {
        const error = new InvalidOrganizationIdError('invalid-id')

        expect(error.message).toBe("Invalid organization ID 'invalid-id'")
        expect(error.code).toBe('INVALID_ORGANIZATION_ID')
        expect(error.statusCode).toBe(400)
        expect(error.context).toEqual({ organizationId: 'invalid-id' })
      })
    })

    describe('OrganizationContextMissingError', () => {
      it('should create context missing error', () => {
        const error = new OrganizationContextMissingError()

        expect(error.message).toBe('Organization context not found')
        expect(error.code).toBe('ORGANIZATION_CONTEXT_MISSING')
        expect(error.statusCode).toBe(400)
      })
    })

    describe('RateLimitError', () => {
      it('should create rate limit error with context', () => {
        const error = new RateLimitError('user123', 'org456')

        expect(error.message).toBe('Too many requests')
        expect(error.code).toBe('RATE_LIMITED')
        expect(error.statusCode).toBe(429)
        expect(error.context).toEqual({ 
          userId: 'user123', 
          organizationId: 'org456' 
        })
      })
    })

    describe('InsufficientPermissionsError', () => {
      it('should create permissions error with full context', () => {
        const error = new InsufficientPermissionsError(
          'delete',
          'clusters',
          'viewer',
          'org123'
        )

        expect(error.message).toBe('Insufficient permissions to delete clusters')
        expect(error.code).toBe('INSUFFICIENT_PERMISSIONS')
        expect(error.statusCode).toBe(403)
        expect(error.context).toEqual({
          action: 'delete',
          resource: 'clusters',
          userRole: 'viewer',
          organizationId: 'org123'
        })
      })
    })
  })

  describe('validateOrganizationContext', () => {
    it('should pass validation when IDs match', () => {
      expect(() => {
        validateOrganizationContext('org123', 'org123', 'user456')
      }).not.toThrow()
    })

    it('should throw OrganizationContextMissingError when context is missing', () => {
      expect(() => {
        validateOrganizationContext('org123', null, 'user456')
      }).toThrow(OrganizationContextMissingError)

      expect(() => {
        validateOrganizationContext('org123', undefined, 'user456')
      }).toThrow(OrganizationContextMissingError)
    })

    it('should throw OrganizationMismatchError when IDs do not match', () => {
      expect(() => {
        validateOrganizationContext('org123', 'org456', 'user789')
      }).toThrow(OrganizationMismatchError)
    })
  })

  describe('createOrganizationErrorResponse', () => {
    const mockNextResponse = {
      json: jest.fn()
    }

    beforeEach(() => {
      ;(NextResponse.json as jest.Mock).mockReturnValue(mockNextResponse)
    })

    it('should handle organization-specific errors', () => {
      const error = new OrganizationNotFoundError('org123')
      
      createOrganizationErrorResponse(error, 'org123', 'user456')

      expect(NextResponse.json).toHaveBeenCalledWith(
        {
          error: "Organization 'org123' not found",
          code: 'ORGANIZATION_NOT_FOUND',
          details: "The organization 'org123' does not exist or you don't have access to it",
          context: { organizationId: 'org123' },
          timestamp: expect.any(String)
        },
        { status: 404 }
      )
    })

    it('should log security events for access denied errors', () => {
      const error = new OrganizationAccessDeniedError('org123', 'user456')
      
      createOrganizationErrorResponse(error)

      expect(mockConsole.warn).toHaveBeenCalledWith(
        '[Security] ORGANIZATION_ACCESS_DENIED: Access denied to organization \'org123\'',
        {
          organizationId: 'org123',
          userId: 'user456',
          timestamp: expect.any(String)
        }
      )
    })

    it('should log security events for rate limiting', () => {
      const error = new RateLimitError('user123', 'org456')
      
      createOrganizationErrorResponse(error)

      expect(mockConsole.warn).toHaveBeenCalledWith(
        '[Security] RATE_LIMITED: Too many requests',
        {
          organizationId: 'org456',
          userId: 'user123',
          timestamp: expect.any(String)
        }
      )
    })

    it('should log security events for insufficient permissions', () => {
      const error = new InsufficientPermissionsError('delete', 'clusters', 'viewer', 'org123')
      
      createOrganizationErrorResponse(error)

      expect(mockConsole.warn).toHaveBeenCalledWith(
        '[Security] INSUFFICIENT_PERMISSIONS: Insufficient permissions to delete clusters',
        expect.objectContaining({
          organizationId: 'org123'
        })
      )
    })

    it('should handle Zod validation errors', () => {
      const zodError = {
        name: 'ZodError',
        errors: [
          {
            path: ['name'],
            message: 'Required',
            code: 'invalid_type'
          }
        ]
      }

      createOrganizationErrorResponse(zodError, 'org123', 'user456')

      expect(NextResponse.json).toHaveBeenCalledWith(
        {
          error: 'Validation failed',
          code: 'VALIDATION_FAILED',
          details: zodError.errors
        },
        { status: 400 }
      )
    })

    it('should handle generic errors', () => {
      const genericError = new Error('Something went wrong')
      
      createOrganizationErrorResponse(genericError, 'org123', 'user456')

      expect(NextResponse.json).toHaveBeenCalledWith(
        {
          error: 'Something went wrong',
          code: 'INTERNAL_ERROR'
        },
        { status: 500 }
      )
    })

    it('should log internal server errors', () => {
      const serverError = new Error('Database connection failed')
      
      createOrganizationErrorResponse(serverError)

      expect(mockConsole.error).toHaveBeenCalledWith(
        '[API Error] INTERNAL_ERROR: Database connection failed',
        expect.objectContaining({
          statusCode: 500
        })
      )
    })
  })

  describe('withOrganizationErrorHandler', () => {
    it('should catch and convert errors to responses', async () => {
      const mockHandler = jest.fn().mockRejectedValue(
        new OrganizationNotFoundError('org123')
      )

      const wrappedHandler = withOrganizationErrorHandler(mockHandler)
      
      const result = await wrappedHandler('arg1', 'arg2')

      expect(mockHandler).toHaveBeenCalledWith('arg1', 'arg2')
      expect(result).toBeDefined() // Should return a Response object
    })

    it('should pass through successful responses', async () => {
      const mockResponse = { success: true, data: 'test' }
      const mockHandler = jest.fn().mockResolvedValue(mockResponse)

      const wrappedHandler = withOrganizationErrorHandler(mockHandler)
      
      const result = await wrappedHandler('arg1')

      expect(result).toBe(mockResponse)
    })

    it('should handle non-error exceptions', async () => {
      const mockHandler = jest.fn().mockRejectedValue('String error')

      const wrappedHandler = withOrganizationErrorHandler(mockHandler)
      
      await wrappedHandler()

      expect(NextResponse.json).toHaveBeenCalledWith(
        expect.objectContaining({
          error: 'String error',
          code: 'INTERNAL_ERROR'
        }),
        { status: 500 }
      )
    })
  })

  describe('API_ERROR_CODES constants', () => {
    it('should include all organization-specific error codes', () => {
      expect(API_ERROR_CODES.ORGANIZATION_NOT_FOUND).toBe('ORGANIZATION_NOT_FOUND')
      expect(API_ERROR_CODES.ORGANIZATION_MISMATCH).toBe('ORGANIZATION_MISMATCH')
      expect(API_ERROR_CODES.ORGANIZATION_ACCESS_DENIED).toBe('ORGANIZATION_ACCESS_DENIED')
      expect(API_ERROR_CODES.INVALID_ORGANIZATION_ID).toBe('INVALID_ORGANIZATION_ID')
      expect(API_ERROR_CODES.ORGANIZATION_CONTEXT_MISSING).toBe('ORGANIZATION_CONTEXT_MISSING')
      expect(API_ERROR_CODES.RATE_LIMITED).toBe('RATE_LIMITED')
      expect(API_ERROR_CODES.INSUFFICIENT_PERMISSIONS).toBe('INSUFFICIENT_PERMISSIONS')
    })

    it('should include standard API error codes', () => {
      expect(API_ERROR_CODES.UNAUTHENTICATED).toBe('UNAUTHENTICATED')
      expect(API_ERROR_CODES.ACCESS_DENIED).toBe('ACCESS_DENIED')
      expect(API_ERROR_CODES.VALIDATION_FAILED).toBe('VALIDATION_FAILED')
      expect(API_ERROR_CODES.RESOURCE_NOT_FOUND).toBe('RESOURCE_NOT_FOUND')
      expect(API_ERROR_CODES.INTERNAL_ERROR).toBe('INTERNAL_ERROR')
    })
  })
})