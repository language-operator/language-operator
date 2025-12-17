import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { LanguageAgent, LanguageAgentListParams, LanguageAgentFormData } from '@/types/agent'

export function useAgents(params?: LanguageAgentListParams & { clusterName?: string }) {
  return useQuery({
    queryKey: ['agents', params?.clusterName, params],
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

      // Use cluster-scoped API if cluster name is provided
      const endpoint = params?.clusterName 
        ? `/api/clusters/${params.clusterName}/agents?${searchParams}`
        : `/api/agents?${searchParams}` // Legacy fallback for non-cluster contexts

      const response = await fetch(endpoint)
      if (!response.ok) {
        throw new Error('Failed to fetch agents')
      }
      return response.json()
    },
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
        // Parse error response for better error handling
        let errorMessage = 'Failed to fetch agent'
        let errorData: any = null
        
        try {
          errorData = await response.json()
          errorMessage = errorData.error || errorData.details || errorMessage
        } catch {
          // If JSON parsing fails, use status-based message
          if (response.status === 404) {
            errorMessage = `Agent "${name}" not found${clusterName ? ` in cluster "${clusterName}"` : ''}`
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
    enabled: !!name,
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

export function useCreateAgent(clusterName?: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (agent: LanguageAgentFormData) => {
      if (!clusterName) {
        throw new Error('Cluster name is required for agent creation')
      }

      const response = await fetch(`/api/clusters/${clusterName}/agents`, {
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
      // Invalidate both general and cluster-specific queries
      queryClient.invalidateQueries({ queryKey: ['agents'] })
      if (clusterName) {
        queryClient.invalidateQueries({ queryKey: ['agents', clusterName] })
      }
    },
  })
}

export function useUpdateAgent(clusterName?: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async ({ name, agent }: { name: string; agent: Partial<LanguageAgent> }) => {
      // Use cluster-scoped API if cluster name is provided
      const endpoint = clusterName 
        ? `/api/clusters/${clusterName}/agents/${name}`
        : `/api/agents/${name}`
      
      const response = await fetch(endpoint, {
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
      // Invalidate both general and cluster-specific queries
      queryClient.invalidateQueries({ queryKey: ['agents'] })
      if (clusterName) {
        queryClient.invalidateQueries({ queryKey: ['agents', clusterName] })
      }
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
    onMutate: async (agentName: string) => {
      // Cancel any outgoing refetches  
      await queryClient.cancelQueries({ queryKey: ['agents'] })
      
      // Snapshot the previous value
      const previousAgents = queryClient.getQueryData(['agents'])
      const previousClusterAgents = clusterName ? 
        queryClient.getQueryData(['agents', clusterName]) : null
      
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
      if (clusterName) {
        queryClient.setQueryData(['agents', clusterName], (old: any) => {
          if (!old?.data) return old
          
          return {
            ...old,
            data: old.data.filter((agent: any) => agent.metadata.name !== agentName),
            total: Math.max(0, old.total - 1)
          }
        })
      }
      
      return { previousAgents, previousClusterAgents }
    },
    onError: (err, agentName, context) => {
      // Rollback on error
      if (context?.previousAgents) {
        queryClient.setQueryData(['agents'], context.previousAgents)
      }
      if (context?.previousClusterAgents && clusterName) {
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
      if (clusterName) {
        queryClient.invalidateQueries({ queryKey: ['agents', clusterName] })
      }
    },
  })
}