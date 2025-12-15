import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { LanguageTool, LanguageToolListParams, LanguageToolFormData } from '@/types/tool'

export function useTools(params?: LanguageToolListParams & { clusterName?: string }) {
  return useQuery({
    queryKey: ['tools', params?.clusterName, params],
    queryFn: async () => {
      const searchParams = new URLSearchParams()
      if (params?.page) searchParams.append('page', params.page.toString())
      if (params?.limit) searchParams.append('limit', params.limit.toString())
      if (params?.search) searchParams.append('search', params.search)
      if (params?.type?.length) {
        params.type.forEach(t => searchParams.append('type', t))
      }
      if (params?.phase?.length) {
        params.phase.forEach(p => searchParams.append('phase', p))
      }
      if (params?.sortBy) searchParams.append('sortBy', params.sortBy)
      if (params?.sortOrder) searchParams.append('sortOrder', params.sortOrder)

      // Use cluster-scoped API if cluster name is provided
      const endpoint = params?.clusterName 
        ? `/api/clusters/${params.clusterName}/tools?${searchParams}`
        : `/api/tools?${searchParams}` // Legacy fallback for non-cluster contexts

      const response = await fetch(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch tools')
      }
      return response.json()
    },
    refetchInterval: 5000,
  })
}

export function useTool(name: string) {
  return useQuery({
    queryKey: ['tools', name],
    queryFn: async () => {
      const response = await fetch(`/api/tools/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch tool')
      }
      return response.json()
    },
    enabled: !!name,
  })
}

export function useDeleteTool() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      const response = await fetch(`/api/tools/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete tool')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tools'] })
    },
  })
}