import { useQuery } from '@tanstack/react-query'

export interface Activity {
  id: string
  type: 'agent' | 'model' | 'tool' | 'persona' | 'cluster'
  action: 'created' | 'updated' | 'scaled' | 'failed' | 'ready' | string
  resourceName: string
  namespace: string
  message: string
  timestamp: string
  reason: string
  source: string
}

export interface RecentActivityResponse {
  success: boolean
  data: Activity[]
  total: number
  namespace: string
}

export interface UseRecentActivityOptions {
  limit?: number
  refetchInterval?: number
  enabled?: boolean
}

export function useRecentActivity(options: UseRecentActivityOptions = {}) {
  const { 
    limit = 10, 
    refetchInterval = 30000, // Refetch every 30 seconds
    enabled = true 
  } = options

  return useQuery<RecentActivityResponse>({
    queryKey: ['recent-activity', limit],
    queryFn: async () => {
      const searchParams = new URLSearchParams({
        limit: limit.toString(),
      })

      const response = await fetch(`/api/activity/recent?${searchParams}`)
      if (!response.ok) {
        throw new Error('Failed to fetch recent activity')
      }
      return response.json()
    },
    refetchInterval,
    enabled,
    staleTime: 10000, // Consider data stale after 10 seconds
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes
  })
}