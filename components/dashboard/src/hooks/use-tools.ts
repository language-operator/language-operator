import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganizationStore } from '@/store/organization-store'
import { LanguageTool, LanguageToolListParams, LanguageToolFormData } from '@/types/tool'

export function useTools(params: LanguageToolListParams & { clusterName: string }) {
  const { activeOrganizationId } = useOrganizationStore()
  
  return useQuery({
    queryKey: ['tools', activeOrganizationId, params.clusterName, params],
    queryFn: async () => {
      if (!params.clusterName) {
        throw new Error('Cluster name is required to fetch tools')
      }

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

      const endpoint = `/api/clusters/${params.clusterName}/tools?${searchParams}`

      const response = await fetchWithOrganization(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch tools')
      }
      return response.json()
    },
  })
}

export function useTool(name: string, clusterName: string) {
  return useQuery({
    queryKey: ['tools', clusterName, name],
    queryFn: async () => {
      if (!clusterName) {
        throw new Error('Cluster name is required to fetch tool')
      }
      
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/tools/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch tool')
      }
      return response.json()
    },
    enabled: !!name && !!clusterName,
  })
}

export function useDeleteTool(clusterName: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for tool deletion')
      }
      
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/tools/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete tool')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tools'] })
      queryClient.invalidateQueries({ queryKey: ['tools', clusterName] })
    },
  })
}