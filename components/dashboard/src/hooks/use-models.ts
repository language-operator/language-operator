import { useQuery } from '@tanstack/react-query'
import { LanguageModel } from '@/lib/kubernetes'

export function useModels() {
  return useQuery({
    queryKey: ['models'],
    queryFn: async () => {
      const response = await fetch('/api/models')
      if (!response.ok) {
        throw new Error('Failed to fetch models')
      }
      const data = await response.json()
      return data.models as LanguageModel[]
    },
    refetchInterval: 5000, // Refetch every 5 seconds for real-time updates
  })
}