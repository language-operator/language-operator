import { useQuery } from '@tanstack/react-query'
import { LanguagePersona } from '@/lib/kubernetes'

export function usePersonas() {
  return useQuery({
    queryKey: ['personas'],
    queryFn: async () => {
      const response = await fetch('/api/personas')
      if (!response.ok) {
        throw new Error('Failed to fetch personas')
      }
      const data = await response.json()
      return data.personas as LanguagePersona[]
    },
    refetchInterval: 5000, // Refetch every 5 seconds for real-time updates
  })
}