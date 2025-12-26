'use client'

import { useEffect, useState } from 'react'
import { useConsole } from '@/contexts/console-context'
import { Button } from '@/components/ui/button'
import { ChevronRight, MessageSquare, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { fetchWithOrganization } from '@/lib/api-client'

interface Conversation {
  id: string
  agentName: string
  clusterName: string
  title?: string
  createdAt: string
  updatedAt: string
  messageCount?: number
}

export function ConversationSidebarCollapsed() {
  const { 
    toggleConversationSidebar, 
    selectedAgent,
    selectedCluster,
    conversationDbId,
    loadConversation,
    conversationListRefreshTrigger
  } = useConsole()
  
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchConversations = async () => {
      setIsLoading(true)

      try {
        const response = await fetchWithOrganization('/api/conversations')

        if (!response.ok) {
          throw new Error('Failed to fetch conversations')
        }

        const data = await response.json()
        setConversations(data.conversations || [])
      } catch (err) {
        console.error('Error fetching conversations:', err)
        setConversations([])
      } finally {
        setIsLoading(false)
      }
    }

    fetchConversations()
  }, [conversationListRefreshTrigger])

  const handleConversationClick = (conversation: Conversation) => {
    // Load this conversation
    loadConversation(conversation.id, conversation.agentName, conversation.clusterName)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header with conversations icon */}
      <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-3 px-2 h-[52px] flex items-center justify-center">
        <Button
          variant="ghost"
          size="sm"
          onClick={toggleConversationSidebar}
          className="h-6 w-6 p-0 hover:bg-stone-300/50 dark:hover:bg-stone-700/50 text-stone-600 dark:text-stone-400"
          title="Expand conversations"
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>

      {/* Conversation icons */}
      <div className="flex-1 overflow-y-auto py-2">
        {isLoading ? (
          <div className="flex justify-center">
            <div className="h-10 w-10 rounded-lg bg-stone-300/30 dark:bg-stone-700/30 flex items-center justify-center">
              <Loader2 className="h-4 w-4 animate-spin text-stone-500 dark:text-stone-400" />
            </div>
          </div>
        ) : conversations.length > 0 ? (
          conversations.map((conversation) => {
            const isActive = conversationDbId === conversation.id
            
            return (
              <div key={conversation.id} className="flex justify-center mb-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleConversationClick(conversation)}
                  className={cn(
                    "h-10 w-10 p-0 rounded-lg relative flex items-center justify-center border-l-2",
                    isActive
                      ? "bg-stone-100 dark:bg-stone-800/70 border-l-amber-400"
                      : "border-l-transparent hover:bg-stone-300/30 dark:hover:bg-stone-700/30"
                  )}
                  title={`${conversation.agentName} conversation (${conversation.clusterName})`}
                >
                  <MessageSquare className="h-5 w-5 text-stone-700 dark:text-stone-300" />
                </Button>
              </div>
            )
          })
        ) : (
          <div className="flex justify-center">
            <div className="h-10 w-10 rounded-lg bg-stone-300/30 dark:bg-stone-700/30 flex items-center justify-center">
              <MessageSquare className="h-4 w-4 text-stone-500 dark:text-stone-400" />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}