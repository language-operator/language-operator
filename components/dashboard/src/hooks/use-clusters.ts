import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganizationStore } from '@/store/organization-store'
import { LanguageCluster, LanguageClusterListParams, LanguageClusterFormData } from '@/types/cluster'

export function useClusters(params?: LanguageClusterListParams) {
  const { activeOrganizationId } = useOrganizationStore()
  
  return useQuery({
    queryKey: ['clusters', activeOrganizationId, params],
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

      const response = await fetchWithOrganization(`/api/clusters?${searchParams}`)
      if (!response.ok) {
        throw new Error('Failed to fetch clusters')
      }
      return response.json()
    },
  })
}

export function useCluster(name: string) {
  const { activeOrganizationId } = useOrganizationStore()
  
  return useQuery({
    queryKey: ['clusters', activeOrganizationId, name],
    queryFn: async () => {
      const response = await fetchWithOrganization(`/api/clusters/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch cluster')
      }
      const data = await response.json()
      return data.cluster // Extract cluster from the response wrapper
    },
    enabled: !!name,
  })
}

export function useDeleteCluster() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      const response = await fetchWithOrganization(`/api/clusters/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete cluster')
      }
      return response.json()
    },
    onMutate: async (clusterName: string) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['clusters'] })
      
      // Snapshot the previous value
      const previousClusters = queryClient.getQueryData(['clusters'])
      
      // Optimistically update the cache by removing the cluster
      queryClient.setQueryData(['clusters'], (old: any) => {
        if (!old?.data) return old
        
        return {
          ...old,
          data: old.data.filter((cluster: any) => cluster.metadata.name !== clusterName),
          total: Math.max(0, old.total - 1)
        }
      })
      
      // Return context object with snapshot
      return { previousClusters }
    },
    onError: (err, clusterName, context) => {
      // If mutation fails, rollback to previous state
      if (context?.previousClusters) {
        queryClient.setQueryData(['clusters'], context.previousClusters)
      }
      console.error('Failed to delete cluster:', err)
    },
    onSuccess: (data, clusterName) => {
      // Update any individual cluster queries
      queryClient.removeQueries({ queryKey: ['clusters', clusterName] })
      console.log(`✅ Cluster ${clusterName} deleted successfully`)
    },
    onSettled: () => {
      // Always refetch after mutation, regardless of success or failure
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
    },
  })
}