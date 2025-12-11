// Prisma client stub for CI/build environments
// This provides a mock implementation when the real Prisma client is unavailable

class MockPrismaClient {
  // User operations
  user = {
    findUnique: () => Promise.resolve(null),
    findMany: () => Promise.resolve([]),
    create: () => Promise.resolve({}),
    update: () => Promise.resolve({}),
    delete: () => Promise.resolve({}),
    upsert: () => Promise.resolve({})
  }

  // Organization operations  
  organization = {
    findUnique: () => Promise.resolve(null),
    findMany: () => Promise.resolve([]),
    create: () => Promise.resolve({}),
    update: () => Promise.resolve({}),
    delete: () => Promise.resolve({})
  }

  // Organization member operations
  organizationMember = {
    findUnique: () => Promise.resolve(null),
    findMany: () => Promise.resolve([]),
    create: () => Promise.resolve({}),
    update: () => Promise.resolve({}),
    delete: () => Promise.resolve({})
  }

  // Invite operations
  invite = {
    findUnique: () => Promise.resolve(null),
    findMany: () => Promise.resolve([]),
    create: () => Promise.resolve({}),
    update: () => Promise.resolve({}),
    delete: () => Promise.resolve({})
  }

  // Account operations (for NextAuth)
  account = {
    findUnique: () => Promise.resolve(null),
    findMany: () => Promise.resolve([]),
    create: () => Promise.resolve({}),
    update: () => Promise.resolve({}),
    delete: () => Promise.resolve({})
  }

  // Session operations (for NextAuth)
  session = {
    findUnique: () => Promise.resolve(null),
    findMany: () => Promise.resolve([]),
    create: () => Promise.resolve({}),
    update: () => Promise.resolve({}),
    delete: () => Promise.resolve({})
  }

  // Verification token operations (for NextAuth)
  verificationToken = {
    findUnique: () => Promise.resolve(null),
    findMany: () => Promise.resolve([]),
    create: () => Promise.resolve({}),
    update: () => Promise.resolve({}),
    delete: () => Promise.resolve({})
  }

  // Transaction support
  $transaction = (operations: any) => Promise.resolve(operations)
  
  // Connection management
  $connect = () => Promise.resolve()
  $disconnect = () => Promise.resolve()
  
  // Raw queries
  $queryRaw = () => Promise.resolve([])
  $executeRaw = () => Promise.resolve(0)
}

export { MockPrismaClient }