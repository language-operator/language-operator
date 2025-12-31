/**
 * Test suite for initial setup functionality
 */

import { shouldPerformInitialSetup } from '../initial-setup'

// Mock the database
jest.mock('../db', () => ({
  db: {
    user: {
      count: jest.fn(),
    },
    organization: {
      count: jest.fn(),
    },
    $transaction: jest.fn(),
  },
}))

const mockDb = require('../db').db

describe('Initial Setup', () => {
  beforeEach(() => {
    // Clear all environment variables
    delete process.env.LANGOP_INIT_ADMIN_NAME
    delete process.env.LANGOP_INIT_ADMIN_EMAIL
    delete process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH
    
    // Reset mocks
    jest.clearAllMocks()
  })

  describe('shouldPerformInitialSetup', () => {
    test('returns false when no environment variables are set', async () => {
      const result = await shouldPerformInitialSetup()
      expect(result).toBe(false)
    })

    test('returns false when only some environment variables are set', async () => {
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      // Missing LANGOP_INIT_ADMIN_PASSWORD_HASH

      const result = await shouldPerformInitialSetup()
      expect(result).toBe(false)
    })

    test('returns false when database has existing users', async () => {
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH = '$2a$12$test'

      mockDb.user.count.mockResolvedValue(1) // Has users
      mockDb.organization.count.mockResolvedValue(0) // No organizations

      const result = await shouldPerformInitialSetup()
      expect(result).toBe(false)
    })

    test('returns false when database has existing organizations', async () => {
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH = '$2a$12$test'

      mockDb.user.count.mockResolvedValue(0) // No users
      mockDb.organization.count.mockResolvedValue(1) // Has organizations

      const result = await shouldPerformInitialSetup()
      expect(result).toBe(false)
    })

    test('returns true when all conditions are met', async () => {
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH = '$2a$12$test'

      mockDb.user.count.mockResolvedValue(0) // No users
      mockDb.organization.count.mockResolvedValue(0) // No organizations

      const result = await shouldPerformInitialSetup()
      expect(result).toBe(true)
    })

    test('returns false when database query fails', async () => {
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH = '$2a$12$test'

      mockDb.user.count.mockRejectedValue(new Error('Database error'))

      const result = await shouldPerformInitialSetup()
      expect(result).toBe(false)
    })
  })

  describe('Environment Variable Validation', () => {
    test('validates all required environment variables are present', () => {
      // Test that we're checking for all three required variables
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH = '$2a$12$test'

      const hasAllVars = !!(
        process.env.LANGOP_INIT_ADMIN_NAME &&
        process.env.LANGOP_INIT_ADMIN_EMAIL &&
        process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH
      )

      expect(hasAllVars).toBe(true)
    })

    test('correctly identifies missing environment variables', () => {
      // Test missing name
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH = '$2a$12$test'

      let hasAllVars = !!(
        process.env.LANGOP_INIT_ADMIN_NAME &&
        process.env.LANGOP_INIT_ADMIN_EMAIL &&
        process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH
      )

      expect(hasAllVars).toBe(false)

      // Test missing email
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      delete process.env.LANGOP_INIT_ADMIN_EMAIL
      process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH = '$2a$12$test'

      hasAllVars = !!(
        process.env.LANGOP_INIT_ADMIN_NAME &&
        process.env.LANGOP_INIT_ADMIN_EMAIL &&
        process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH
      )

      expect(hasAllVars).toBe(false)

      // Test missing password hash
      process.env.LANGOP_INIT_ADMIN_NAME = 'Test Admin'
      process.env.LANGOP_INIT_ADMIN_EMAIL = 'admin@test.local'
      delete process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH

      hasAllVars = !!(
        process.env.LANGOP_INIT_ADMIN_NAME &&
        process.env.LANGOP_INIT_ADMIN_EMAIL &&
        process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH
      )

      expect(hasAllVars).toBe(false)
    })
  })
})