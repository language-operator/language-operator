import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { LanguageAgent } from '@/lib/kubernetes'

export function useAgents() {
  return useQuery({
    queryKey: ['agents'],
    queryFn: async () => {
      const response = await fetch('/api/agents')
      if (!response.ok) {
        throw new Error('Failed to fetch agents')
      }
      const data = await response.json()
      return data.agents as LanguageAgent[]
    },
    refetchInterval: 5000, // Refetch every 5 seconds for real-time updates
  })
}

export function useCreateAgent() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (agent: Partial<LanguageAgent>) => {
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

export function useDeleteAgent() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (name: string) => {
      const response = await fetch(`/api/agents/${name}`, {
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