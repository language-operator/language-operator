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
  enabled?: boolean
}

export function useRecentActivity(options: UseRecentActivityOptions = {}) {
  const { 
    limit = 10, 
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
    enabled,
    staleTime: 5 * 60 * 1000, // 5 minutes - activity updates via events now
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes
  })
}