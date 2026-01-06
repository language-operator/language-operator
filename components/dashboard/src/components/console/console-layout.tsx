'use client'

import { useConsole } from '@/contexts/console-context'
import { ConversationSidebar } from './conversation-sidebar'
import { ConversationSidebarCollapsed } from './conversation-sidebar-collapsed'
import { MessagePanel } from './message-panel'
import { WorkspacePanel } from './workspace-panel'
import { WorkspacePanelCollapsed } from './workspace-panel-collapsed'
import { cn } from '@/lib/utils'
import { ChevronLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'

export function ConsoleLayout() {
  const { 
    isWorkspaceVisible, 
    isConversationSidebarVisible, 
    selectedAgent,
    toggleConversationSidebar,
    toggleWorkspace
  } = useConsole()

  // Show sidebars when an agent is selected (active conversation)
  const hasActiveConversation = selectedAgent !== null

  return (
    <div className="flex h-full bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950 relative">
      {/* Left Column: Agents & Conversations - always shown for browsing */}
      <div
        className={cn(
          'flex-shrink-0 border-r border-stone-800/80 dark:border-stone-600/80 backdrop-blur-sm transition-all duration-300',
          isConversationSidebarVisible ? 'w-80' : 'w-16 overflow-hidden'
        )}
      >
        {isConversationSidebarVisible ? (
          <ConversationSidebar />
        ) : (
          <ConversationSidebarCollapsed />
        )}
      </div>

      {/* Middle Column: Chat Messages */}
      <div className="flex-1 min-w-[500px] flex flex-col">
        <MessagePanel />
      </div>

      {/* Right Column: Workspace (collapsible) - shown when conversation is active */}
      {hasActiveConversation && (
        <div
          className={cn(
            'flex-shrink-0 border-l border-stone-800/80 dark:border-stone-600/80 backdrop-blur-sm transition-all duration-300',
            isWorkspaceVisible ? 'w-[420px]' : 'w-16 overflow-hidden'
          )}
        >
          {isWorkspaceVisible ? (
            <WorkspacePanel />
          ) : (
            <WorkspacePanelCollapsed />
          )}
        </div>
      )}
    </div>
  )
}
