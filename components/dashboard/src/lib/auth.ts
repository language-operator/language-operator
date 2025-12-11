import { NextAuthOptions } from 'next-auth'
import GoogleProvider from 'next-auth/providers/google'
import GitHubProvider from 'next-auth/providers/github'
import CredentialsProvider from 'next-auth/providers/credentials'
import { compare } from 'bcryptjs'
import { db } from './db'
import { MockPrismaAdapter } from './mock-prisma-adapter'
import { isBuildOnly, hasDatabaseUrl } from './env'

// Create adapter conditionally
let adapter: any = undefined

if (isBuildOnly || !hasDatabaseUrl) {
  console.log('Using mock adapter for build-only environment')
  adapter = MockPrismaAdapter()
} else {
  try {
    const { PrismaAdapter } = require('@next-auth/prisma-adapter')
    adapter = PrismaAdapter(db)
  } catch (error) {
    console.warn('PrismaAdapter not available, using mock adapter')
    adapter = MockPrismaAdapter()
  }
}

export const authOptions: NextAuthOptions = {
  adapter,
  session: {
    strategy: 'jwt', // Use JWT sessions instead of database sessions
  },
  pages: {
    signIn: '/login',
    signOut: '/login',
    error: '/login',
  },
  providers: [
    GoogleProvider({
      clientId: process.env.GOOGLE_CLIENT_ID || '',
      clientSecret: process.env.GOOGLE_CLIENT_SECRET || '',
    }),
    GitHubProvider({
      clientId: process.env.GITHUB_CLIENT_ID || '',
      clientSecret: process.env.GITHUB_CLIENT_SECRET || '',
    }),
    CredentialsProvider({
      name: 'credentials',
      credentials: {
        email: { label: 'Email', type: 'email' },
        password: { label: 'Password', type: 'password' },
      },
      async authorize(credentials) {
        console.log('🔐 [AUTH] Starting authorization for:', credentials?.email)
        
        if (!credentials?.email || !credentials?.password) {
          console.log('❌ [AUTH] Missing credentials')
          throw new Error('Invalid credentials')
        }

        console.log('🔍 [AUTH] Looking up user in database...')
        const user = await db.user.findUnique({
          where: { email: credentials.email },
        })

        if (!user) {
          console.log('❌ [AUTH] User not found')
          throw new Error('Invalid credentials')
        }

        if (!user.password) {
          console.log('❌ [AUTH] User has no password')
          throw new Error('Invalid credentials')
        }

        console.log('🔑 [AUTH] Comparing password...')
        const isValidPassword = await compare(credentials.password, user.password)

        if (!isValidPassword) {
          console.log('❌ [AUTH] Password mismatch')
          throw new Error('Invalid credentials')
        }

        console.log('✅ [AUTH] Authentication successful for:', user.email)
        return {
          id: user.id,
          email: user.email,
          name: user.name,
          image: user.image,
        }
      },
    }),
  ],
  callbacks: {
    async jwt({ token, user }) {
      // Store user ID in JWT token when user first logs in
      if (user) {
        token.sub = user.id
      }
      return token
    },
    async session({ session, token }) {
      console.log('🔧 [SESSION] Building JWT session for user:', token.sub)
      
      if (session.user && token.sub) {
        session.user.id = token.sub

        try {
          console.log('👥 [SESSION] Fetching organizations for user:', token.sub)
          // Get user's organizations and active organization
          const memberships = await db.organizationMember.findMany({
            where: { userId: token.sub },
            include: {
              organization: true,
            },
            orderBy: { createdAt: 'asc' },
          })

          console.log('📋 [SESSION] Found', memberships.length, 'memberships')

          // Store organizations in session for easy access
          session.organizations = memberships.map((m: any) => ({
            id: m.organization.id,
            name: m.organization.name,
            slug: m.organization.slug,
            namespace: m.organization.namespace,
            role: m.role,
          }))

          // Set active organization (first one by default)
          session.activeOrganization = session.organizations[0] || null
          
          console.log('✅ [SESSION] Session built successfully')
        } catch (error) {
          console.error('❌ [SESSION] Error building session:', error)
          // Continue with basic session even if organizations fail
          session.organizations = []
          session.activeOrganization = null
        }
      }

      return session
    },
  },
  debug: true, // Force debug mode to see what's happening
}
