import { useQuery } from '@tanstack/react-query'
import { LanguageCluster } from '@/lib/kubernetes'

export function useClusters() {
  return useQuery({
    queryKey: ['clusters'],
    queryFn: async () => {
      const response = await fetch('/api/clusters')
      if (!response.ok) {
        throw new Error('Failed to fetch clusters')
      }
      const data = await response.json()
      return data.clusters as LanguageCluster[]
    },
    refetchInterval: 5000, // Refetch every 5 seconds for real-time updates
  })
}