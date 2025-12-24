'use client'

import { Bot, MessageSquare, Zap } from 'lucide-react'

export function WelcomeMessage() {
  return (
    <div className="flex items-center justify-center h-full bg-gradient-to-br from-stone-50 via-amber-50/30 to-neutral-50 dark:from-stone-950 dark:via-stone-900/50 dark:to-stone-950">
      <div className="text-center max-w-md px-6">
        <div className="inline-flex items-center justify-center w-16 h-16 mb-6 bg-stone-900 dark:bg-amber-400">
          <MessageSquare className="h-8 w-8 text-amber-400 dark:text-stone-900" />
        </div>

        <h1 className="text-2xl font-light tracking-widest uppercase text-stone-900 dark:text-stone-300 mb-3">
          Console
        </h1>

        <p className="text-sm text-stone-600 dark:text-stone-400 font-light">
          Conversations with your agents will appear here
        </p>
      </div>
    </div>
  )
}
