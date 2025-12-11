import { useQuery } from '@tanstack/react-query'
import { LanguageTool } from '@/lib/kubernetes'

export function useTools() {
  return useQuery({
    queryKey: ['tools'],
    queryFn: async () => {
      const response = await fetch('/api/tools')
      if (!response.ok) {
        throw new Error('Failed to fetch tools')
      }
      const data = await response.json()
      return data.tools as LanguageTool[]
    },
    refetchInterval: 5000, // Refetch every 5 seconds for real-time updates
  })
}