'use client'

import { useState } from 'react'
import { useConsole } from '@/contexts/console-context'
import { cn } from '@/lib/utils'
import { MessageSquare, MoreVertical, ExternalLink, Trash2, Edit3, AlertTriangle } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { DeleteConversationDialog } from './delete-conversation-dialog'
import { RenameConversationDialog } from './rename-conversation-dialog'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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
  isOrphaned?: boolean
}

export function AgentListItem({ conversation, isOrphaned = false }: AgentListItemProps) {
  const { selectedAgent, selectedCluster, conversationDbId, loadConversation, deleteConversation, refreshConversationList } = useConsole()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [showRenameDialog, setShowRenameDialog] = useState(false)
  
  const isActive = conversationDbId === conversation.id

  const handleConversationClick = () => {
    if (isOrphaned) {
      toast.error(`Cannot load conversation: Agent "${conversation.agentName}" or cluster "${conversation.clusterName}" no longer exists`)
      return
    }
    loadConversation(conversation.id, conversation.agentName, conversation.clusterName)
  }

  const handleOpenClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    handleConversationClick()
  }

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    setShowDeleteDialog(true)
  }

  const handleRenameClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    setShowRenameDialog(true)
  }

  const handleConfirmDelete = async (conversationId: string) => {
    try {
      await deleteConversation(conversationId)
      toast.success('Conversation deleted successfully')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to delete conversation')
    }
  }

  const handleConfirmRename = async (conversationId: string, newTitle: string) => {
    // Refresh the conversation list to show the updated title
    refreshConversationList()
  }

  const displayTitle = conversation.title || conversation.agentName

  return (
    <>
      <div
        className={cn(
          'w-full px-4 py-3 text-left transition-colors border-l-2 border-b border-b-stone-200 dark:border-b-stone-700 relative group',
          // Orphaned state styling
          isOrphaned 
            ? 'bg-amber-50/50 dark:bg-amber-900/10 border-l-amber-500/50 cursor-not-allowed opacity-75'
            // Active state styling
            : isActive
            ? 'bg-stone-100 border-stone-900 dark:bg-stone-800/70 dark:border-l-amber-400 cursor-pointer'
            // Normal state styling
            : 'border-l-transparent hover:bg-stone-50 dark:hover:bg-stone-800/30 cursor-pointer'
        )}
        onClick={handleConversationClick}
      >
        <div className="flex items-start gap-3">
          <div className="mt-1">
            {isOrphaned ? (
              <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
            ) : (
              <MessageSquare className="h-4 w-4 text-stone-600 dark:text-stone-400" />
            )}
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between gap-2 mb-1">
              <h3 className="text-sm font-light text-stone-900 dark:text-stone-300 truncate">
                {displayTitle}
              </h3>
              {/* Dropdown menu */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 w-6 p-0 text-stone-600 dark:text-stone-400 hover:bg-stone-200 dark:hover:bg-stone-700/50"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <MoreVertical className="h-3 w-3" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                  {!isOrphaned && (
                    <>
                      <DropdownMenuItem onClick={handleOpenClick}>
                        <ExternalLink className="mr-2 h-4 w-4" />
                        Open Conversation
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={handleRenameClick}>
                        <Edit3 className="mr-2 h-4 w-4" />
                        Rename Conversation
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                    </>
                  )}
                  <DropdownMenuItem onClick={handleDeleteClick} variant="destructive">
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete Conversation
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
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

      <RenameConversationDialog
        conversation={conversation}
        open={showRenameDialog}
        onOpenChange={setShowRenameDialog}
        onConfirm={handleConfirmRename}
      />
    </>
  )
}
