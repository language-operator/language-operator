'use client'

import { useState } from 'react'
import { AgentList } from './agent-list'
import { Search, ChevronLeft, MessageSquare } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useConsole } from '@/contexts/console-context'

export function ConversationSidebar() {
  const [searchQuery, setSearchQuery] = useState('')
  const { conversationListRefreshTrigger, toggleConversationSidebar } = useConsole()

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-3 px-4 h-[52px] flex items-center justify-between">
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

      {/* Search */}
      <div className="p-4 border-b border-stone-800/80 dark:border-stone-600/80">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-stone-400" />
          <Input
            type="text"
            placeholder="Search conversations..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9 text-sm"
          />
        </div>
      </div>

      {/* Conversation List */}
      <div className="flex-1 overflow-y-auto">
        <AgentList searchQuery={searchQuery} refreshTrigger={conversationListRefreshTrigger} />
      </div>
    </div>
  )
}
