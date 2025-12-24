'use client'

import { useEffect, useState } from 'react'
import { useConsole } from '@/contexts/console-context'
import { AgentListItem } from './agent-list-item'
import { Loader2, MessageSquare } from 'lucide-react'
import { fetchWithOrganization } from '@/lib/api-client'

interface Conversation {
  id: string
  agentName: string
  clusterName: string
  title: string | null
  updatedAt: string
  messageCount?: number
}

interface AgentListProps {
  searchQuery: string
  refreshTrigger?: number
}

export function AgentList({ searchQuery, refreshTrigger = 0 }: AgentListProps) {
  const { selectedAgent, selectedCluster } = useConsole()
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchConversations = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await fetchWithOrganization('/api/conversations')

        if (!response.ok) {
          throw new Error('Failed to fetch conversations')
        }

        const data = await response.json()
        setConversations(data.conversations || [])
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load conversations')
      } finally {
        setIsLoading(false)
      }
    }

    fetchConversations()
  }, [refreshTrigger])

  const filteredConversations = conversations.filter((conversation) =>
    conversation.agentName.toLowerCase().includes(searchQuery.toLowerCase()) ||
    conversation.title?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-stone-400" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-4 text-center">
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      </div>
    )
  }

  if (filteredConversations.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-4">
        <MessageSquare className="h-12 w-12 text-stone-400 dark:text-stone-500 mb-3" />
        <p className="text-[11px] font-light text-stone-600 dark:text-stone-400 text-center">
          {searchQuery ? 'No conversations match your search' : 'No conversations yet'}
        </p>
        <p className="text-[10px] font-light text-stone-500 dark:text-stone-500 text-center mt-2">
          Connect to an agent to start
        </p>
      </div>
    )
  }

  return (
    <div className="py-2">
      {filteredConversations.map((conversation) => (
        <AgentListItem
          key={conversation.id}
          conversation={conversation}
          isActive={selectedAgent === conversation.agentName && selectedCluster === conversation.clusterName}
        />
      ))}
    </div>
  )
}
