'use client'

import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ArrowLeft, Edit, Trash2, Play, Pause } from 'lucide-react'
import { useTools } from '@/hooks/use-tools'

export default function ClusterToolDetailPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const toolName = params?.toolName as string

  const { data: tools, isLoading } = useTools()
  const tool = tools?.find((t: any) => t.metadata.name === toolName)

  const handleEdit = () => {
    router.push(`/clusters/${clusterName}/tools/${toolName}/edit`)
  }

  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete the tool "${toolName}"?`)) {
      return
    }

    try {
      const response = await fetch(`/api/tools/${toolName}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error('Failed to delete tool')
      }

      router.push(`/clusters/${clusterName}/tools`)
    } catch (error) {
      console.error('Error deleting tool:', error)
      alert('Failed to delete tool')
    }
  }

  const handleBack = () => {
    router.push(`/clusters/${clusterName}/tools`)
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <div className="h-8 w-48 bg-gray-200 rounded animate-pulse"></div>
              <div className="h-4 w-32 bg-gray-200 rounded mt-2 animate-pulse"></div>
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (!tool) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold">Tool Not Found</h1>
              <p className="text-muted-foreground mt-1">
                The tool "{toolName}" was not found in cluster "{clusterName}"
              </p>
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold">{tool.metadata.name}</h1>
              <p className="text-muted-foreground mt-1">
                Language Tool in {clusterName} cluster
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Button variant="outline" onClick={handleEdit}>
              <Edit className="h-4 w-4 mr-2" />
              Edit
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </Button>
          </div>
        </div>

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
              </div>

              {tool.spec.description && (
                <div>
                  <p className="text-sm font-medium">Description</p>
                  <p className="text-sm text-muted-foreground">{tool.spec.description}</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Configuration */}
          <Card>
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
              <CardDescription>Tool configuration settings</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {tool.spec.image && (
                  <div>
                    <p className="text-sm font-medium">Image</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.image}</p>
                  </div>
                )}

                {tool.spec.endpoint && (
                  <div>
                    <p className="text-sm font-medium">Endpoint</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.endpoint}</p>
                  </div>
                )}

                {tool.spec.port && (
                  <div>
                    <p className="text-sm font-medium">Port</p>
                    <p className="text-sm text-muted-foreground">{tool.spec.port}</p>
                  </div>
                )}

                {tool.spec.resources && (
                  <div>
                    <p className="text-sm font-medium">Resource Limits</p>
                    <div className="grid grid-cols-2 gap-2 text-sm text-muted-foreground">
                      {tool.spec.resources.requests && (
                        <div>
                          <span className="font-mono">CPU Request: {tool.spec.resources.requests.cpu}</span><br />
                          <span className="font-mono">Memory Request: {tool.spec.resources.requests.memory}</span>
                        </div>
                      )}
                      {tool.spec.resources.limits && (
                        <div>
                          <span className="font-mono">CPU Limit: {tool.spec.resources.limits.cpu}</span><br />
                          <span className="font-mono">Memory Limit: {tool.spec.resources.limits.memory}</span>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}