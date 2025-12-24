'use client'

import { useConsole } from '@/contexts/console-context'
import { cn } from '@/lib/utils'
import { MessageSquare } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

interface Conversation {
  id: string
  agentName: string
  clusterName: string
  title?: string
  createdAt: string
  updatedAt: string
  messageCount?: number
}

interface AgentListItemProps {
  conversation: Conversation
}

export function AgentListItem({ conversation }: AgentListItemProps) {
  const { selectedAgent, selectedCluster, loadConversation } = useConsole()
  const isActive =
    selectedAgent === conversation.agentName &&
    selectedCluster === conversation.clusterName

  const handleClick = () => {
    loadConversation(conversation.id, conversation.agentName, conversation.clusterName)
  }

  const displayTitle = conversation.title || conversation.agentName

  return (
    <button
      onClick={handleClick}
      className={cn(
        'w-full px-4 py-3 text-left transition-colors border-l-2',
        isActive
          ? 'bg-stone-200 border-stone-900 dark:bg-stone-800 dark:border-amber-400'
          : 'border-transparent hover:bg-stone-100/50 dark:hover:bg-stone-800/50'
      )}
    >
      <div className="flex items-start gap-3">
        <div className="mt-1">
          <MessageSquare className="h-4 w-4 text-stone-600 dark:text-stone-400" />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2 mb-1">
            <h3 className="text-sm font-light text-stone-900 dark:text-stone-300 truncate">
              {displayTitle}
            </h3>
          </div>

          <div className="flex items-center gap-2 text-[10px] text-stone-500 dark:text-stone-400">
            <span className="tracking-wider uppercase">{conversation.agentName}</span>
            <span className="text-stone-400">•</span>
            <span className="tracking-wider uppercase">{conversation.clusterName}</span>
          </div>

          <div className="text-[10px] text-stone-500 dark:text-stone-400 mt-1">
            {formatDistanceToNow(new Date(conversation.updatedAt), {
              addSuffix: true,
            })}
          </div>
        </div>
      </div>
    </button>
  )
}
