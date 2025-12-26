'use client'

import { useEffect, useState, useMemo } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'

interface Agent {
  metadata: {
    name: string
    namespace: string
  }
  spec: {
    clusterRef: string
  }
}

interface Conversation {
  id: string
  agentName: string
  clusterName: string
  title?: string
  createdAt: string
  updatedAt: string
  messageCount?: number
}

interface AgentOption {
  value: string
  label: string
  count: number
}

interface UseAgentFilterResult {
  agentOptions: AgentOption[]
  isLoading: boolean
  error: string | null
}

export function useAgentFilter(clusterName: string | null, conversations: Conversation[]): UseAgentFilterResult {
  const [agents, setAgents] = useState<Agent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchAgents = async () => {
      if (!clusterName) {
        setIsLoading(false)
        return
      }

      setIsLoading(true)
      setError(null)

      try {
        const response = await fetchWithOrganization(`/api/clusters/${clusterName}/agents`)

        if (!response.ok) {
          throw new Error('Failed to fetch agents')
        }

        const data = await response.json()
        setAgents(data.agents || [])
      } catch (err) {
        console.error('Error fetching agents:', err)
        setError(err instanceof Error ? err.message : 'Failed to load agents')
        setAgents([])
      } finally {
        setIsLoading(false)
      }
    }

    fetchAgents()
  }, [clusterName])

  const agentOptions = useMemo(() => {
    // Get conversation counts by agent name
    const conversationCounts = conversations.reduce((acc, conv) => {
      acc[conv.agentName] = (acc[conv.agentName] || 0) + 1
      return acc
    }, {} as Record<string, number>)

    // Get unique agent names from conversations and available agents
    const agentNamesFromConversations = Object.keys(conversationCounts)
    const agentNamesFromAgents = agents.map(agent => agent.metadata.name)
    
    // Combine and deduplicate agent names
    const allAgentNames = [...new Set([...agentNamesFromConversations, ...agentNamesFromAgents])]
    
    // Create options for agents that have conversations
    const options: AgentOption[] = allAgentNames
      .filter(agentName => conversationCounts[agentName] > 0) // Only include agents with conversations
      .map(agentName => ({
        value: agentName,
        label: `${agentName} (${conversationCounts[agentName]})`,
        count: conversationCounts[agentName]
      }))
      .sort((a, b) => a.value.localeCompare(b.value))

    // Add "All Agents" option at the beginning
    return [
      {
        value: 'all',
        label: `All Agents (${conversations.length})`,
        count: conversations.length
      },
      ...options
    ]
  }, [agents, conversations])

  return {
    agentOptions,
    isLoading,
    error
  }
}