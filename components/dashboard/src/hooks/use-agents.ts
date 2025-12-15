import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { LanguageAgent, LanguageAgentListParams, LanguageAgentFormData } from '@/types/agent'

export function useAgents(params?: LanguageAgentListParams) {
  return useQuery({
    queryKey: ['agents', params],
    queryFn: async () => {
      const searchParams = new URLSearchParams()
      if (params?.page) searchParams.append('page', params.page.toString())
      if (params?.limit) searchParams.append('limit', params.limit.toString())
      if (params?.search) searchParams.append('search', params.search)
      if (params?.phase?.length) {
        params.phase.forEach(p => searchParams.append('phase', p))
      }
      if (params?.executionMode?.length) {
        params.executionMode.forEach(e => searchParams.append('executionMode', e))
      }
      if (params?.sortBy) searchParams.append('sortBy', params.sortBy)
      if (params?.sortOrder) searchParams.append('sortOrder', params.sortOrder)

      const response = await fetch(`/api/agents?${searchParams}`)
      if (!response.ok) {
        throw new Error('Failed to fetch agents')
      }
      return response.json()
    },
    refetchInterval: 5000,
  })
}

export function useAgent(name: string, clusterName?: string) {
  return useQuery({
    queryKey: ['agents', clusterName, name],
    queryFn: async () => {
      // Use cluster-scoped API if cluster name is provided
      const endpoint = clusterName 
        ? `/api/clusters/${clusterName}/agents/${name}`
        : `/api/agents/${name}`
      
      const response = await fetch(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch agent')
      }
      return response.json()
    },
    enabled: !!name,
  })
}

export function useCreateAgent() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (agent: LanguageAgentFormData) => {
      const response = await fetch('/api/agents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(agent),
      })
      
      if (!response.ok) {
        throw new Error('Failed to create agent')
      }
      
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agents'] })
    },
  })
}

export function useUpdateAgent() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async ({ name, agent }: { name: string; agent: Partial<LanguageAgent> }) => {
      const response = await fetch(`/api/agents/${name}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(agent),
      })
      
      if (!response.ok) {
        throw new Error('Failed to update agent')
      }
      
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agents'] })
    },
  })
}

export function useDeleteAgent(clusterName?: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      // Use cluster-scoped API if cluster name is provided
      const endpoint = clusterName 
        ? `/api/clusters/${clusterName}/agents/${name}`
        : `/api/agents/${name}`
      
      const response = await fetch(endpoint, {
        method: 'DELETE',
      })
      
      if (!response.ok) {
        throw new Error('Failed to delete agent')
      }
      
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agents'] })
    },
  })
}