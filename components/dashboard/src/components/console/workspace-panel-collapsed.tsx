'use client'

import { useConsole } from '@/contexts/console-context'
import { Button } from '@/components/ui/button'
import { ChevronLeft } from 'lucide-react'

export function WorkspacePanelCollapsed() {
  const { toggleWorkspace } = useConsole()

  return (
    <div className="flex flex-col h-full">
      {/* Header with expand button */}
      <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-3 px-2 h-[52px] flex items-center justify-center">
        <Button
          variant="ghost"
          size="sm"
          onClick={toggleWorkspace}
          className="h-6 w-6 p-0 hover:bg-stone-300/50 dark:hover:bg-stone-700/50 text-stone-600 dark:text-stone-400"
          title="Expand workspace"
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
      </div>

      {/* Empty content area for future use */}
      <div className="flex-1 overflow-y-auto py-2">
        {/* Content placeholder */}
      </div>
    </div>
  )
}