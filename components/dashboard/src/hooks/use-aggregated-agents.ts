import { useQuery } from '@tanstack/react-query'
import { useClusters } from '@/hooks/use-clusters'
import { useAgents } from '@/hooks/use-agents'
import { fetchWithOrganization } from '@/lib/api-client'
import { LanguageCluster } from '@/types/cluster'

export function useAggregatedAgents(limit?: number) {
  // First get all clusters
  const { data: clustersData, isLoading: clustersLoading, error: clustersError } = useClusters()
  const clusters = clustersData?.data || []

  return useQuery({
    queryKey: ['aggregated-agents', limit, clusters.map((c: LanguageCluster) => c.metadata.name)],
    queryFn: async () => {
      if (!clusters.length) {
        return { data: [], total: 0 }
      }

      // Fetch agents from each cluster
      const agentPromises = clusters.map(async (cluster: LanguageCluster) => {
        try {
          const response = await fetchWithOrganization(`/api/clusters/${cluster.metadata.name}/agents?limit=${limit || 50}`)
          if (!response.ok) {
            console.warn(`Failed to fetch agents from cluster ${cluster.metadata.name}`)
            return []
          }
          const data = await response.json()
          return data.data || []
        } catch (error) {
          console.warn(`Error fetching agents from cluster ${cluster.metadata.name}:`, error)
          return []
        }
      })

      const agentArrays = await Promise.all(agentPromises)
      const allAgents = agentArrays.flat()

      // Sort by creation time (newest first) and apply limit
      const sortedAgents = allAgents
        .sort((a, b) => {
          const aTime = new Date(a.metadata?.creationTimestamp || 0).getTime()
          const bTime = new Date(b.metadata?.creationTimestamp || 0).getTime()
          return bTime - aTime
        })
        .slice(0, limit || 50)

      return {
        data: sortedAgents,
        total: allAgents.length
      }
    },
    enabled: !clustersLoading && !clustersError && clusters.length > 0,
    refetchInterval: 30000, // Refetch every 30 seconds
  })
}