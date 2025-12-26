'use client'

import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { cn } from '@/lib/utils'

interface MarkdownContentProps {
  content: string
  className?: string
}

export function MarkdownContent({ content, className }: MarkdownContentProps) {
  return (
    <div className={cn("prose prose-sm prose-stone dark:prose-invert max-w-none", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // Customize paragraph spacing to work better in chat bubbles
          p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
          // Customize list spacing
          ul: ({ children }) => <ul className="mb-2 last:mb-0 pl-4">{children}</ul>,
          ol: ({ children }) => <ol className="mb-2 last:mb-0 pl-4">{children}</ol>,
          li: ({ children }) => <li className="mb-1">{children}</li>,
          // Customize headings for chat context
          h1: ({ children }) => <h1 className="text-base font-semibold mb-2">{children}</h1>,
          h2: ({ children }) => <h2 className="text-sm font-semibold mb-1">{children}</h2>,
          h3: ({ children }) => <h3 className="text-sm font-medium mb-1">{children}</h3>,
          // Code blocks
          code: ({ className, children, ...props }) => {
            const match = /language-(\w+)/.exec(className || '')
            return match ? (
              <pre className="bg-stone-100 dark:bg-stone-800 p-2 rounded text-xs overflow-x-auto">
                <code className={className} {...props}>
                  {children}
                </code>
              </pre>
            ) : (
              <code className="bg-stone-100 dark:bg-stone-800 px-1 py-0.5 rounded text-xs" {...props}>
                {children}
              </code>
            )
          },
          // Blockquotes
          blockquote: ({ children }) => (
            <blockquote className="border-l-2 border-stone-300 dark:border-stone-600 pl-3 italic text-stone-600 dark:text-stone-400">
              {children}
            </blockquote>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}