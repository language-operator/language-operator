'use client'

import Link from 'next/link'
import { useOrganization } from '@/components/organization-provider'
import { UsageDashboard } from '@/components/usage/usage-dashboard'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Server } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Settings } from 'lucide-react'

export default function UsagePage() {
  const { organization, getOrgUrl } = useOrganization()

  return (
    <div className="space-y-6">
        {/* Standard Resource Header */}
        <ResourceHeader
          icon={Server}
          title="Capacity"
          subtitle="Monitor your organization's resource consumption and plan limits"
          actions={
            organization?.id && (
              <Link href={getOrgUrl(`/settings/organizations/${organization.id}/edit`)}>
                <Button variant="outline">
                  <Settings className="h-4 w-4 mr-2" />
                  Edit Quotas
                </Button>
              </Link>
            )
          }
        />

        {/* Usage Dashboard Component */}
        <UsageDashboard />
      </div>
  )
}