import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganizationStore } from '@/store/organization-store'
import { LanguagePersona, LanguagePersonaListParams, LanguagePersonaFormData } from '@/types/persona'

export function usePersonas(params: LanguagePersonaListParams & { clusterName: string }) {
  const { activeOrganizationId } = useOrganizationStore()
  
  return useQuery({
    queryKey: ['personas', activeOrganizationId, params.clusterName, params],
    queryFn: async () => {
      if (!params.clusterName) {
        throw new Error('Cluster name is required to fetch personas')
      }

      const searchParams = new URLSearchParams()
      if (params?.page) searchParams.append('page', params.page.toString())
      if (params?.limit) searchParams.append('limit', params.limit.toString())
      if (params?.search) searchParams.append('search', params.search)
      if (params?.tone?.length) {
        params.tone.forEach(t => searchParams.append('tone', t))
      }
      if (params?.sortBy) searchParams.append('sortBy', params.sortBy)
      if (params?.sortOrder) searchParams.append('sortOrder', params.sortOrder)

      const endpoint = `/api/clusters/${params.clusterName}/personas?${searchParams}`

      const response = await fetchWithOrganization(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch personas')
      }
      return response.json()
    },
  })
}

export function usePersona(name: string, clusterName: string) {
  return useQuery({
    queryKey: ['personas', clusterName, name],
    queryFn: async () => {
      if (!clusterName) {
        throw new Error('Cluster name is required to fetch persona')
      }
      
      const response = await fetch(`/api/clusters/${clusterName}/personas/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch persona')
      }
      return response.json()
    },
    enabled: !!name && !!clusterName,
  })
}

export function useDeletePersona(clusterName: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for persona deletion')
      }
      
      const response = await fetch(`/api/clusters/${clusterName}/personas/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete persona')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['personas'] })
      queryClient.invalidateQueries({ queryKey: ['personas', clusterName] })
    },
  })
}