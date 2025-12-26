'use client'

import { useState } from 'react'
import { useConsole } from '@/contexts/console-context'
import { cn } from '@/lib/utils'
import { MessageSquare, Trash2 } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { DeleteConversationDialog } from './delete-conversation-dialog'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'

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
  const { selectedAgent, selectedCluster, loadConversation, deleteConversation } = useConsole()
  const [isHovered, setIsHovered] = useState(false)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  
  const isActive =
    selectedAgent === conversation.agentName &&
    selectedCluster === conversation.clusterName

  const handleClick = () => {
    loadConversation(conversation.id, conversation.agentName, conversation.clusterName)
  }

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation() // Prevent triggering conversation selection
    setShowDeleteDialog(true)
  }

  const handleConfirmDelete = async (conversationId: string) => {
    try {
      await deleteConversation(conversationId)
      toast.success('Conversation deleted successfully')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to delete conversation')
    }
  }

  const displayTitle = conversation.title || conversation.agentName

  return (
    <>
      <div
        className={cn(
          'w-full px-4 py-3 text-left transition-colors border-l-2 relative group cursor-pointer',
          isActive
            ? 'bg-stone-200 border-stone-900 dark:bg-stone-800 dark:border-amber-400'
            : 'border-transparent hover:bg-stone-100/50 dark:hover:bg-stone-800/50'
        )}
        onClick={handleClick}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
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
              {/* Delete button - shown on hover */}
              {isHovered && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleDeleteClick}
                  className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-100 dark:hover:bg-red-900/20 text-red-600 dark:text-red-400"
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              )}
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
      </div>

      <DeleteConversationDialog
        conversation={conversation}
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleConfirmDelete}
      />
    </>
  )
}
