import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganizationStore } from '@/store/organization-store'
import { LanguageModel, LanguageModelListParams, LanguageModelFormData } from '@/types/model'

export function useModels(params: LanguageModelListParams & { clusterName: string }) {
  const { activeOrganizationId } = useOrganizationStore()
  
  return useQuery({
    queryKey: ['models', activeOrganizationId, params.clusterName, params],
    queryFn: async () => {
      if (!params.clusterName) {
        throw new Error('Cluster name is required to fetch models')
      }

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

      const endpoint = `/api/clusters/${params.clusterName}/models?${searchParams}`

      const response = await fetchWithOrganization(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch models')
      }
      return response.json()
    },
  })
}

export function useModel(name: string, clusterName: string) {
  return useQuery({
    queryKey: ['models', clusterName, name],
    queryFn: async () => {
      if (!clusterName) {
        throw new Error('Cluster name is required to fetch model')
      }
      
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/models/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch model')
      }
      return response.json()
    },
    enabled: !!name && !!clusterName,
  })
}

export function useDeleteModel(clusterName: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for model deletion')
      }
      
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/models/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete model')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['models'] })
      queryClient.invalidateQueries({ queryKey: ['models', clusterName] })
    },
  })
}