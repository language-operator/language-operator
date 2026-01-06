/**
 * Organization-scoped layout for RESTful URL-based organization routing
 * 
 * This layout handles:
 * - Organization validation and access control
 * - Organization context provision to child components  
 * - Redirection to accessible organizations for unauthorized access
 * - Integration with the existing authenticated layout
 */

import { redirect, notFound } from 'next/navigation'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { OrganizationProvider } from '@/components/organization-provider'
import { ClusterProvider } from '@/contexts/cluster-context'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'

interface OrganizationLayoutProps {
  children: React.ReactNode
  params: Promise<{ org_id: string }>
}

export default async function OrganizationLayout({
  children,
  params
}: OrganizationLayoutProps) {
  const session = await getServerSession(authOptions)
  
  if (!session?.user?.id) {
    redirect('/login')
  }

  const resolvedParams = await params
  const orgId = resolvedParams.org_id

  // Note: Organization ID format validation is handled by middleware
  // If the request reaches this layout, the org ID format is already validated

  // Check if user has access to this organization
  const membership = await db.organizationMember.findUnique({
    where: {
      organizationId_userId: {
        organizationId: orgId,
        userId: session.user.id
      }
    },
    include: {
      organization: true
    }
  })

  if (!membership) {
    // User doesn't have access to this organization
    // Try to redirect to their first accessible organization
    const userOrganizations = await db.organizationMember.findMany({
      where: { userId: session.user.id },
      include: { organization: true },
      orderBy: { createdAt: 'asc' }
    })

    if (userOrganizations.length > 0) {
      // Preserve the current path when redirecting to accessible organization
      const pathSegments = decodeURIComponent(new URL('http://localhost' + '/').pathname)
        .split('/')
        .filter(Boolean)
        .slice(1) // Remove the org_id segment
        
      const newPath = pathSegments.length > 0 ? `/${pathSegments.join('/')}` : ''
      redirect(`/${userOrganizations[0].organizationId}${newPath}`)
    } else {
      // No organizations at all - redirect to organization management
      redirect('/settings/organizations')
    }
  }

  // Organization exists and user has access
  const organization = membership.organization
  const userRole = membership.role

  return (
    <OrganizationProvider 
      organization={organization}
      userRole={userRole}
      orgId={orgId}
    >
      <ClusterProvider>
        <div className="flex h-screen overflow-hidden bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950">
          <Sidebar />
          <div className="flex flex-1 flex-col overflow-hidden">
            <Header />
            <main className="flex-1 overflow-y-auto p-generous relative">
              {children}
            </main>
          </div>
        </div>
      </ClusterProvider>
    </OrganizationProvider>
  )
}

// Generate static params for build optimization
export async function generateStaticParams() {
  // In production, you might want to generate params for common organizations
  // For now, return empty array to use dynamic routing
  return []
}