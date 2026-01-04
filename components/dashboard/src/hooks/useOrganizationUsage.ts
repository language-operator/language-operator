import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from 'next-auth/react'
import { useActiveOrganization } from './use-organizations'
import type { OrganizationUsage, UsageApiResponse, ResourceMetrics } from '@/types/usage'
import { getUsageStatus, formatResourceValue, calculateUsagePercentage } from '@/types/usage'
import { QUOTA_LABELS } from '@/types/quota'

export function useOrganizationUsage() {
  const { data: session } = useSession()
  const { organization } = useActiveOrganization()

  return useQuery<OrganizationUsage>({
    queryKey: ['organization-usage', organization?.id],
    queryFn: async () => {
      if (!organization?.id) {
        throw new Error('No active organization')
      }

      const response = await fetch(`/api/organizations/${organization.id}/quota`)
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to fetch usage data')
      }

      const result: UsageApiResponse = await response.json()
      return result.data
    },
    enabled: !!session?.user && !!organization?.id,
    staleTime: 30 * 1000, // 30 seconds
    refetchInterval: 60 * 1000, // Refresh every minute for real-time updates
    refetchOnWindowFocus: true,
  })
}

export function useResourceMetrics() {
  const { data: usage, isLoading, error } = useOrganizationUsage()

  if (isLoading || error || !usage) {
    return {
      metrics: null,
      isLoading,
      error: error as Error | null
    }
  }

  const { quota, used, percentUsed } = usage

  // Helper function to create usage card data
  const createUsageCard = (quotaKey: keyof typeof quota) => {
    const current = used[quotaKey] || '0'
    const limit = quota[quotaKey] || '0'
    const percentage = percentUsed[quotaKey] || 0
    
    return {
      resource: QUOTA_LABELS[quotaKey] || quotaKey,
      current: quotaKey.includes('cpu') || quotaKey.includes('memory') 
        ? formatResourceValue(current) 
        : current,
      limit: quotaKey.includes('cpu') || quotaKey.includes('memory') 
        ? formatResourceValue(limit) 
        : limit,
      percentage,
      status: getUsageStatus(percentage),
      description: `${percentage.toFixed(1)}% of allocated ${QUOTA_LABELS[quotaKey]?.toLowerCase() || quotaKey}`
    }
  }

  const metrics: ResourceMetrics = {
    cpu: createUsageCard('requests.cpu'),
    memory: createUsageCard('requests.memory'),
    agents: createUsageCard('count/languageagents'),
    tools: createUsageCard('count/languagetools'),
    models: createUsageCard('count/languagemodels'),
    members: createUsageCard('count/members')
  }

  return {
    metrics,
    isLoading: false,
    error: null
  }
}

export function useUpdateOrganizationQuota() {
  const queryClient = useQueryClient()
  const { organization } = useActiveOrganization()

  return useMutation({
    mutationFn: async (data: { plan?: string; quotas?: Record<string, string> }) => {
      if (!organization?.id) {
        throw new Error('No active organization')
      }

      const response = await fetch(`/api/organizations/${organization.id}/quota`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      })

      if (!response.ok) {
        const errorData = await response.json()
        
        if (response.status === 403) {
          throw new Error('You do not have permission to modify quotas. Contact your organization administrator.')
        } else if (response.status === 400) {
          throw new Error(errorData.details ? errorData.details.join(', ') : errorData.error)
        } else {
          throw new Error(errorData.error || 'Failed to update quota')
        }
      }

      return response.json()
    },
    onSuccess: () => {
      // Invalidate usage data to refetch updated values
      queryClient.invalidateQueries({ queryKey: ['organization-usage', organization?.id] })
    }
  })
}

export function useUsageWarnings() {
  const { data: usage } = useOrganizationUsage()
  
  return {
    warnings: usage?.warnings || [],
    isNearLimit: usage?.isNearLimit || false,
    hasWarnings: (usage?.warnings?.length || 0) > 0
  }
}

// Helper hook for getting plan display information
export function usePlanInfo() {
  const { organization } = useActiveOrganization()
  
  const planDisplayNames = {
    free: 'Free',
    pro: 'Professional', 
    enterprise: 'Enterprise',
    custom: 'Custom'
  }
  
  const planDescriptions = {
    free: 'Perfect for getting started with AI agents',
    pro: 'Advanced features for growing teams',
    enterprise: 'Full-scale deployment capabilities',
    custom: 'Tailored resources for your organization'
  }
  
  return {
    currentPlan: organization?.plan || 'free',
    displayName: planDisplayNames[organization?.plan as keyof typeof planDisplayNames] || 'Free',
    description: planDescriptions[organization?.plan as keyof typeof planDescriptions] || planDescriptions.free,
    isUpgradeable: organization?.plan === 'free' || organization?.plan === 'pro'
  }
}