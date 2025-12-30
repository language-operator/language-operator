'use client'

import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { fetchWithOrganization } from '@/lib/api-client'
import { toast } from 'sonner'

interface Conversation {
  id: string
  agentName: string
  clusterName: string
  title?: string
}

interface RenameConversationDialogProps {
  conversation: Conversation | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (conversationId: string, newTitle: string) => void
}

export function RenameConversationDialog({
  conversation,
  open,
  onOpenChange,
  onConfirm,
}: RenameConversationDialogProps) {
  const [newTitle, setNewTitle] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  // Update the input when conversation changes
  useEffect(() => {
    if (conversation) {
      setNewTitle(conversation.title || conversation.agentName)
    }
  }, [conversation])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!conversation || !newTitle.trim()) return

    setIsLoading(true)

    try {
      const response = await fetchWithOrganization(`/api/conversations/${conversation.id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          title: newTitle.trim(),
        }),
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to rename conversation')
      }

      onConfirm(conversation.id, newTitle.trim())
      onOpenChange(false)
      toast.success('Conversation renamed successfully')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to rename conversation')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    setNewTitle(conversation?.title || conversation?.agentName || '')
    onOpenChange(false)
  }

  if (!conversation) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Rename Conversation</DialogTitle>
          </DialogHeader>
          
          <div className="py-4">
            <Label htmlFor="conversation-title" className="text-sm font-medium">
              Title
            </Label>
            <Input
              id="conversation-title"
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="Enter title..."
              className="mt-2"
              disabled={isLoading}
              autoFocus
            />
            <p className="text-xs text-muted-foreground mt-2">
              This will help you identify the conversation in your list.
            </p>
          </div>

          <DialogFooter className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={handleCancel}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isLoading || !newTitle.trim()}
            >
              {isLoading ? 'Renaming...' : 'Rename'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}