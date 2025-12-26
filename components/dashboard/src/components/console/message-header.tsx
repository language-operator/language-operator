'use client'

import { useConsole } from '@/contexts/console-context'

export function MessageHeader() {
  const { selectedAgent } = useConsole()

  if (!selectedAgent) return null

  return (
    <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-4 px-4 bg-white dark:bg-stone-950 h-[52px] flex items-center justify-center">
      <h2 className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300">
        {selectedAgent}
      </h2>
    </div>
  )
}
