import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganizationStore } from '@/store/organization-store'
import { LanguageAgent, LanguageAgentListParams, LanguageAgentFormData } from '@/types/agent'

export function useAgents(params: LanguageAgentListParams & { clusterName: string }) {
  const { activeOrganizationId } = useOrganizationStore()
  
  return useQuery({
    queryKey: ['agents', activeOrganizationId, params.clusterName, params],
    queryFn: async () => {
      if (!params.clusterName) {
        throw new Error('Cluster name is required to fetch agents')
      }

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

      const endpoint = `/api/clusters/${params.clusterName}/agents?${searchParams}`

      const response = await fetchWithOrganization(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch agents')
      }
      return response.json()
    },
  })
}

export function useAgent(name: string, clusterName: string) {
  return useQuery({
    queryKey: ['agents', clusterName, name],
    queryFn: async () => {
      if (!clusterName) {
        throw new Error('Cluster name is required to fetch agent')
      }
      
      const endpoint = `/api/clusters/${clusterName}/agents/${name}`
      
      const response = await fetchWithOrganization(endpoint)
      if (!response.ok) {
        // Parse error response for better error handling
        let errorMessage = 'Failed to fetch agent'
        let errorData: any = null
        
        try {
          errorData = await response.json()
          errorMessage = errorData.error || errorData.details || errorMessage
        } catch {
          // If JSON parsing fails, use status-based message
          if (response.status === 404) {
            errorMessage = `Agent "${name}" not found in cluster "${clusterName}"`
          }
        }
        
        // Create error object with additional context
        const error = new Error(errorMessage) as any
        error.status = response.status
        error.data = errorData
        throw error
      }
      return response.json()
    },
    enabled: !!name && !!clusterName,
    retry: (failureCount, error: any) => {
      // Don't retry on 404 or other client errors
      if (error?.status >= 400 && error?.status < 500) {
        return false
      }
      // Use default retry logic for server errors
      return failureCount < 3
    },
  })
}

export function useCreateAgent(clusterName: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (agent: LanguageAgentFormData) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for agent creation')
      }

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/agents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(agent),
      })
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }))
        throw new Error(errorData.error || 'Failed to create agent')
      }
      
      return response.json()
    },
    onSuccess: () => {
      // Invalidate cluster-specific queries
      queryClient.invalidateQueries({ queryKey: ['agents'] })
      queryClient.invalidateQueries({ queryKey: ['agents', clusterName] })
    },
  })
}

export function useUpdateAgent(clusterName: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async ({ name, agent }: { name: string; agent: Partial<LanguageAgent> }) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for agent update')
      }
      
      const endpoint = `/api/clusters/${clusterName}/agents/${name}`
      
      const response = await fetchWithOrganization(endpoint, {
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
      // Invalidate cluster-specific queries
      queryClient.invalidateQueries({ queryKey: ['agents'] })
      queryClient.invalidateQueries({ queryKey: ['agents', clusterName] })
    },
  })
}

export function useDeleteAgent(clusterName: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for agent deletion')
      }
      
      const endpoint = `/api/clusters/${clusterName}/agents/${name}`
      
      const response = await fetchWithOrganization(endpoint, {
        method: 'DELETE',
      })
      
      if (!response.ok) {
        throw new Error('Failed to delete agent')
      }
      
      return response.json()
    },
    onMutate: async (agentName: string) => {
      // Cancel any outgoing refetches  
      await queryClient.cancelQueries({ queryKey: ['agents'] })
      
      // Snapshot the previous value
      const previousAgents = queryClient.getQueryData(['agents'])
      const previousClusterAgents = queryClient.getQueryData(['agents', clusterName])
      
      // Optimistically update the general agents cache
      queryClient.setQueryData(['agents'], (old: any) => {
        if (!old?.data) return old
        
        return {
          ...old,
          data: old.data.filter((agent: any) => agent.metadata.name !== agentName),
          total: Math.max(0, old.total - 1)
        }
      })
      
      // Optimistically update cluster-specific agents cache
      queryClient.setQueryData(['agents', clusterName], (old: any) => {
        if (!old?.data) return old
        
        return {
          ...old,
          data: old.data.filter((agent: any) => agent.metadata.name !== agentName),
          total: Math.max(0, old.total - 1)
        }
      })
      
      return { previousAgents, previousClusterAgents }
    },
    onError: (err, agentName, context) => {
      // Rollback on error
      if (context?.previousAgents) {
        queryClient.setQueryData(['agents'], context.previousAgents)
      }
      if (context?.previousClusterAgents) {
        queryClient.setQueryData(['agents', clusterName], context.previousClusterAgents)
      }
      console.error('Failed to delete agent:', err)
    },
    onSuccess: (data, agentName) => {
      // Remove individual agent queries
      queryClient.removeQueries({ queryKey: ['agents', clusterName, agentName] })
      queryClient.removeQueries({ queryKey: ['agent', agentName] })
      console.log(`✅ Agent ${agentName} deleted successfully`)
    },
    onSettled: () => {
      // Always refetch after mutation
      queryClient.invalidateQueries({ queryKey: ['agents'] })
      queryClient.invalidateQueries({ queryKey: ['agents', clusterName] })
    },
  })
}