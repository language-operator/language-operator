'use client'

import { useParams } from 'next/navigation'
import { useAgent } from '@/hooks/use-agents'
import { AgentWorkspace } from '@/components/workspace/agent-workspace'
import { Card, CardContent } from '@/components/ui/card'
import { FolderOpen } from 'lucide-react'

export default function AgentWorkspacePage() {
  const params = useParams()
  const clusterName = params.name as string
  const agentName = params.agentName as string

  const { data: agentResponse, isLoading } = useAgent(agentName, clusterName)
  const agent = agentResponse?.data

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading workspace...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!agent) {
    return null // Layout handles error state
  }

  // Check if workspace is enabled
  if (!agent.spec?.workspace?.enabled) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-16">
          <FolderOpen className="w-16 h-16 text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium mb-2">Workspace Not Enabled</h3>
          <p className="text-muted-foreground text-center max-w-md">
            This agent doesn't have workspace storage enabled.
            Enable workspace in the agent configuration to access files.
          </p>
        </CardContent>
      </Card>
    )
  }

  return <AgentWorkspace agent={agent} clusterName={clusterName} />
}
