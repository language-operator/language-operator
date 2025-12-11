import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { LanguageCluster, LanguageClusterListParams, LanguageClusterFormData } from '@/types/cluster'

export function useClusters(params?: LanguageClusterListParams) {
  return useQuery({
    queryKey: ['clusters', params],
    queryFn: async () => {
      const searchParams = new URLSearchParams()
      if (params?.page) searchParams.append('page', params.page.toString())
      if (params?.limit) searchParams.append('limit', params.limit.toString())
      if (params?.search) searchParams.append('search', params.search)
      if (params?.phase?.length) {
        params.phase.forEach(p => searchParams.append('phase', p))
      }
      if (params?.sortBy) searchParams.append('sortBy', params.sortBy)
      if (params?.sortOrder) searchParams.append('sortOrder', params.sortOrder)

      const response = await fetch(`/api/clusters?${searchParams}`)
      if (!response.ok) {
        throw new Error('Failed to fetch clusters')
      }
      return response.json()
    },
    refetchInterval: 5000,
  })
}

export function useCluster(name: string) {
  return useQuery({
    queryKey: ['clusters', name],
    queryFn: async () => {
      const response = await fetch(`/api/clusters/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch cluster')
      }
      return response.json()
    },
    enabled: !!name,
  })
}

export function useDeleteCluster() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      const response = await fetch(`/api/clusters/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete cluster')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
    },
  })
}