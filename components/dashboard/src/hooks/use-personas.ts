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
      
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/personas/${name}`)
      if (!response.ok) {
        throw new Error('Failed to fetch persona')
      }
      return response.json()
    },
    enabled: !!name && !!clusterName,
  })
}

export function useUpdatePersona(clusterName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ name, data }: { name: string; data: any }) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for persona update')
      }

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/personas/${name}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update persona')
      }
      return response.json()
    },
    onSuccess: (data, variables) => {
      // Invalidate and refetch persona queries
      queryClient.invalidateQueries({ queryKey: ['personas'] })
      queryClient.invalidateQueries({ queryKey: ['personas', clusterName] })
      queryClient.invalidateQueries({ queryKey: ['personas', clusterName, variables.name] })
    },
    onError: (err) => {
      console.error('Failed to update persona:', err)
    },
  })
}

export function useDeletePersona(clusterName: string) {
  const queryClient = useQueryClient()
  const { activeOrganizationId } = useOrganizationStore()

  return useMutation({
    mutationFn: async (name: string) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for persona deletion')
      }

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/personas/${name}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error('Failed to delete persona')
      }
      return response.json()
    },
    onMutate: async (personaName: string) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['personas'] })

      // Snapshot the previous value
      const previousPersonas = queryClient.getQueryData(['personas'])
      const previousClusterPersonas = queryClient.getQueryData(['personas', activeOrganizationId, clusterName])

      // Optimistically remove from cache
      queryClient.setQueryData(['personas', activeOrganizationId, clusterName], (old: any) => {
        if (!old?.data) return old

        return {
          ...old,
          data: old.data.filter((persona: any) => persona.metadata.name !== personaName),
          total: Math.max(0, old.total - 1)
        }
      })

      return { previousPersonas, previousClusterPersonas }
    },
    onError: (err, personaName, context) => {
      // Rollback on error
      if (context?.previousPersonas) {
        queryClient.setQueryData(['personas'], context.previousPersonas)
      }
      if (context?.previousClusterPersonas) {
        queryClient.setQueryData(['personas', activeOrganizationId, clusterName], context.previousClusterPersonas)
      }
      console.error('Failed to delete persona:', err)
    },
    onSettled: (data, error, personaName) => {
      // Always refetch after mutation
      queryClient.invalidateQueries({ queryKey: ['personas'] })
      queryClient.invalidateQueries({ queryKey: ['personas', clusterName] })
      if (personaName) {
        queryClient.removeQueries({ queryKey: ['personas', clusterName, personaName] })
      }
    },
  })
}

export interface GeneratePersonaParams {
  idea: string
  modelName: string
}

export function useGeneratePersona() {
  return useMutation({
    mutationFn: async (params: GeneratePersonaParams) => {
      const response = await fetchWithOrganization('/api/personas/generate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params),
      })
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to generate persona')
      }
      return response.json()
    },
    onError: (err) => {
      console.error('Failed to generate persona:', err)
    },
  })
}