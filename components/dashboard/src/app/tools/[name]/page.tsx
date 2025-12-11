'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Wrench, AlertCircle, CheckCircle, Clock, ArrowLeft, 
  Edit, Trash2, Activity, TrendingUp, Code, Globe, 
  Database, Settings, Key, Shield, FileText, Monitor
} from 'lucide-react'
import { useTool, useDeleteTool } from '@/hooks/use-tools'
import { LanguageTool } from '@/types/tool'
import { Skeleton } from '@/components/ui/skeleton'

function formatTimeAgo(timestamp?: string) {
  if (!timestamp) return 'Unknown'
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (days > 0) return `${days} day${days !== 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours !== 1 ? 's' : ''} ago`
  if (minutes > 0) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`
  return 'Just now'
}

function getToolTypeIcon(type: string) {
  switch (type) {
    case 'webhook':
      return <Globe className="h-4 w-4 text-blue-500" />
    case 'container':
      return <Settings className="h-4 w-4 text-green-500" />
    case 'function':
      return <Code className="h-4 w-4 text-purple-500" />
    case 'builtin':
      return <Database className="h-4 w-4 text-orange-500" />
    default:
      return <Wrench className="h-4 w-4 text-gray-500" />
  }
}

interface ToolOverviewProps {
  tool: LanguageTool
}

function ToolOverview({ tool }: ToolOverviewProps) {
  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Name</p>
              <p className="text-sm">{tool.metadata.name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Namespace</p>
              <p className="text-sm">{tool.metadata.namespace}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Type</p>
              <div className="flex items-center space-x-2">
                {getToolTypeIcon(tool.spec.type)}
                <Badge variant="outline">{tool.spec.type}</Badge>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Description</p>
              <p className="text-sm">{tool.spec.description || 'No description provided'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Created</p>
              <p className="text-sm">{formatTimeAgo(tool.metadata.creationTimestamp)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Status & Health</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Phase</p>
              <div className="flex items-center space-x-2">
                {tool.status?.phase === 'Running' ? (
                  <>
                    <CheckCircle className="h-4 w-4 text-green-500" />
                    <Badge variant="default" className="bg-green-100 text-green-800">Running</Badge>
                  </>
                ) : tool.status?.phase === 'Pending' ? (
                  <>
                    <Clock className="h-4 w-4 text-yellow-500" />
                    <Badge variant="secondary">Pending</Badge>
                  </>
                ) : tool.status?.phase === 'Failed' ? (
                  <>
                    <AlertCircle className="h-4 w-4 text-red-500" />
                    <Badge variant="destructive">Failed</Badge>
                  </>
                ) : (
                  <>
                    <AlertCircle className="h-4 w-4 text-gray-500" />
                    <Badge variant="secondary">{tool.status?.phase || 'Unknown'}</Badge>
                  </>
                )}
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Deployment Mode</p>
              <p className="text-sm">
                {tool.spec.scaling?.replicas ? `Replicated (${tool.spec.scaling.replicas} replicas)` : 'Single Instance'}
              </p>
            </div>
            {tool.status?.lastHealthCheck && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Last Health Check</p>
                <p className="text-sm">{formatTimeAgo(tool.status.lastHealthCheck)}</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Configuration based on type */}
      {tool.spec.type === 'webhook' && tool.spec.webhook && (
        <Card>
          <CardHeader>
            <CardTitle>Webhook Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">URL</p>
              <p className="text-sm font-mono text-xs break-all">{tool.spec.webhook.url}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Method</p>
              <Badge variant="outline">{tool.spec.webhook.method || 'POST'}</Badge>
            </div>
            {tool.spec.webhook.timeoutSeconds && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Timeout</p>
                <p className="text-sm">{tool.spec.webhook.timeoutSeconds} seconds</p>
              </div>
            )}
            {tool.spec.webhook.headers && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Custom Headers</p>
                <div className="space-y-1">
                  {Object.entries(tool.spec.webhook.headers).map(([key, value]) => (
                    <div key={key} className="flex items-center space-x-2 text-xs">
                      <code className="bg-gray-100 px-1 rounded">{key}</code>
                      <span>:</span>
                      <code className="bg-gray-100 px-1 rounded">{value}</code>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {tool.spec.type === 'container' && tool.spec.container && (
        <Card>
          <CardHeader>
            <CardTitle>Container Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Image</p>
              <p className="text-sm font-mono">{tool.spec.container.image}</p>
            </div>
            {tool.spec.container.command && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Command</p>
                <div className="bg-gray-100 p-2 rounded text-xs font-mono">
                  {tool.spec.container.command.join(' ')}
                </div>
              </div>
            )}
            {tool.spec.container.env && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Environment Variables</p>
                <div className="space-y-1">
                  {tool.spec.container.env.map((env, idx) => (
                    <div key={idx} className="flex items-center space-x-2 text-xs">
                      <code className="bg-gray-100 px-1 rounded">{env.name}</code>
                      <span>=</span>
                      <code className="bg-gray-100 px-1 rounded">{env.value || '[from secret]'}</code>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {tool.spec.container.resources && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Resource Limits</p>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  {tool.spec.container.resources.limits?.cpu && (
                    <div>
                      <span className="font-medium">CPU:</span> {tool.spec.container.resources.limits.cpu}
                    </div>
                  )}
                  {tool.spec.container.resources.limits?.memory && (
                    <div>
                      <span className="font-medium">Memory:</span> {tool.spec.container.resources.limits.memory}
                    </div>
                  )}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {tool.spec.type === 'function' && tool.spec.function && (
        <Card>
          <CardHeader>
            <CardTitle>Function Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Runtime</p>
              <Badge variant="outline">{tool.spec.function.runtime}</Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Handler</p>
              <p className="text-sm font-mono">{tool.spec.function.handler}</p>
            </div>
            {tool.spec.function.code && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Code Source</p>
                <div className="bg-gray-100 p-3 rounded">
                  <div className="flex items-center space-x-2">
                    <FileText className="h-4 w-4" />
                    <span className="text-sm">
                      {tool.spec.function.code.source === 'inline' 
                        ? 'Inline code provided'
                        : `External source: ${tool.spec.function.code.source}`
                      }
                    </span>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Schema */}
      {tool.spec.schema && (
        <Card>
          <CardHeader>
            <CardTitle>Tool Schema</CardTitle>
            <CardDescription>Input and output schema for this tool</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Tool Name</p>
              <p className="text-sm font-mono">{tool.spec.schema.name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Description</p>
              <p className="text-sm">{tool.spec.schema.description}</p>
            </div>
            {tool.spec.schema.inputSchema && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Input Schema</p>
                <div className="bg-gray-50 p-3 rounded">
                  <pre className="text-xs overflow-auto">
                    {JSON.stringify(tool.spec.schema.inputSchema, null, 2)}
                  </pre>
                </div>
              </div>
            )}
            {tool.spec.schema.outputSchema && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Output Schema</p>
                <div className="bg-gray-50 p-3 rounded">
                  <pre className="text-xs overflow-auto">
                    {JSON.stringify(tool.spec.schema.outputSchema, null, 2)}
                  </pre>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Security */}
      {tool.spec.security && (
        <Card>
          <CardHeader>
            <CardTitle>Security Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {tool.spec.security.authentication && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Authentication</p>
                <div className="flex items-center space-x-2">
                  <Shield className="h-4 w-4 text-blue-500" />
                  <Badge variant="outline">{tool.spec.security.authentication.type}</Badge>
                </div>
              </div>
            )}
            {tool.spec.security.apiKeyRef && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">API Key Reference</p>
                <div className="flex items-center space-x-2">
                  <Key className="h-4 w-4 text-blue-500" />
                  <p className="text-sm font-mono">{tool.spec.security.apiKeyRef.name}</p>
                  <Badge variant="outline" className="text-xs">
                    {tool.spec.security.apiKeyRef.key || 'api-key'}
                  </Badge>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

interface ToolMetricsProps {
  tool: LanguageTool
}

function ToolMetrics({ tool }: ToolMetricsProps) {
  const metrics = tool.status?.metrics

  return (
    <div className="space-y-6">
      {/* Key Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Invocations</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.invocationCount?.toLocaleString() ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              All time invocations
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.successRate ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Success percentage
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Duration</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.averageDuration ? `${metrics.averageDuration}ms` : 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Average execution time
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Error Rate</CardTitle>
            <AlertCircle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.errorRate ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Error percentage
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Detailed Metrics */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Performance Metrics</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Total Executions</p>
              <p className="text-sm">{metrics?.invocationCount?.toLocaleString() ?? 'N/A'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Failed Executions</p>
              <p className="text-sm">{metrics?.failedInvocations?.toLocaleString() ?? 'N/A'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Last Execution</p>
              <p className="text-sm">{formatTimeAgo(metrics?.lastInvocation)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Endpoint Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {tool.status?.endpointStatus ? (
              <>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Health Check Status</p>
                  <div className="flex items-center space-x-2">
                    {tool.status.endpointStatus.healthy ? (
                      <>
                        <CheckCircle className="h-4 w-4 text-green-500" />
                        <Badge variant="default" className="bg-green-100 text-green-800">Healthy</Badge>
                      </>
                    ) : (
                      <>
                        <AlertCircle className="h-4 w-4 text-red-500" />
                        <Badge variant="destructive">Unhealthy</Badge>
                      </>
                    )}
                  </div>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Response Time</p>
                  <p className="text-sm">
                    {tool.status.endpointStatus.responseTime ? `${tool.status.endpointStatus.responseTime}ms` : 'N/A'}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Last Check</p>
                  <p className="text-sm">{formatTimeAgo(tool.status.endpointStatus.lastCheck)}</p>
                </div>
                {tool.status.endpointStatus.error && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Last Error</p>
                    <p className="text-sm text-red-600 bg-red-50 p-2 rounded">
                      {tool.status.endpointStatus.error}
                    </p>
                  </div>
                )}
              </>
            ) : (
              <p className="text-sm text-muted-foreground">No endpoint status available</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default function ToolDetailPage() {
  const params = useParams()
  const toolName = params.name as string
  const [activeTab, setActiveTab] = useState('overview')
  
  const { data: toolResponse, isLoading, error } = useTool(toolName)
  const deleteTool = useDeleteTool()

  const tool = toolResponse?.data

  const getStatusIcon = (tool: LanguageTool) => {
    const phase = tool.status?.phase
    if (phase === 'Running') {
      return <CheckCircle className="h-5 w-5 text-green-500" />
    } else if (phase === 'Pending') {
      return <Clock className="h-5 w-5 text-yellow-500" />
    } else if (phase === 'Failed') {
      return <AlertCircle className="h-5 w-5 text-red-500" />
    } else {
      return <AlertCircle className="h-5 w-5 text-gray-500" />
    }
  }

  const getStatusColor = (tool: LanguageTool) => {
    const phase = tool.status?.phase || 'Unknown'
    if (phase === 'Running') {
      return 'bg-green-100 text-green-800'
    } else if (phase === 'Pending') {
      return 'bg-yellow-100 text-yellow-800'
    } else if (phase === 'Failed') {
      return 'bg-red-100 text-red-800'
    } else {
      return 'bg-gray-100 text-gray-800'
    }
  }

  const handleDeleteTool = async () => {
    if (!tool || !tool.metadata.name) return
    
    if (confirm(`Are you sure you want to delete tool "${tool.metadata.name}"?`)) {
      try {
        await deleteTool.mutateAsync(tool.metadata.name)
        // Redirect to tools list after successful deletion
        window.location.href = '/tools'
      } catch (error) {
        console.error('Failed to delete tool:', error)
        alert('Failed to delete tool. Please try again.')
      }
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Skeleton className="h-8 w-8" />
            <div className="space-y-2">
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-4 w-32" />
            </div>
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error || !tool) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Tool not found</h3>
            <p className="text-muted-foreground mb-4">
              The tool "{toolName}" could not be found.
            </p>
            <Link href="/tools">
              <Button variant="outline">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Tools
              </Button>
            </Link>
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
            <Link href="/tools">
              <Button variant="outline" size="icon">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            {getToolTypeIcon(tool.spec.type)}
            <div>
              <div className="flex items-center space-x-3">
                <h1 className="text-3xl font-bold">{tool.metadata.name}</h1>
                <div className="flex items-center space-x-2">
                  {getStatusIcon(tool)}
                  <Badge className={getStatusColor(tool)}>
                    {tool.status?.phase || 'Unknown'}
                  </Badge>
                </div>
              </div>
              <p className="text-muted-foreground">
                {tool.spec.type} Tool • {tool.metadata.namespace}
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Link href={`/tools/${tool.metadata.name}/edit`}>
              <Button variant="outline">
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </Button>
            </Link>
            <Button 
              variant="destructive"
              onClick={handleDeleteTool}
              disabled={deleteTool.isPending}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              {deleteTool.isPending ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="metrics">Metrics</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            <ToolOverview tool={tool} />
          </TabsContent>

          <TabsContent value="metrics" className="space-y-6">
            <ToolMetrics tool={tool} />
          </TabsContent>
        </Tabs>
      </div>
    </AuthenticatedLayout>
  )
}