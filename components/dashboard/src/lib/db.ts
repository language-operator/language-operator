import { MockPrismaClient } from './prisma-stub'
import { isBuildOnly, hasDatabaseUrl } from './env'

// Conditional import for build-time compatibility
let PrismaClient: any
let isUsingMock = false

// Use mock client if:
// 1. We're in a CI build without database URL, OR
// 2. Prisma client is not available
if (isBuildOnly || !hasDatabaseUrl) {
  console.log('Using mock Prisma client for build-only environment')
  PrismaClient = MockPrismaClient
  isUsingMock = true
} else {
  try {
    ({ PrismaClient } = require('@prisma/client'))
  } catch (error) {
    console.warn('Prisma client not available, falling back to mock client')
    PrismaClient = MockPrismaClient
    isUsingMock = true
  }
}

const globalForPrisma = global as unknown as { prisma: any }

export const db = globalForPrisma.prisma || new PrismaClient(
  isUsingMock ? {} : {
    log: process.env.NODE_ENV === 'development' ? ['query', 'error', 'warn'] : ['error'],
  }
)

if (process.env.NODE_ENV !== 'production') globalForPrisma.prisma = db

// DATABASE_URL format:
// postgresql://user:password@host:port/database?schema=public
