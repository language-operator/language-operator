'use client'

import { useState } from 'react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Loader2 } from 'lucide-react'

interface Conversation {
  id: string
  agentName: string
  clusterName: string
  title?: string
}

interface DeleteConversationDialogProps {
  conversation: Conversation | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (conversationId: string) => Promise<void>
}

export function DeleteConversationDialog({
  conversation,
  open,
  onOpenChange,
  onConfirm,
}: DeleteConversationDialogProps) {
  const [isDeleting, setIsDeleting] = useState(false)

  const handleConfirm = async () => {
    if (!conversation) return

    setIsDeleting(true)
    try {
      await onConfirm(conversation.id)
      onOpenChange(false)
    } catch (error) {
      // Error handling is done in the parent component
      console.error('Error in delete dialog:', error)
    } finally {
      setIsDeleting(false)
    }
  }

  const displayTitle = conversation?.title || conversation?.agentName || 'Unknown'

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Conversation</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete the conversation &quot;{displayTitle}&quot;?
            This action cannot be undone and will permanently delete all messages in this conversation.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={isDeleting}
            className="bg-red-600 hover:bg-red-700 dark:bg-red-600 dark:hover:bg-red-700"
          >
            {isDeleting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Deleting...
              </>
            ) : (
              'Delete'
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}