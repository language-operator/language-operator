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
  orphanedConversations: Conversation[]
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

  const { agentOptions, orphanedConversations } = useMemo(() => {
    const availableAgentNames = new Set(agents.map(agent => agent.metadata.name))
    
    // Separate valid and orphaned conversations
    const validConversations = conversations.filter(conv => 
      // Check if the conversation references an existing agent AND the current cluster
      availableAgentNames.has(conv.agentName) && conv.clusterName === clusterName
    )
    
    const orphaned = conversations.filter(conv => 
      // Orphaned if agent doesn't exist OR conversation is from different cluster
      !availableAgentNames.has(conv.agentName) || conv.clusterName !== clusterName
    )

    // Get conversation counts by agent name (only for valid conversations)
    const conversationCounts = validConversations.reduce((acc, conv) => {
      acc[conv.agentName] = (acc[conv.agentName] || 0) + 1
      return acc
    }, {} as Record<string, number>)

    // Create options for agents that have valid conversations
    const options: AgentOption[] = Object.entries(conversationCounts)
      .map(([agentName, count]) => ({
        value: agentName,
        label: `${agentName} (${count})`,
        count: count
      }))
      .sort((a, b) => a.value.localeCompare(b.value))

    // Add "All Agents" option at the beginning
    const agentOptions = [
      {
        value: 'all',
        label: `All Agents (${validConversations.length})`,
        count: validConversations.length
      },
      ...options
    ]

    return { agentOptions, orphanedConversations: orphaned }
  }, [agents, conversations, clusterName])

  return {
    agentOptions,
    isLoading,
    error,
    orphanedConversations
  }
}