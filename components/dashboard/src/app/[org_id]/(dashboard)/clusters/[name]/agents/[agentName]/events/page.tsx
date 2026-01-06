'use client'

import { useParams } from 'next/navigation'
import { useAgent } from '@/hooks/use-agents'
import { ResourceEventsActivity } from '@/components/ui/events-activity'
import { Card, CardContent } from '@/components/ui/card'

export default function AgentEventsPage() {
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
              <p className="text-gray-600">Loading events...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!agent) {
    return null // Layout handles error state
  }

  return (
    <div className="space-y-4">
      <ResourceEventsActivity
        resourceType="agent"
        resourceName={agentName}
        namespace={agent.metadata.namespace}
        limit={20}
      />
    </div>
  )
}
