'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  LanguageAgent, 
  LanguageAgentListParams, 
  LanguageAgentListResponse,
  LanguageAgentResponse,
  LanguageAgentFormData 
} from '@/types/agent'

// API client functions
async function fetchAgents(params: LanguageAgentListParams = {}): Promise<LanguageAgentListResponse> {
  const searchParams = new URLSearchParams()
  
  if (params.page) searchParams.append('page', params.page.toString())
  if (params.limit) searchParams.append('limit', params.limit.toString())
  if (params.sortBy) searchParams.append('sortBy', params.sortBy)
  if (params.sortOrder) searchParams.append('sortOrder', params.sortOrder)
  if (params.search) searchParams.append('search', params.search)
  
  if (params.phase) {
    params.phase.forEach(phase => searchParams.append('phase', phase))
  }
  
  if (params.executionMode) {
    params.executionMode.forEach(mode => searchParams.append('executionMode', mode))
  }

  const response = await fetch(`/api/agents?${searchParams.toString()}`)
  
  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || 'Failed to fetch agents')
  }
  
  return response.json()
}

async function fetchAgent(name: string): Promise<LanguageAgentResponse> {
  const response = await fetch(`/api/agents/${name}`)
  
  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || 'Failed to fetch agent')
  }
  
  return response.json()
}

async function createAgent(formData: LanguageAgentFormData): Promise<LanguageAgentResponse> {
  const response = await fetch('/api/agents', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(formData),
  })
  
  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || 'Failed to create agent')
  }
  
  return response.json()
}

async function updateAgent(name: string, agent: Partial<LanguageAgent>): Promise<LanguageAgentResponse> {
  const response = await fetch(`/api/agents/${name}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(agent),
  })
  
  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || 'Failed to update agent')
  }
  
  return response.json()
}

async function deleteAgent(name: string): Promise<{ success: boolean }> {
  const response = await fetch(`/api/agents/${name}`, {
    method: 'DELETE',
  })
  
  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || 'Failed to delete agent')
  }
  
  return response.json()
}

// React Query hooks

export function useAgents(params: LanguageAgentListParams = {}) {
  return useQuery({
    queryKey: ['agents', params],
    queryFn: () => fetchAgents(params),
    staleTime: 5 * 60 * 1000, // 5 minutes - use real-time updates instead
  })
}

export function useAgent(name: string, enabled: boolean = true) {
  return useQuery({
    queryKey: ['agent', name],
    queryFn: () => fetchAgent(name),
    enabled: enabled && !!name,
    staleTime: 5 * 60 * 1000, // 5 minutes - use real-time updates instead
  })
}

export function useCreateAgent() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: createAgent,
    onSuccess: () => {
      // Invalidate and refetch agents list
      queryClient.invalidateQueries({ queryKey: ['agents'] })
    },
  })
}

export function useUpdateAgent() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ name, agent }: { name: string; agent: Partial<LanguageAgent> }) => 
      updateAgent(name, agent),
    onSuccess: (data, variables) => {
      // Invalidate and refetch the specific agent and the agents list
      queryClient.invalidateQueries({ queryKey: ['agent', variables.name] })
      queryClient.invalidateQueries({ queryKey: ['agents'] })
    },
  })
}

export function useDeleteAgent() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: deleteAgent,
    onSuccess: (data, deletedName) => {
      // Remove the specific agent from cache and refetch agents list
      queryClient.removeQueries({ queryKey: ['agent', deletedName] })
      queryClient.invalidateQueries({ queryKey: ['agents'] })
    },
  })
}

// Utility hooks for common operations

export function useAgentStats() {
  const { data } = useAgents()
  
  if (!data?.data) {
    return {
      total: 0,
      running: 0,
      pending: 0,
      failed: 0,
      succeeded: 0,
    }
  }
  
  const agents = data.data
  
  return {
    total: agents.length,
    running: agents.filter(a => a.status?.phase === 'Running').length,
    pending: agents.filter(a => a.status?.phase === 'Pending').length,
    failed: agents.filter(a => a.status?.phase === 'Failed').length,
    succeeded: agents.filter(a => a.status?.phase === 'Succeeded').length,
  }
}

export function useAgentSearch(searchTerm: string) {
  return useAgents({
    search: searchTerm,
    limit: 50,
  })
}

export function useAgentsByPhase(phases: string[]) {
  return useAgents({
    phase: phases,
    limit: 100,
  })
}

export function useAgentsByExecutionMode(executionModes: string[]) {
  return useAgents({
    executionMode: executionModes,
    limit: 100,
  })
}