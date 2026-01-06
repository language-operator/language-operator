'use client'

import { useParams, usePathname, useRouter } from 'next/navigation'
import { Settings, BarChart3, Building2, Copy } from 'lucide-react'
import { ResourceHeader } from '@/components/ui/resource-header'
import { useOrganization } from '@/hooks/use-organizations'
import { useOrganization as useOrgContext } from '@/components/organization-provider'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface OrganizationLayoutProps {
  children: React.ReactNode
}

export default function OrganizationLayout({ children }: OrganizationLayoutProps) {
  const params = useParams()
  const pathname = usePathname()
  const router = useRouter()
  const organizationId = params.id as string
  const { getOrgUrl } = useOrgContext()
  
  const { data: organizationData, isLoading } = useOrganization(organizationId)
  const organization = organizationData?.organization

  // Determine current tab based on pathname
  const currentTab = pathname.endsWith('/edit') ? 'usage-limits' : 'general'

  const handleTabChange = (value: string) => {
    if (value === 'general') {
      router.push(getOrgUrl(`/settings/organizations/${organizationId}`))
    } else if (value === 'usage-limits') {
      router.push(getOrgUrl(`/settings/organizations/${organizationId}/edit`))
    }
  }

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
          backHref={getOrgUrl("/settings/organizations")}
          backLabel="Back to Organizations"
          icon={Building2}
          title={organization.name}
          subtitle={
            <div className="flex items-center gap-2">
              <span>Organization</span>
              <span className="text-stone-400 dark:text-stone-500">|</span>
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
          <div className="h-8 bg-stone-200 rounded w-1/4 animate-pulse dark:bg-stone-700"></div>
          <div className="h-4 bg-stone-200 rounded w-1/3 animate-pulse dark:bg-stone-700"></div>
        </div>
      )}

      {/* Tabs Navigation */}
      {organization && (
        <Tabs value={currentTab} onValueChange={handleTabChange}>
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="general" className="flex items-center gap-2">
              <Settings className="w-4 h-4" />
              General
            </TabsTrigger>
            <TabsTrigger value="usage-limits" className="flex items-center gap-2">
              <BarChart3 className="w-4 h-4" />
              Usage Limits
            </TabsTrigger>
          </TabsList>
          
          <TabsContent value="general" className="mt-6">
            {currentTab === 'general' && children}
          </TabsContent>
          <TabsContent value="usage-limits" className="mt-6">
            {currentTab === 'usage-limits' && children}
          </TabsContent>
        </Tabs>
      )}
    </div>
  )
}