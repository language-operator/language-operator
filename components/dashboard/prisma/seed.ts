import { PrismaClient } from '@prisma/client'

const prisma = new PrismaClient()

async function main() {
  console.log('🌱 Seeding database...')

  // Create a test user
  const user = await prisma.user.upsert({
    where: { email: 'admin@langop.io' },
    update: {},
    create: {
      email: 'admin@langop.io',
      name: 'Admin User',
      emailVerified: new Date(),
    },
  })

  console.log('✅ Created user:', user.email)

  // Create organization with org-james namespace to match existing models
  const organization = await prisma.organization.upsert({
    where: { namespace: 'org-james' },
    update: {},
    create: {
      name: 'Default Organization',
      slug: 'default-org',
      namespace: 'org-james', // This matches where our models exist
      plan: 'free',
    },
  })

  console.log('✅ Created organization:', organization.name, 'with namespace:', organization.namespace)

  // Create organization membership
  const membership = await prisma.organizationMember.upsert({
    where: { 
      organizationId_userId: {
        organizationId: organization.id,
        userId: user.id,
      }
    },
    update: {},
    create: {
      organizationId: organization.id,
      userId: user.id,
      role: 'owner',
    },
  })

  console.log('✅ Created membership for user:', user.email, 'in organization:', organization.name)

  console.log('🎉 Database seeded successfully!')
}

main()
  .then(async () => {
    await prisma.$disconnect()
  })
  .catch(async (e) => {
    console.error('❌ Error seeding database:', e)
    await prisma.$disconnect()
    process.exit(1)
  })