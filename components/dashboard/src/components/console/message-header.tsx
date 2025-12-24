'use client'

import { useConsole } from '@/contexts/console-context'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { PanelRightClose, PanelRightOpen, Bot } from 'lucide-react'

export function MessageHeader() {
  const { selectedAgent, isWorkspaceVisible, toggleWorkspace } = useConsole()

  if (!selectedAgent) return null

  return (
    <div className="py-3 px-4 bg-white dark:bg-stone-950">
      <div className="flex items-center justify-between">
        <h2 className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300">
          {selectedAgent}
        </h2>

        <Button
          variant="ghost"
          size="sm"
          onClick={toggleWorkspace}
        >
          {isWorkspaceVisible ? (
            <PanelRightClose className="h-3 w-3" />
          ) : (
            <PanelRightOpen className="h-3 w-3" />
          )}
        </Button>
      </div>
    </div>
  )
}
