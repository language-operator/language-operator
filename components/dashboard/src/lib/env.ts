// Environment variable validation and helpers
import { z } from 'zod'

const envSchema = z.object({
  // Database (optional for CI builds)
  DATABASE_URL: z.string().optional(),
  
  // NextAuth
  NEXTAUTH_SECRET: z.string().min(1).optional(),
  NEXTAUTH_URL: z.string().url().optional(),
  
  // OAuth Providers (optional)
  GOOGLE_CLIENT_ID: z.string().optional(),
  GOOGLE_CLIENT_SECRET: z.string().optional(),
  GITHUB_CLIENT_ID: z.string().optional(),
  GITHUB_CLIENT_SECRET: z.string().optional(),
  
  // Build environment
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),
  CI: z.string().optional(),
  
  // Kubernetes (optional for development)
  KUBECONFIG: z.string().optional(),
  
  // Signup control
  LANGOP_SIGNUPS_DISABLED: z.string().optional(),
  
  // Organization namespace prefix
  LANGOP_ORGANIZATION_NAMESPACE_PREFIX: z.string().optional(),
})

// Validate environment variables
function validateEnv() {
  try {
    return envSchema.parse(process.env)
  } catch (error) {
    console.warn('Environment validation warnings:', error)
    return process.env
  }
}

export const env = validateEnv()

// Helper functions
export const isDevelopment = env.NODE_ENV === 'development'
export const isProduction = env.NODE_ENV === 'production'
export const isCIBuild = Boolean(env.CI)
export const hasDatabaseUrl = Boolean(env.DATABASE_URL)

// Check if we're in a build-only environment (CI without database)
export const isBuildOnly = isCIBuild && !hasDatabaseUrl

// Check if signups are disabled (require invitation tokens)
export const isSignupsDisabled = env.LANGOP_SIGNUPS_DISABLED === 'true'

// Get organization namespace prefix (defaults to "language-operator-")
export const getOrganizationNamespacePrefix = () => 
  env.LANGOP_ORGANIZATION_NAMESPACE_PREFIX || 'language-operator-'

export default env