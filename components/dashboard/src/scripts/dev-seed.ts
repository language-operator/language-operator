#!/usr/bin/env npx tsx

/**
 * Development seed script
 * 
 * Creates the default admin user for local development
 * Only runs if the database is empty (no users exist)
 * 
 * Usage: npm run dev:seed
 */

import bcrypt from 'bcryptjs'
import { db } from '../lib/db'
import { k8sClient } from '../lib/k8s-client'
import { generateOrganizationNamespace } from '../lib/namespace-utils'

const DEV_ADMIN = {
  name: 'Development Admin',
  email: 'james@theryans.io',
  password: 'password123' // Plain text - will be hashed
}

async function seedDevDatabase() {
  console.log('🌱 [DEV-SEED] Seeding development database...')
  
  try {
    // Check if any users already exist
    const userCount = await db.user.count()
    
    if (userCount > 0) {
      console.log(`ℹ️ [DEV-SEED] Database already has ${userCount} users, skipping seed`)
      return false
    }
    
    console.log('🔐 [DEV-SEED] Hashing development password...')
    const passwordHash = await bcrypt.hash(DEV_ADMIN.password, 12)
    
    await db.$transaction(async (tx) => {
      // Create development admin user
      const user = await tx.user.create({
        data: {
          name: DEV_ADMIN.name,
          email: DEV_ADMIN.email,
          password: passwordHash,
          emailVerified: new Date(), // Mark as verified for development
        }
      })
      
      console.log(`👤 [DEV-SEED] Created development admin user: ${user.email}`)
      
      // Create default development organization
      const orgNamespace = generateOrganizationNamespace()
      const organization = await tx.organization.create({
        data: {
          name: "Development Organization",
          slug: "development", 
          namespace: orgNamespace,
          plan: "free",
        }
      })
      
      console.log(`🏢 [DEV-SEED] Created development organization: ${organization.name} (${orgNamespace})`)
      
      // Try to create Kubernetes namespace (optional for dev)
      try {
        await k8sClient.createOrganizationNamespace(orgNamespace, organization.id, organization.plan)
        console.log(`☸️ [DEV-SEED] Created Kubernetes namespace: ${orgNamespace}`)
      } catch (err: any) {
        console.log(`⚠️ [DEV-SEED] Could not create K8s namespace (OK for dev): ${err.message}`)
      }
      
      // Add admin user as organization owner
      await tx.organizationMember.create({
        data: {
          userId: user.id,
          organizationId: organization.id,
          role: "owner"
        }
      })
      
      console.log(`👑 [DEV-SEED] Added ${user.email} as owner of ${organization.name}`)
    })
    
    console.log('✅ [DEV-SEED] Development database seeded successfully!')
    console.log(`🔑 [DEV-SEED] Login credentials: ${DEV_ADMIN.email} / ${DEV_ADMIN.password}`)
    
    return true
    
  } catch (error) {
    console.error('❌ [DEV-SEED] Failed to seed development database:', error)
    throw error
  }
}

async function main() {
  try {
    await seedDevDatabase()
    process.exit(0)
  } catch (error) {
    console.error('💥 [DEV-SEED] Seed script failed:', error)
    process.exit(1)
  }
}

// Only run if this script is executed directly
if (require.main === module) {
  main()
}

export { seedDevDatabase }