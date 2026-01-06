import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from 'next-auth/react'
import { useOrganization } from '@/components/organization-provider'
import type { OrganizationUsage, UsageApiResponse, ResourceMetrics, ComputeMemoryCardData } from '@/types/usage'
import { getUsageStatus, formatResourceValue, calculateUsagePercentage } from '@/types/usage'
import { QUOTA_LABELS } from '@/types/quota'

export function useOrganizationUsage() {
  const { data: session } = useSession()
  const { organization } = useOrganization()

  return useQuery<OrganizationUsage>({
    queryKey: ['organization-usage', organization?.id],
    queryFn: async () => {
      if (!organization?.id) {
        throw new Error('No active organization')
      }

      const response = await fetch(`/api/${organization.id}/quota`)
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
    const percentage = percentUsed[quotaKey] || 0  // Use corrected backend percentage
    
    return {
      resource: QUOTA_LABELS[quotaKey] || quotaKey,
      current: quotaKey.includes('cpu') || quotaKey.includes('memory') 
        ? formatResourceValue(current) 
        : current,
      limit: quotaKey.includes('cpu') || quotaKey.includes('memory') 
        ? formatResourceValue(limit) 
        : limit,
      percentage,
      status: getUsageStatus(percentage)
    }
  }

  // Helper function to create compute/memory cards with requests and limits
  const createComputeMemoryCard = (resourceType: 'cpu' | 'memory') => {
    const requestsKey = `requests.${resourceType}` as keyof typeof quota
    const limitsKey = `limits.${resourceType}` as keyof typeof quota
    
    const requestsCurrent = used[requestsKey] || '0'
    const requestsLimit = quota[requestsKey] || '0'
    const requestsPercentage = percentUsed[requestsKey] || 0  // Use corrected backend percentage
    
    const limitsCurrent = used[limitsKey] || '0'
    const limitsLimit = quota[limitsKey] || '0'
    const limitsPercentage = percentUsed[limitsKey] || 0  // Use corrected backend percentage
    
    return {
      resource: resourceType.toUpperCase(),
      requests: {
        current: formatResourceValue(requestsCurrent),
        limit: formatResourceValue(requestsLimit),
        percentage: requestsPercentage,
        status: getUsageStatus(requestsPercentage)
      },
      limits: {
        current: formatResourceValue(limitsCurrent),
        limit: formatResourceValue(limitsLimit),
        percentage: limitsPercentage,
        status: getUsageStatus(limitsPercentage)
      }
    }
  }

  const metrics: ResourceMetrics = {
    compute: createComputeMemoryCard('cpu'),
    memory: createComputeMemoryCard('memory'),
    clusters: createUsageCard('count/languageclusters'),
    agents: createUsageCard('count/languageagents'),
    tools: createUsageCard('count/languagetools'),
    models: createUsageCard('count/languagemodels'),
    personas: createUsageCard('count/languagepersonas')
  }

  return {
    metrics,
    isLoading: false,
    error: null
  }
}

export function useUpdateOrganizationQuota() {
  const queryClient = useQueryClient()
  const { organization } = useOrganization()

  return useMutation({
    mutationFn: async (data: { plan?: string; quotas?: Record<string, string> }) => {
      if (!organization?.id) {
        throw new Error('No active organization')
      }

      const response = await fetch(`/api/${organization.id}/quota`, {
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
  const { organization } = useOrganization()
  const { data: usage } = useOrganizationUsage()
  
  const planDisplayNames = {
    free: 'Free',
    pro: 'Professional', 
    enterprise: 'Enterprise',
    custom: 'Custom'
  }
  
  const planDescriptions = {
    free: 'Perfect for exploring the frontier',
    pro: 'Advanced features for growing teams',
    enterprise: 'Full-scale deployment capabilities',
    custom: 'Tailored resources for your organization'
  }

  // Derive plan features from actual quota data
  const getPlanFeatures = () => {
    if (!usage?.quota) return []
    
    const features = []
    
    // Clusters
    if (usage.quota['count/languageclusters']) {
      const count = usage.quota['count/languageclusters']
      features.push(`${count} clusters`)
    }
    
    // Agents
    if (usage.quota['count/languageagents']) {
      const count = usage.quota['count/languageagents']
      features.push(`${count} agents`)
    }
    
    // Models
    if (usage.quota['count/languagemodels']) {
      const count = usage.quota['count/languagemodels']
      features.push(`${count} models`)
    }
    
    // Tools
    if (usage.quota['count/languagetools']) {
      const count = usage.quota['count/languagetools']
      features.push(`${count} tools`)
    }
    
    // Personas
    if (usage.quota['count/languagepersonas']) {
      const count = usage.quota['count/languagepersonas']
      features.push(`${count} personas`)
    }
    
    // Memory (convert from Gi to GB for display)
    if (usage.quota['limits.memory']) {
      const memory = usage.quota['limits.memory']
      if (memory.includes('Gi')) {
        const gb = parseInt(memory.replace('Gi', ''))
        features.push(`${gb} GB memory`)
      } else if (memory.includes('Mi')) {
        const mb = parseInt(memory.replace('Mi', ''))
        const gb = Math.round(mb / 1024)
        features.push(`${gb} GB memory`)
      }
    }
    
    // CPU (convert from millicores to cores if needed)
    if (usage.quota['limits.cpu']) {
      const cpu = usage.quota['limits.cpu']
      if (cpu.includes('m')) {
        const millicores = parseInt(cpu.replace('m', ''))
        const cores = millicores / 1000
        features.push(`${cores} CPU cores`)
      } else {
        features.push(`${cpu} CPU cores`)
      }
    }
    
    return features
  }
  
  return {
    currentPlan: organization?.plan || 'free',
    displayName: planDisplayNames[organization?.plan as keyof typeof planDisplayNames] || 'Free',
    description: planDescriptions[organization?.plan as keyof typeof planDescriptions] || planDescriptions.free,
    isUpgradeable: organization?.plan === 'free' || organization?.plan === 'pro',
    features: getPlanFeatures()
  }
}