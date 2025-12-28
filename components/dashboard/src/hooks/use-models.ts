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

export function useUpdateModel(clusterName: string) {
  const queryClient = useQueryClient()
  const { activeOrganizationId } = useOrganizationStore()

  return useMutation({
    mutationFn: async ({ modelName, updateData }: { modelName: string; updateData: any }) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for model update')
      }

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/models/${modelName}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateData),
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update model')
      }
      return response.json()
    },
    onSuccess: (data, variables) => {
      // Invalidate and refetch the individual model query
      queryClient.invalidateQueries({ 
        queryKey: ['models', clusterName, variables.modelName] 
      })
      
      // Also invalidate the models list to update any summary data
      queryClient.invalidateQueries({ 
        queryKey: ['models', activeOrganizationId, clusterName] 
      })
      queryClient.invalidateQueries({ 
        queryKey: ['models'] 
      })
    },
    onError: (error) => {
      console.error('Failed to update model:', error)
    }
  })
}

export function useDeleteModel(clusterName: string) {
  const queryClient = useQueryClient()
  const { activeOrganizationId } = useOrganizationStore()

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
    onMutate: async (modelName: string) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['models'] })

      // Snapshot the previous value
      const previousModels = queryClient.getQueryData(['models'])
      const previousClusterModels = queryClient.getQueryData(['models', activeOrganizationId, clusterName])

      // Optimistically remove from cache
      queryClient.setQueryData(['models', activeOrganizationId, clusterName], (old: any) => {
        if (!old?.data) return old

        return {
          ...old,
          data: old.data.filter((model: any) => model.metadata.name !== modelName),
          total: Math.max(0, old.total - 1)
        }
      })

      return { previousModels, previousClusterModels }
    },
    onError: (err, modelName, context) => {
      // Rollback on error
      if (context?.previousModels) {
        queryClient.setQueryData(['models'], context.previousModels)
      }
      if (context?.previousClusterModels) {
        queryClient.setQueryData(['models', activeOrganizationId, clusterName], context.previousClusterModels)
      }
      console.error('Failed to delete model:', err)
    },
    onSettled: (data, error, modelName) => {
      // Always refetch after mutation
      queryClient.invalidateQueries({ queryKey: ['models'] })
      queryClient.invalidateQueries({ queryKey: ['models', clusterName] })
      if (modelName) {
        queryClient.removeQueries({ queryKey: ['models', clusterName, modelName] })
      }
    },
  })
}