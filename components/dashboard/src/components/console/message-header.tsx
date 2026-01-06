'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useConsole } from '@/contexts/console-context'
import { DeleteConversationDialog } from './delete-conversation-dialog'
import { useToast } from '@/hooks/use-toast'
import { useOrganization } from '@/components/organization-provider'

export function MessageHeader() {
  const { 
    selectedAgent, 
    selectedCluster, 
    conversationDbId, 
    deleteConversation, 
    refreshConversationList,
    startNewConversation
  } = useConsole()
  const { getOrgUrl } = useOrganization()
  const router = useRouter()
  const { toast } = useToast()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  if (!selectedAgent) return null

  const handleNewConversation = () => {
    if (!selectedAgent || !selectedCluster) return
    
    // Start a completely fresh conversation
    startNewConversation(selectedAgent, selectedCluster)
    
    toast({
      title: 'New conversation started',
      description: `Started a new conversation with ${selectedAgent}`,
    })
  }

  const handleDeleteConversation = async () => {
    if (!conversationDbId) return
    
    try {
      await deleteConversation(conversationDbId)
      refreshConversationList()
      
      toast({
        title: 'Conversation deleted',
        description: 'The conversation has been permanently deleted.',
      })
      
      // Navigate to console without a specific conversation
      router.push(getOrgUrl(`/clusters/${selectedCluster}/console`))
    } catch (error) {
      toast({
        title: 'Failed to delete conversation',
        description: 'An error occurred while deleting the conversation.',
        variant: 'destructive',
      })
    }
  }

  const currentConversation = conversationDbId ? {
    id: conversationDbId,
    agentName: selectedAgent,
    clusterName: selectedCluster || '',
    title: selectedAgent
  } : null

  return (
    <>
      <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-4 px-4 bg-white dark:bg-stone-950 h-[52px] flex items-center justify-between">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleNewConversation}
          className="h-8 w-8 p-0"
          title="New conversation"
        >
          <Plus className="h-4 w-4" />
        </Button>

        <Link 
          href={getOrgUrl(`/clusters/${selectedCluster}/agents/${selectedAgent}`)}
          className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300 hover:text-stone-600 dark:hover:text-stone-400 transition-colors cursor-pointer"
        >
          {selectedAgent}
        </Link>

        <Button
          variant="ghost"
          size="sm"
          onClick={() => setDeleteDialogOpen(true)}
          disabled={!conversationDbId}
          className="h-8 w-8 p-0 text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950"
          title="Delete conversation"
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      <DeleteConversationDialog
        conversation={currentConversation}
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onConfirm={handleDeleteConversation}
      />
    </>
  )
}
