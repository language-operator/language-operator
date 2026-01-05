'use client'

import Link from 'next/link'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { UsageDashboard } from '@/components/usage/usage-dashboard'
import { ResourceHeader } from '@/components/ui/resource-header'
import { CapacityIcon } from '@/components/ui/icons'
import { Button } from '@/components/ui/button'
import { useActiveOrganization } from '@/hooks/use-organizations'
import { Settings } from 'lucide-react'

export default function UsagePage() {
  const { organization } = useActiveOrganization()

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Standard Resource Header */}
        <ResourceHeader
          icon={CapacityIcon}
          title="Capacity"
          subtitle="Monitor your organization's resource consumption and plan limits"
          actions={
            organization?.id && (
              <Link href={`/settings/organizations/${organization.id}/edit`}>
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
    </AuthenticatedLayout>
  )
}