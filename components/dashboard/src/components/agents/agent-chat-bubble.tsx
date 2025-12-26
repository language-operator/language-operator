'use client'

import { forwardRef } from 'react'
import { ChatMessage } from '@/types/chat'
import { Bot, User, AlertCircle, Clock, Check, CheckCheck } from 'lucide-react'
import { cn } from '@/lib/utils'
import { ThinkingSection } from './thinking-section'
import { MarkdownContent } from '@/components/ui/markdown-content'

interface ChatBubbleProps {
  message: ChatMessage
  className?: string
}

function formatTime(timestamp: string) {
  return new Date(timestamp).toLocaleTimeString([], { 
    hour: '2-digit', 
    minute: '2-digit' 
  })
}

function getStatusIcon(status?: string) {
  switch (status) {
    case 'sending':
      return <Clock className="h-3 w-3 text-stone-400" />
    case 'sent':
      return <Check className="h-3 w-3 text-stone-500" />
    case 'delivered':
      return <CheckCheck className="h-3 w-3 text-stone-600" />
    case 'error':
      return <AlertCircle className="h-3 w-3 text-red-500" />
    default:
      return null
  }
}

export const ChatBubble = forwardRef<HTMLDivElement, ChatBubbleProps>(
  ({ message, className }, ref) => {
    const isUser = message.role === 'user'
    const isAssistant = message.role === 'assistant'
    const isSystem = message.role === 'system'

    if (isSystem) {
      return (
        <div ref={ref} className={cn("flex justify-center my-4", className)}>
          <div className="text-xs text-stone-500 bg-stone-100 dark:bg-stone-800 dark:text-stone-400 px-3 py-1 rounded-full max-w-xs text-center">
            {message.content}
          </div>
        </div>
      )
    }

    return (
      <div 
        ref={ref} 
        className={cn(
          "flex gap-3 mb-4 group",
          isUser ? "flex-row-reverse" : "flex-row",
          className
        )}
      >
        {/* Avatar */}
        <div className={cn(
          "flex items-center justify-center w-8 h-8 rounded-full shrink-0",
          isUser 
            ? "bg-amber-100 text-amber-600 dark:bg-amber-900/50 dark:text-amber-400"
            : "bg-blue-100 text-blue-600 dark:bg-blue-900/50 dark:text-blue-400"
        )}>
          {isUser ? (
            <User className="h-4 w-4" />
          ) : (
            <Bot className="h-4 w-4" />
          )}
        </div>

        {/* Message Container */}
        <div className={cn(
          "flex flex-col gap-1 max-w-[70%]",
          isUser ? "items-end" : "items-start"
        )}>
          {/* Agent Name (for assistant messages) */}
          {isAssistant && message.agentName && (
            <div className="text-xs font-medium text-stone-600 dark:text-stone-400 px-1">
              {message.agentName}
            </div>
          )}

          {/* Thinking Section (for assistant messages with thinking content) */}
          {isAssistant && message.hasThinking && message.thinkingContent && (
            <ThinkingSection thinkingContent={message.thinkingContent} />
          )}

          {/* Message Bubble */}
          <div
            className={cn(
              "px-4 py-3 text-sm leading-relaxed break-words",
              isUser
                ? "bg-amber-600 text-white rounded-2xl rounded-tr-md"
                : "bg-white border border-stone-200 text-stone-900 rounded-2xl rounded-tl-md dark:bg-stone-800 dark:border-stone-700 dark:text-stone-100",
              message.status === 'error' && isUser && "bg-red-500"
            )}
          >
            {/* Render markdown for assistant messages, plain text for user messages */}
            {isAssistant ? (
              <MarkdownContent 
                content={message.responseContent || message.content} 
                className="prose-sm [&>*]:text-stone-900 dark:[&>*]:text-stone-100"
              />
            ) : (
              message.content
            )}
          </div>

          {/* Status and Timestamp */}
          <div className={cn(
            "flex items-center gap-2 text-xs text-stone-500 dark:text-stone-400 px-1 opacity-0 group-hover:opacity-100 transition-opacity",
            isUser ? "flex-row-reverse" : "flex-row"
          )}>
            <span>{formatTime(message.timestamp)}</span>
            {isUser && getStatusIcon(message.status)}
          </div>
        </div>
      </div>
    )
  }
)

ChatBubble.displayName = 'ChatBubble'

// Typing indicator component for when agent is responding
export function TypingIndicator({ agentName }: { agentName?: string }) {
  return (
    <div className="flex gap-3 mb-4">
      {/* Agent Avatar */}
      <div className="flex items-center justify-center w-8 h-8 rounded-full bg-blue-100 text-blue-600 dark:bg-blue-900/50 dark:text-blue-400 shrink-0">
        <Bot className="h-4 w-4" />
      </div>

      <div className="flex flex-col gap-1">
        {agentName && (
          <div className="text-xs font-medium text-stone-600 dark:text-stone-400 px-1">
            {agentName}
          </div>
        )}
        
        <div className="bg-white border border-stone-200 dark:bg-stone-800 dark:border-stone-700 px-4 py-3 rounded-2xl rounded-tl-md">
          <div className="flex items-center gap-1">
            <div className="flex space-x-1">
              <div className="w-2 h-2 bg-stone-400 rounded-full animate-pulse" style={{ animationDelay: '0ms' }} />
              <div className="w-2 h-2 bg-stone-400 rounded-full animate-pulse" style={{ animationDelay: '150ms' }} />
              <div className="w-2 h-2 bg-stone-400 rounded-full animate-pulse" style={{ animationDelay: '300ms' }} />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}