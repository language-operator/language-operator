// Mock PrismaAdapter for CI builds
import type { Adapter } from 'next-auth/adapters'

export function MockPrismaAdapter(): Adapter {
  return {
    createUser: async (user: any) => ({ 
      id: 'mock-user-id', 
      email: user.email || 'test@example.com',
      emailVerified: null,
      ...user 
    }),
    
    getUser: async (id: string) => null,
    
    getUserByEmail: async (email: string) => null,
    
    getUserByAccount: async ({ providerAccountId, provider }: any) => null,
    
    updateUser: async (user: any) => ({ 
      id: user.id || 'mock-user-id',
      email: 'test@example.com',
      emailVerified: null,
      ...user 
    }),
    
    deleteUser: async (userId: string) => undefined,
    
    linkAccount: async (account: any) => undefined,
    
    unlinkAccount: async ({ providerAccountId, provider }: any) => undefined,
    
    createSession: async ({ sessionToken, userId, expires }: any) => ({
      sessionToken,
      userId,
      expires,
      id: 'mock-session-id',
      createdAt: new Date(),
      updatedAt: new Date()
    }),
    
    getSessionAndUser: async (sessionToken: string) => null,
    
    updateSession: async (session: any) => ({
      id: 'mock-session-id',
      sessionToken: session.sessionToken || 'mock-token',
      userId: 'mock-user-id',
      expires: session.expires || new Date(),
      createdAt: new Date(),
      updatedAt: new Date(),
      ...session
    }),
    
    deleteSession: async (sessionToken: string) => undefined,
    
    createVerificationToken: async ({ identifier, expires, token }: any) => ({
      identifier,
      expires,
      token
    }),
    
    useVerificationToken: async ({ identifier, token }: any) => null,
  }
}