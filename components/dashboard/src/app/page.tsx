import { redirect } from 'next/navigation'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'

/**
 * Root page redirects users directly to their organization homepage
 * The organization homepage is the main dashboard within organization context
 */
export default async function RootPage() {
  const session = await getServerSession(authOptions)
  
  if (!session?.user?.id) {
    redirect('/login')
  }

  // Get user's accessible organizations
  const userOrganizations = await db.organizationMember.findMany({
    where: { userId: session.user.id },
    include: { organization: true },
    orderBy: { createdAt: 'asc' }
  })

  if (userOrganizations.length > 0) {
    // Redirect to first accessible organization's dashboard
    // This shows the same System Overview dashboard but within organization context
    redirect(`/${userOrganizations[0].organizationId}`)
  } else {
    // No organizations - redirect to organization management to create/join one
    redirect('/settings/organizations')
  }
}