'use client'

import { useConsole } from '@/contexts/console-context'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ChatBubble, TypingIndicator } from '@/components/agents/agent-chat-bubble'
import { useEffect, useRef } from 'react'

export function MessageStream() {
  const { getActiveConversation } = useConsole()
  const conversation = getActiveConversation()
  const scrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [conversation?.messages.length])

  if (!conversation) return null

  return (
    <ScrollArea className="h-full">
      <div className="p-6 space-y-4">
        {conversation.messages.map((message) => (
          <ChatBubble key={message.id} message={message} />
        ))}

        {conversation.isLoading && <TypingIndicator />}

        {conversation.error && (
          <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-sm text-red-600 dark:text-red-400">
            {conversation.error}
          </div>
        )}

        <div ref={scrollRef} />
      </div>
    </ScrollArea>
  )
}
