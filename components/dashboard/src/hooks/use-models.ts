import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganizationStore } from '@/store/organization-store'
import { LanguageModel, LanguageModelListParams, LanguageModelFormData } from '@/types/model'

export function useModels(params?: LanguageModelListParams & { clusterName?: string }) {
  const { activeOrganizationId } = useOrganizationStore()
  
  return useQuery({
    queryKey: ['models', activeOrganizationId, params?.clusterName, params],
    queryFn: async () => {
      const searchParams = new URLSearchParams()
      if (params?.page) searchParams.append('page', params.page.toString())
      if (params?.limit) searchParams.append('limit', params.limit.toString())
      if (params?.search) searchParams.append('search', params.search)
      if (params?.provider?.length) {
        params.provider.forEach(p => searchParams.append('provider', p))
      }
      if (params?.phase?.length) {
        params.phase.forEach(p => searchParams.append('phase', p))
      }
      if (params?.sortBy) searchParams.append('sortBy', params.sortBy)
      if (params?.sortOrder) searchParams.append('sortOrder', params.sortOrder)
      if (params?.healthy !== undefined) searchParams.append('healthy', params.healthy.toString())

      // Use cluster-scoped API if cluster name is provided
      const endpoint = params?.clusterName 
        ? `/api/clusters/${params.clusterName}/models?${searchParams}`
        : `/api/models?${searchParams}` // Legacy fallback for non-cluster contexts

      const response = await fetchWithOrganization(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch models')
      }
      return response.json()
    },
  })
}

export function useModel(name: string) {
  return useQuery({
    queryKey: ['models', name],
    queryFn: async () => {
      const response = await fetch(`/api/models/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch model')
      }
      return response.json()
    },
    enabled: !!name,
  })
}

export function useDeleteModel() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      const response = await fetch(`/api/models/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete model')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['models'] })
    },
  })
}