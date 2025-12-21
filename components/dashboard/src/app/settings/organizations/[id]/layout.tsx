'use client'

import { useParams, usePathname } from 'next/navigation'
import Link from 'next/link'
import { Settings, Users, BarChart3, Building2, Copy } from 'lucide-react'
import { ResourceHeader } from '@/components/ui/resource-header'
import { cn } from '@/lib/utils'
import { useOrganization } from '@/hooks/use-organizations'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'

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
      name: 'Usage Limits',
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

  const handleCopyNamespace = () => {
    if (organization?.namespace) {
      navigator.clipboard.writeText(organization.namespace)
      toast.success('Namespace copied to clipboard')
    }
  }

  return (
    <div className="space-y-6">
      {/* Resource Header */}
      {organization && (
        <ResourceHeader
          backHref="/settings/organizations"
          backLabel="Back to Organizations"
          icon={Building2}
          iconColor="text-blue-600"
          title={organization.name}
          subtitle={
            <div className="flex items-center gap-2">
              <span>Organization</span>
              <span className="text-gray-400">|</span>
              <span className="font-mono">{organization.namespace}</span>
              <Button
                variant="ghost"
                size="sm"
                className="h-6 w-6 p-0"
                onClick={handleCopyNamespace}
              >
                <Copy className="h-3 w-3" />
              </Button>
            </div>
          }
        />
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

      {/* Page Content */}
      {children}
    </div>
  )
}