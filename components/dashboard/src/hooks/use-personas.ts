import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { LanguagePersona, LanguagePersonaListParams, LanguagePersonaFormData } from '@/types/persona'

export function usePersonas(params?: LanguagePersonaListParams & { clusterName?: string }) {
  return useQuery({
    queryKey: ['personas', params?.clusterName, params],
    queryFn: async () => {
      const searchParams = new URLSearchParams()
      if (params?.page) searchParams.append('page', params.page.toString())
      if (params?.limit) searchParams.append('limit', params.limit.toString())
      if (params?.search) searchParams.append('search', params.search)
      if (params?.tone?.length) {
        params.tone.forEach(t => searchParams.append('tone', t))
      }
      if (params?.sortBy) searchParams.append('sortBy', params.sortBy)
      if (params?.sortOrder) searchParams.append('sortOrder', params.sortOrder)

      // Use cluster-scoped API if cluster name is provided
      const endpoint = params?.clusterName 
        ? `/api/clusters/${params.clusterName}/personas?${searchParams}`
        : `/api/personas?${searchParams}` // Legacy fallback for non-cluster contexts

      const response = await fetch(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch personas')
      }
      return response.json()
    },
  })
}

export function usePersona(name: string) {
  return useQuery({
    queryKey: ['personas', name],
    queryFn: async () => {
      const response = await fetch(`/api/personas/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch persona')
      }
      return response.json()
    },
    enabled: !!name,
  })
}

export function useDeletePersona() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      const response = await fetch(`/api/personas/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete persona')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['personas'] })
    },
  })
}