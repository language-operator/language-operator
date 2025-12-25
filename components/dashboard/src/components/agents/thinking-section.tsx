'use client'

import { useState } from 'react'
import { ChevronDown, ChevronRight, Brain } from 'lucide-react'
import { cn } from '@/lib/utils'
import { formatThinkingContent } from '@/lib/message-parser'

interface ThinkingSectionProps {
  thinkingContent: string[]
  className?: string
}

export function ThinkingSection({ thinkingContent, className }: ThinkingSectionProps) {
  const [isExpanded, setIsExpanded] = useState(false)

  if (!thinkingContent || thinkingContent.length === 0) {
    return null
  }

  const formattedContent = formatThinkingContent(thinkingContent)

  return (
    <div className={cn("mb-2", className)}>
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center gap-2 px-3 py-2 text-xs text-stone-500 dark:text-stone-400 hover:text-stone-600 dark:hover:text-stone-300 transition-colors border-2 border-dashed border-stone-200 dark:border-stone-700 bg-stone-50 dark:bg-stone-800/30 rounded-lg w-full text-left group"
        type="button"
      >
        <Brain className="h-3 w-3 shrink-0" />
        <span className="font-medium">Thinking</span>
        <span className="text-stone-400 dark:text-stone-500">
          ({thinkingContent.length} section{thinkingContent.length > 1 ? 's' : ''})
        </span>
        <div className="ml-auto">
          {isExpanded ? (
            <ChevronDown className="h-3 w-3" />
          ) : (
            <ChevronRight className="h-3 w-3" />
          )}
        </div>
      </button>

      {isExpanded && (
        <div className="mt-2 px-4 py-3 text-xs text-stone-600 dark:text-stone-400 border border-stone-200 dark:border-stone-700 bg-stone-50/50 dark:bg-stone-800/20 rounded-lg">
          <div className="prose prose-sm prose-stone dark:prose-invert max-w-none">
            <pre className="whitespace-pre-wrap font-sans text-xs leading-relaxed">
              {formattedContent}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}