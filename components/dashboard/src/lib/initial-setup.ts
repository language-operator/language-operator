import { db } from './db'
import { randomBytes } from 'crypto'

/**
 * Check if initial setup should be performed
 * Only performs setup if:
 * 1. All LANGOP_INIT_* environment variables are present
 * 2. Database is completely empty (no users and no organizations)
 */
export async function shouldPerformInitialSetup(): Promise<boolean> {
  console.log('🔍 [INITIAL-SETUP] Checking if initial setup is needed...')
  
  // Check for CLI-provided init environment variables
  const hasInitVars = !!(
    process.env.LANGOP_INIT_ADMIN_NAME &&
    process.env.LANGOP_INIT_ADMIN_EMAIL &&
    process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH
  )
  
  if (!hasInitVars) {
    console.log('ℹ️ [INITIAL-SETUP] No LANGOP_INIT_* environment variables found, skipping setup')
    return false
  }
  
  console.log('✅ [INITIAL-SETUP] Found LANGOP_INIT_* environment variables')
  
  // Only setup if database is completely empty
  try {
    const [userCount, orgCount] = await Promise.all([
      db.user.count(),
      db.organization.count()
    ])
    
    console.log(`📊 [INITIAL-SETUP] Database state: ${userCount} users, ${orgCount} organizations`)
    
    if (userCount === 0 && orgCount === 0) {
      console.log('🎯 [INITIAL-SETUP] Database is empty, setup needed')
      return true
    } else {
      console.log('⏭️ [INITIAL-SETUP] Database not empty, skipping setup')
      return false
    }
  } catch (error) {
    console.error('❌ [INITIAL-SETUP] Error checking database state:', error)
    return false
  }
}

/**
 * Generate a unique namespace for an organization
 * Format: org-xxxxxxxx (8 random hex characters)
 */
function generateOrgNamespace(): string {
  const randomHex = randomBytes(4).toString('hex')
  return `org-${randomHex}`
}

/**
 * Perform initial setup by creating the admin user and default organization
 * This is called when the dashboard starts up with empty database and CLI-provided credentials
 */
export async function performInitialSetup(): Promise<void> {
  console.log('🚀 [INITIAL-SETUP] Starting initial setup...')
  
  const adminName = process.env.LANGOP_INIT_ADMIN_NAME!
  const adminEmail = process.env.LANGOP_INIT_ADMIN_EMAIL!
  const adminPasswordHash = process.env.LANGOP_INIT_ADMIN_PASSWORD_HASH!
  
  console.log(`👤 [INITIAL-SETUP] Creating admin user: ${adminEmail}`)
  
  try {
    await db.$transaction(async (tx) => {
      // Create initial admin user
      const user = await tx.user.create({
        data: {
          name: adminName,
          email: adminEmail,
          password: adminPasswordHash, // Already hashed by CLI
          emailVerified: new Date(), // Mark email as verified since it came from CLI setup
        }
      })
      
      console.log(`✅ [INITIAL-SETUP] Created admin user with ID: ${user.id}`)
      
      // Create default organization
      const orgNamespace = generateOrgNamespace()
      const organization = await tx.organization.create({
        data: {
          name: "Default Organization",
          slug: "default",
          namespace: orgNamespace,
          plan: "free",
        }
      })
      
      console.log(`🏢 [INITIAL-SETUP] Created organization: ${organization.name} (namespace: ${orgNamespace})`)
      
      // Add admin user as organization owner
      await tx.organizationMember.create({
        data: {
          userId: user.id,
          organizationId: organization.id,
          role: "owner"
        }
      })
      
      console.log(`👑 [INITIAL-SETUP] Added ${adminEmail} as owner of ${organization.name}`)
    })
    
    console.log('🎉 [INITIAL-SETUP] Initial admin setup completed successfully')
    console.log(`🔐 [INITIAL-SETUP] Admin can now login with email: ${adminEmail}`)
    
  } catch (error) {
    console.error('💥 [INITIAL-SETUP] Failed to create initial admin user:', error)
    throw error
  }
}

/**
 * Main entry point for initial setup
 * Checks if setup is needed and performs it if so
 */
export async function checkAndPerformInitialSetup(): Promise<boolean> {
  try {
    if (await shouldPerformInitialSetup()) {
      await performInitialSetup()
      return true
    }
    return false
  } catch (error) {
    console.error('❌ [INITIAL-SETUP] Initial setup failed:', error)
    // Don't throw - we want the dashboard to continue starting up even if setup fails
    return false
  }
}