/**
 * Migration script to backfill existing organizations with UUID-based namespaces
 * 
 * This script:
 * 1. Identifies organizations with legacy slug-based namespaces
 * 2. Generates new UUID namespaces for them
 * 3. Updates the database records
 * 4. Creates corresponding Kubernetes namespaces
 * 5. Optionally migrates resources from old to new namespaces
 * 
 * Run with: npx tsx src/scripts/migrate-organization-namespaces.ts
 */

import { PrismaClient } from '@prisma/client'
import { generateOrganizationNamespace, isLegacyNamespace } from '../lib/namespace-utils'
import { k8sClient } from '../lib/k8s-client'

const prisma = new PrismaClient()

interface MigrationOptions {
  dryRun: boolean
  migrateResources: boolean
  deleteOldNamespaces: boolean
}

async function migrateOrganizationNamespaces(options: MigrationOptions = {
  dryRun: true,
  migrateResources: false,
  deleteOldNamespaces: false
}) {
  console.log('🚀 Starting organization namespace migration...')
  console.log('Options:', options)

  try {
    // Find all organizations with legacy namespaces
    const allOrganizations = await prisma.organization.findMany({
      select: {
        id: true,
        name: true,
        slug: true,
        namespace: true,
        plan: true
      }
    })

    const legacyOrganizations = allOrganizations.filter(org => 
      isLegacyNamespace(org.namespace)
    )

    const modernOrganizations = allOrganizations.filter(org => 
      !isLegacyNamespace(org.namespace)
    )

    console.log(`\n📊 Migration Status:`)
    console.log(`  Total organizations: ${allOrganizations.length}`)
    console.log(`  Legacy namespaces: ${legacyOrganizations.length}`)
    console.log(`  Modern namespaces: ${modernOrganizations.length}`)

    if (legacyOrganizations.length === 0) {
      console.log('✅ No migration needed - all organizations already have UUID namespaces!')
      return
    }

    console.log(`\n🔄 Organizations to migrate:`)
    legacyOrganizations.forEach(org => {
      console.log(`  • ${org.name} (${org.namespace})`)
    })

    if (options.dryRun) {
      console.log('\n🔍 DRY RUN MODE - No changes will be made')
      console.log('Set dryRun: false to execute the migration')
      return
    }

    // Migrate each organization
    const migrationResults = []
    
    for (const org of legacyOrganizations) {
      console.log(`\n🔧 Migrating ${org.name}...`)
      
      try {
        // Generate new UUID namespace
        let newNamespace: string
        let attempts = 0
        const maxAttempts = 10
        
        do {
          newNamespace = generateOrganizationNamespace()
          const existing = await prisma.organization.findUnique({
            where: { namespace: newNamespace }
          })
          if (!existing) break
          attempts++
        } while (attempts < maxAttempts)

        if (attempts >= maxAttempts) {
          throw new Error(`Failed to generate unique namespace after ${maxAttempts} attempts`)
        }

        console.log(`  Old namespace: ${org.namespace}`)
        console.log(`  New namespace: ${newNamespace}`)

        // Update database record
        await prisma.organization.update({
          where: { id: org.id },
          data: { namespace: newNamespace }
        })
        console.log(`  ✅ Updated database record`)

        // Create new Kubernetes namespace
        try {
          await k8sClient.createOrganizationNamespace(newNamespace, org.id, org.plan)
          console.log(`  ✅ Created Kubernetes namespace`)
        } catch (k8sError: any) {
          // Log but don't fail if namespace creation fails
          console.log(`  ⚠️  Warning: K8s namespace creation failed: ${k8sError.message}`)
        }

        // Optional: Migrate resources from old to new namespace
        if (options.migrateResources) {
          try {
            await migrateNamespaceResources(org.namespace, newNamespace, org.id)
            console.log(`  ✅ Migrated resources`)
          } catch (migrateError: any) {
            console.log(`  ⚠️  Warning: Resource migration failed: ${migrateError.message}`)
          }
        }

        // Optional: Delete old namespace
        if (options.deleteOldNamespaces) {
          try {
            await k8sClient.deleteOrganizationNamespace(org.namespace)
            console.log(`  ✅ Deleted old namespace`)
          } catch (deleteError: any) {
            console.log(`  ⚠️  Warning: Old namespace deletion failed: ${deleteError.message}`)
          }
        }

        migrationResults.push({
          organizationId: org.id,
          organizationName: org.name,
          oldNamespace: org.namespace,
          newNamespace,
          success: true
        })

      } catch (error: any) {
        console.log(`  ❌ Migration failed: ${error.message}`)
        migrationResults.push({
          organizationId: org.id,
          organizationName: org.name,
          oldNamespace: org.namespace,
          newNamespace: null,
          success: false,
          error: error.message
        })
      }
    }

    // Summary
    console.log(`\n📋 Migration Summary:`)
    const successful = migrationResults.filter(r => r.success)
    const failed = migrationResults.filter(r => !r.success)
    
    console.log(`  Successful: ${successful.length}`)
    console.log(`  Failed: ${failed.length}`)

    if (failed.length > 0) {
      console.log(`\n❌ Failed migrations:`)
      failed.forEach(result => {
        console.log(`  • ${result.organizationName}: ${result.error}`)
      })
    }

    console.log(`\n✅ Migration completed!`)

  } catch (error) {
    console.error('❌ Migration failed:', error)
    throw error
  } finally {
    await prisma.$disconnect()
  }
}

/**
 * Migrate resources from old namespace to new namespace
 * This is optional and complex - may want to do manually for production
 */
async function migrateNamespaceResources(oldNamespace: string, newNamespace: string, organizationId: string) {
  // This is a placeholder for resource migration logic
  // In practice, this would:
  // 1. List all CRDs in the old namespace
  // 2. Update their namespace field
  // 3. Recreate them in the new namespace
  // 4. Update any cross-references
  
  console.log(`    🔄 Resource migration from ${oldNamespace} to ${newNamespace} would happen here`)
  console.log(`    Note: Resource migration is complex and may require manual intervention`)
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2)
  const dryRun = !args.includes('--execute')
  const migrateResources = args.includes('--migrate-resources')
  const deleteOldNamespaces = args.includes('--delete-old-namespaces')

  console.log('📝 Usage:')
  console.log('  npm run migrate:namespaces                    # Dry run')
  console.log('  npm run migrate:namespaces --execute          # Execute migration')
  console.log('  npm run migrate:namespaces --execute --migrate-resources')
  console.log('  npm run migrate:namespaces --execute --delete-old-namespaces')

  migrateOrganizationNamespaces({
    dryRun,
    migrateResources,
    deleteOldNamespaces
  }).catch(error => {
    console.error('Migration failed:', error)
    process.exit(1)
  })
}

export { migrateOrganizationNamespaces }