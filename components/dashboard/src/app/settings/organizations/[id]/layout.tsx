'use client'

import { useParams, usePathname } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, Settings, Users, BarChart3 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useOrganization } from '@/hooks/use-organizations'

interface OrganizationLayoutProps {
  children: React.ReactNode
}

export default function OrganizationLayout({ children }: OrganizationLayoutProps) {
  const params = useParams()
  const pathname = usePathname()
  const organizationId = params.id as string
  
  const { data: organizationData, isLoading } = useOrganization(organizationId)
  const organization = organizationData?.organization

  const tabs = [
    {
      name: 'General',
      href: `/settings/organizations/${organizationId}`,
      icon: Settings,
      current: pathname === `/settings/organizations/${organizationId}`
    },
    {
      name: 'Quotas',
      href: `/settings/organizations/${organizationId}/edit`,
      icon: BarChart3,
      current: pathname === `/settings/organizations/${organizationId}/edit`
    },
    {
      name: 'Members',
      href: `/settings/organizations/${organizationId}/members`,
      icon: Users,
      current: pathname === `/settings/organizations/${organizationId}/members`
    }
  ]

  return (
    <div className="space-y-6">
      {/* Breadcrumb and Navigation */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/settings/organizations">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Organizations
            </Link>
          </Button>
        </div>
        
        {organization && (
          <div>
            <h1 className="text-3xl font-bold">{organization.name}</h1>
            <p className="text-gray-600">
              Namespace: <code className="bg-gray-100 px-2 py-1 rounded text-sm">{organization.namespace}</code>
            </p>
          </div>
        )}
        
        {isLoading && (
          <div className="space-y-2">
            <div className="h-8 bg-gray-200 rounded w-1/4 animate-pulse"></div>
            <div className="h-4 bg-gray-200 rounded w-1/3 animate-pulse"></div>
          </div>
        )}

        {/* Tabs Navigation */}
        {organization && (
          <div className="bg-muted text-muted-foreground inline-flex h-12 w-fit items-center justify-center rounded-lg p-1">
            {tabs.map((tab) => {
              const Icon = tab.icon
              return (
                <Link
                  key={tab.name}
                  href={tab.href}
                  className={cn(
                    'inline-flex h-[calc(100%-8px)] items-center justify-center gap-2 rounded-md border border-transparent px-4 py-2 text-sm font-medium whitespace-nowrap transition-[color,box-shadow]',
                    tab.current
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-foreground hover:bg-background/50'
                  )}
                >
                  <Icon className="w-4 h-4" />
                  {tab.name}
                </Link>
              )
            })}
          </div>
        )}
      </div>

      {/* Page Content */}
      {children}
    </div>
  )
}