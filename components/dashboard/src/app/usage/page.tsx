'use client'

import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { UsageDashboard } from '@/components/usage/usage-dashboard'

export default function UsagePage() {
  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Hero Section with consistent styling to other pages */}
        <div>
          <h1 className="text-3xl font-bold font-mono">Resource Usage</h1>
          <p className="text-stone-600 dark:text-stone-400 mt-2">
            Monitor your organization's resource consumption and plan limits
          </p>
        </div>

        {/* Usage Dashboard Component */}
        <UsageDashboard />
      </div>
    </AuthenticatedLayout>
  )
}