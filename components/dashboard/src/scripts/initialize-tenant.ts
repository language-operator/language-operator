#!/usr/bin/env npx tsx

/**
 * Tenant initialization script
 * 
 * This script runs as a Kubernetes Job after Helm install/upgrade
 * to set up the initial admin user when LANGOP_INIT_* environment
 * variables are provided by the CLI.
 * 
 * Usage: npm run initialize-tenant
 */

import { checkAndPerformInitialSetup } from '../lib/initial-setup'

async function main() {
  console.log('🚀 [TENANT-INIT] Starting tenant initialization...')
  
  try {
    const setupPerformed = await checkAndPerformInitialSetup()
    
    if (setupPerformed) {
      console.log('✅ [TENANT-INIT] Tenant initialization completed successfully')
      console.log('🎯 [TENANT-INIT] Admin user created and ready for login')
      process.exit(0)
    } else {
      console.log('ℹ️ [TENANT-INIT] No initialization needed - tenant already configured or no init credentials provided')
      process.exit(0)
    }
  } catch (error) {
    console.error('❌ [TENANT-INIT] Tenant initialization failed:', error)
    process.exit(1)
  }
}

// Only run if this script is executed directly
if (require.main === module) {
  main()
}

export { main }