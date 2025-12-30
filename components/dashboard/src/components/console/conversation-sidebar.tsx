'use client'

import { useState } from 'react'
import { AgentList } from './agent-list'
import { ChevronLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useConsole } from '@/contexts/console-context'
import { useAgentFilter } from '@/hooks/use-agent-filter'

export function ConversationSidebar() {
  const [selectedAgentFilter, setSelectedAgentFilter] = useState('all')
  const { conversationListRefreshTrigger, toggleConversationSidebar, selectedCluster } = useConsole()

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-3 px-4 h-[52px] flex items-center justify-between bg-white dark:bg-stone-950">
        <h2 className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300">
          Conversations
        </h2>
        <Button
          variant="ghost"
          size="sm"
          onClick={toggleConversationSidebar}
          className="h-6 w-6 p-0 hover:bg-stone-300/50 dark:hover:bg-stone-700/50 text-stone-600 dark:text-stone-400"
          title="Collapse sidebar"
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
      </div>

      {/* Conversation List */}
      <div className="flex-1 overflow-y-auto">
        <AgentList 
          selectedAgentFilter={selectedAgentFilter} 
          onAgentFilterChange={setSelectedAgentFilter}
          refreshTrigger={conversationListRefreshTrigger} 
        />
      </div>
    </div>
  )
}
