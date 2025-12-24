'use client'

import { useConsole } from '@/contexts/console-context'
import { ConversationSidebar } from './conversation-sidebar'
import { MessagePanel } from './message-panel'
import { WorkspacePanel } from './workspace-panel'
import { cn } from '@/lib/utils'

export function ConsoleLayout() {
  const { isWorkspaceVisible, selectedAgent } = useConsole()

  // Show sidebars when an agent is selected (active conversation)
  const hasActiveConversation = selectedAgent !== null

  return (
    <div className="flex h-[calc(100vh-4rem)] bg-stone-50 dark:bg-stone-950">
      {/* Left Column: Agents & Conversations - shown when conversation is active */}
      {hasActiveConversation && (
        <div className="flex-shrink-0 w-80 border-r border-stone-800/80 dark:border-stone-600/80 bg-stone-200 dark:bg-stone-950">
          <ConversationSidebar />
        </div>
      )}

      {/* Middle Column: Chat Messages */}
      <div className="flex-1 min-w-[500px] flex flex-col">
        <MessagePanel />
      </div>

      {/* Right Column: Workspace (collapsible) - shown when conversation is active */}
      {hasActiveConversation && (
        <div
          className={cn(
            'flex-shrink-0 border-l border-stone-800/80 dark:border-stone-600/80 bg-stone-200 dark:bg-stone-950 transition-all duration-300',
            isWorkspaceVisible ? 'w-[420px]' : 'w-0 overflow-hidden'
          )}
        >
          {isWorkspaceVisible && <WorkspacePanel />}
        </div>
      )}
    </div>
  )
}
