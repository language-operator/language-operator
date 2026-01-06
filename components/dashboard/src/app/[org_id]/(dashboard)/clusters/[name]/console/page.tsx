'use client'

import { useSearchParams, useParams } from 'next/navigation'
import { useEffect, Suspense, useRef } from 'react'
import { ConsoleLayout } from '@/components/console/console-layout'
import { ConsoleProvider } from '@/contexts/console-context'
import { useConsole } from '@/contexts/console-context'
import { getLastConversation } from '@/lib/conversation-storage'

function ConsoleContentWithAutoLoad({ clusterName }: { clusterName: string }) {
  const { loadConversation, selectedAgent } = useConsole()
  const hasAutoLoaded = useRef(false)
  const searchParams = useSearchParams()

  // Extract URL parameters for initial agent selection
  const agentParam = searchParams.get('agent')

  useEffect(() => {
    // Only auto-load if we haven't already and there's no agent selected
    if (!hasAutoLoaded.current && !selectedAgent && !agentParam) {
      hasAutoLoaded.current = true
      
      // Try to load last conversation from localStorage
      const lastConv = getLastConversation()
      if (lastConv && lastConv.clusterName === clusterName) {
        // Verify conversation still exists and load it
        loadConversation(
          lastConv.conversationId,
          lastConv.agentName,
          lastConv.clusterName
        ).catch(error => {
          // If conversation doesn't exist anymore, fail silently
          console.log('Could not restore last conversation:', error)
        })
      }
    }
  }, [clusterName, loadConversation, selectedAgent, agentParam])

  return <ConsoleLayout />
}

function ConsoleContent({ clusterName }: { clusterName: string }) {
  const searchParams = useSearchParams()

  // Extract URL parameters for initial agent selection
  const agentParam = searchParams.get('agent')

  return (
    <ConsoleProvider initialAgent={agentParam} initialCluster={clusterName}>
      <ConsoleContentWithAutoLoad clusterName={clusterName} />
    </ConsoleProvider>
  )
}

export default function ConsolePage() {
  const params = useParams()
  const clusterName = params.name as string

  return (
    <Suspense fallback={<div className="flex h-full items-center justify-center"><div className="text-sm">Loading console...</div></div>}>
      <ConsoleContent clusterName={clusterName} />
    </Suspense>
  )
}
