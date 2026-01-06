'use client'

import { useParams } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useTool } from '@/hooks/use-tools'
import { ResourceEventsActivity } from '@/components/ui/events-activity'

export default function ClusterToolDetailPage() {
  const params = useParams()
  const clusterName = params?.name as string
  const toolName = params?.toolName as string

  const { data: toolResponse, isLoading } = useTool(toolName, clusterName)
  const tool = toolResponse?.data

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-48 bg-gray-200 rounded animate-pulse"></div>
        <div className="h-64 w-full bg-gray-200 rounded animate-pulse"></div>
      </div>
    )
  }

  if (!tool) {
    return null // Layout handles error state
  }

  return (
    <div className="space-y-6">
      {/* Tool Details */}
      <div className="grid gap-6">
        {/* Overview */}
        <Card>
          <CardHeader>
            <CardTitle>Overview</CardTitle>
            <CardDescription>Basic tool information and status</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm font-medium">Name</p>
                <p className="text-sm text-muted-foreground">{tool.metadata.name}</p>
              </div>
              <div>
                <p className="text-sm font-medium">Type</p>
                <p className="text-sm text-muted-foreground">{tool.spec.type || 'Unknown'}</p>
              </div>
              <div>
                <p className="text-sm font-medium">Status</p>
                <Badge variant={tool.status?.phase === 'Ready' ? 'default' : 'secondary'}>
                  {tool.status?.phase || 'Unknown'}
                </Badge>
              </div>
              <div>
                <p className="text-sm font-medium">Created</p>
                <p className="text-sm text-muted-foreground">
                  {tool.metadata.creationTimestamp ? new Date(tool.metadata.creationTimestamp).toLocaleDateString() : 'Unknown'}
                </p>
              </div>
              {tool.spec.image && (
                <div>
                  <p className="text-sm font-medium">Image</p>
                  <p className="text-sm text-muted-foreground font-mono">{tool.spec.image}</p>
                </div>
              )}
              {tool.spec.port && (
                <div>
                  <p className="text-sm font-medium">Port</p>
                  <p className="text-sm text-muted-foreground">{tool.spec.port}</p>
                </div>
              )}
            </div>

            {tool.spec.description && (
              <div>
                <p className="text-sm font-medium">Description</p>
                <p className="text-sm text-muted-foreground">{tool.spec.description}</p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Resources */}
        {tool.spec.resources && (
          <Card>
            <CardHeader>
              <CardTitle>Resources</CardTitle>
              <CardDescription>Resource requests and limits</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4">
                {tool.spec.resources.requests?.cpu && (
                  <div>
                    <p className="text-sm font-medium">CPU Request</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.resources.requests.cpu}</p>
                  </div>
                )}
                {tool.spec.resources.limits?.cpu && (
                  <div>
                    <p className="text-sm font-medium">CPU Limit</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.resources.limits.cpu}</p>
                  </div>
                )}
                {tool.spec.resources.requests?.memory && (
                  <div>
                    <p className="text-sm font-medium">Memory Request</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.resources.requests.memory}</p>
                  </div>
                )}
                {tool.spec.resources.limits?.memory && (
                  <div>
                    <p className="text-sm font-medium">Memory Limit</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.resources.limits.memory}</p>
                  </div>
                )}
              </div>
              {tool.spec.endpoint && (
                <div className="mt-4">
                  <p className="text-sm font-medium">Endpoint</p>
                  <p className="text-sm text-muted-foreground font-mono">{tool.spec.endpoint}</p>
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>

      {/* Tool Events */}
      <ResourceEventsActivity
        resourceType="tool"
        resourceName={toolName}
        namespace={tool.metadata.namespace}
        limit={15}
      />
    </div>
  )
}