'use client'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertTriangle, Loader2 } from 'lucide-react'
import { ResourceUsageCard } from './resource-usage-card'
import { PlanUpgradeBanner } from './plan-upgrade-banner'
import { useResourceMetrics, useUsageWarnings } from '@/hooks/useOrganizationUsage'

function LoadingSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="space-y-6 p-8 bg-white/95 border border-stone-800/90 dark:bg-stone-900/95 dark:border-stone-700/90">
          <div className="space-y-4">
            <Skeleton className="h-4 w-20" />
          </div>
          <div className="space-y-6">
            <div className="flex items-end gap-2">
              <Skeleton className="h-8 w-16" />
              <Skeleton className="h-6 w-12" />
            </div>
            <Skeleton className="h-1 w-full" />
            <div className="flex justify-between">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-12" />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

function ErrorState({ error }: { error: Error }) {
  return (
    <Alert className="max-w-2xl mx-auto">
      <AlertTriangle className="h-4 w-4" />
      <AlertDescription className="font-light">
        Failed to load usage data: {error.message}
      </AlertDescription>
    </Alert>
  )
}

function UsageWarnings() {
  const { warnings, hasWarnings } = useUsageWarnings()

  if (!hasWarnings) return null

  return (
    <Alert className="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/10">
      <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
      <AlertDescription className="font-light text-amber-800 dark:text-amber-200">
        <div className="space-y-2">
          <p className="text-sm font-light">Resource usage approaching limits:</p>
          <ul className="space-y-1">
            {warnings.map((warning, index) => (
              <li key={index} className="text-xs font-light">
                • {warning}
              </li>
            ))}
          </ul>
        </div>
      </AlertDescription>
    </Alert>
  )
}

export function UsageDashboard() {
  const { metrics, isLoading, error } = useResourceMetrics()

  if (isLoading) {
    return (
      <div className="space-y-12">
        <div className="flex items-center gap-2 justify-center">
          <Loader2 className="h-4 w-4 animate-spin text-stone-600 dark:text-stone-400" />
          <span className="text-sm font-light text-stone-600 dark:text-stone-400">
            Loading usage data...
          </span>
        </div>
        <LoadingSkeleton />
      </div>
    )
  }

  if (error) {
    return <ErrorState error={error} />
  }

  if (!metrics) {
    return (
      <Alert>
        <AlertDescription className="font-light">
          No usage data available for your organization.
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-12">
      {/* Usage Warnings */}
      <UsageWarnings />

      {/* Usage Grid - Marfa 2x3 responsive layout */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
        {/* Resource Cards */}
        <ResourceUsageCard data={metrics.cpu} />
        <ResourceUsageCard data={metrics.memory} />
        <ResourceUsageCard data={metrics.agents} />
        <ResourceUsageCard data={metrics.tools} />
        <ResourceUsageCard data={metrics.models} />
        <ResourceUsageCard data={metrics.members} />
      </div>

      {/* Plan Section */}
      <PlanUpgradeBanner />
    </div>
  )
}