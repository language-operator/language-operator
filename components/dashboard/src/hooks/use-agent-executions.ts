import { useQuery } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
// Local type definitions
interface AgentExecution {
  traceId: string
  executionId: string
  startTime: Date
  endTime: Date
  duration: number
  status: 'success' | 'error' | 'running'
  rootSpanName: string
  spanCount: number
}

interface AgentExecutionsResponse {
  success: boolean
  data: AgentExecution[]
}

export function useAgentExecutions(agentName: string, clusterName: string, options?: {
  limit?: number
  timeRange?: number
  enabled?: boolean
}) {
  const { limit = 50, timeRange = 24 * 60 * 60 * 1000, enabled = true } = options || {}
  
  return useQuery<AgentExecutionsResponse>({
    queryKey: ['agent-executions', clusterName, agentName, { limit, timeRange }],
    queryFn: async () => {
      const searchParams = new URLSearchParams({
        limit: limit.toString(),
        timeRange: timeRange.toString(),
      })

      const response = await fetchWithOrganization(
        `/api/clusters/${clusterName}/agents/${agentName}/executions?${searchParams}`
      )

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `Failed to fetch agent executions: ${response.status}`)
      }

      return response.json()
    },
    enabled: enabled && !!agentName && !!clusterName,
    staleTime: 30 * 1000, // 30 seconds
    refetchInterval: 60 * 1000, // 1 minute
    retry: 2,
  })
}