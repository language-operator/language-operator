'use client'

import { useState } from 'react'
import { AgentList } from './agent-list'
import { Search } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { useConsole } from '@/contexts/console-context'

export function ConversationSidebar() {
  const [searchQuery, setSearchQuery] = useState('')
  const { conversationListRefreshTrigger } = useConsole()

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-6 px-4">
        <h2 className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300">
          Conversations
        </h2>
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
