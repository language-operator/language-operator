'use client'

import { useEffect, useState } from 'react'
import { useConsole } from '@/contexts/console-context'
import { AgentListItem } from './agent-list-item'
import { Loader2, MessageSquare } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { fetchWithOrganization } from '@/lib/api-client'
import { useAgentFilter } from '@/hooks/use-agent-filter'

interface Conversation {
  id: string
  agentName: string
  clusterName: string
  title?: string
  createdAt: string
  updatedAt: string
  messageCount?: number
}

interface AgentListProps {
  selectedAgentFilter: string
  onAgentFilterChange: (value: string) => void
  refreshTrigger?: number
}

export function AgentList({ selectedAgentFilter, onAgentFilterChange, refreshTrigger = 0 }: AgentListProps) {
  const { selectedAgent, selectedCluster } = useConsole()
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  // Get agent filter options
  const { agentOptions, isLoading: isLoadingAgents } = useAgentFilter(selectedCluster, conversations)

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
    selectedAgentFilter === 'all' || conversation.agentName === selectedAgentFilter
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

  return (
    <div className="flex flex-col h-full">
      {/* Agent Filter Dropdown */}
      <div className="p-4 border-b border-stone-800/80 dark:border-stone-600/80">
        <Select value={selectedAgentFilter} onValueChange={onAgentFilterChange}>
          <SelectTrigger className="w-full text-sm">
            <SelectValue placeholder="Filter by agent..." />
          </SelectTrigger>
          <SelectContent>
            {agentOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Conversation List */}
      <div className="flex-1 overflow-y-auto">
        {filteredConversations.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 px-4">
            <MessageSquare className="h-12 w-12 text-stone-400 dark:text-stone-500 mb-3" />
            <p className="text-[11px] font-light text-stone-600 dark:text-stone-400 text-center">
              {selectedAgentFilter === 'all' ? 'No conversations yet' : `No conversations for ${selectedAgentFilter}`}
            </p>
            <p className="text-[10px] font-light text-stone-500 dark:text-stone-500 text-center mt-2">
              Connect to an agent to start
            </p>
          </div>
        ) : (
          <div className="py-2">
            {filteredConversations.map((conversation, index) => (
              <AgentListItem
                key={conversation.id}
                conversation={conversation}
                isLast={index === filteredConversations.length - 1}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
