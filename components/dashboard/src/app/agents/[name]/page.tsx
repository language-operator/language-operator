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
  Bot, AlertCircle, CheckCircle, Clock, ArrowLeft, 
  Edit, FileText, Trash2, Play, Square, RotateCcw,
  Activity, Zap, DollarSign, TrendingUp
} from 'lucide-react'
import { useAgent, useDeleteAgent } from '@/hooks/use-agents'
import { useModels } from '@/hooks/use-models'
import { useTools } from '@/hooks/use-tools'
import { usePersonas } from '@/hooks/use-personas'
import { LanguageAgent } from '@/types/agent'
import { LanguageModel } from '@/types/model'
import { LanguageTool } from '@/types/tool'
import { LanguagePersona } from '@/types/persona'
import { Skeleton } from '@/components/ui/skeleton'

function formatTimeAgo(timestamp?: string | Date) {
  if (!timestamp) return 'Unknown'
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp
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

function formatDuration(nanoseconds?: number) {
  if (!nanoseconds) return 'N/A'
  const ms = nanoseconds / 1000000
  if (ms < 1000) return `${ms.toFixed(0)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

interface AgentOverviewProps {
  agent: LanguageAgent
}

function AgentOverview({ agent }: AgentOverviewProps) {
  // Fetch referenced resources
  const { data: modelsResponse } = useModels({})
  const { data: toolsResponse } = useTools({})
  const { data: personasResponse } = usePersonas({})

  const allModels = modelsResponse?.data || []
  const allTools = toolsResponse?.data || []
  const allPersonas = personasResponse?.data || []

  // Find referenced resources
  const referencedModel = agent.spec.model?.name 
    ? allModels.find((model: LanguageModel) => model.metadata.name === agent.spec.model?.name)
    : null

  const referencedTools = agent.spec.tools
    ? agent.spec.tools.map((toolRef) => 
        allTools.find((tool: LanguageTool) => tool.metadata.name === toolRef.name)
      ).filter(Boolean)
    : []

  const referencedPersona = agent.spec.persona?.name
    ? allPersonas.find((persona: LanguagePersona) => persona.metadata.name === agent.spec.persona?.name)
    : null

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
              <p className="text-sm">{agent.metadata.name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Namespace</p>
              <p className="text-sm">{agent.metadata.namespace}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Execution Mode</p>
              <Badge variant="secondary">{agent.spec.executionMode}</Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Replicas</p>
              <p className="text-sm">
                {agent.status?.activeReplicas ?? 0} / {agent.spec.replicas ?? 1}
              </p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Created</p>
              <p className="text-sm">{formatTimeAgo(agent.metadata.creationTimestamp)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Model Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Model Reference</p>
              <div className="flex items-center space-x-2">
                <p className="text-sm">{agent.spec.model.name}</p>
                {referencedModel ? (
                  <Link href={`/models/${agent.spec.model.name}`}>
                    <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                      View Model →
                    </Badge>
                  </Link>
                ) : (
                  <Badge variant="destructive">Not Found</Badge>
                )}
              </div>
            </div>
            {referencedModel && (
              <>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Provider</p>
                  <p className="text-sm">{referencedModel.spec.provider}</p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Model Name</p>
                  <p className="text-sm">{referencedModel.spec.modelName}</p>
                </div>
                {referencedModel.spec.endpoint && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Endpoint</p>
                    <p className="text-sm font-mono text-xs">{referencedModel.spec.endpoint}</p>
                  </div>
                )}
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Status</p>
                  <Badge variant={referencedModel.status?.healthy ? 'default' : 'destructive'}>
                    {referencedModel.status?.healthy ? 'Healthy' : 'Unhealthy'}
                  </Badge>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Persona and Tools */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Persona Configuration</CardTitle>
          </CardHeader>
          <CardContent>
            {agent.spec.persona ? (
              <div className="space-y-4">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Persona Reference</p>
                  <div className="flex items-center space-x-2">
                    <p className="text-sm">{agent.spec.persona.name}</p>
                    {referencedPersona ? (
                      <Link href={`/personas/${agent.spec.persona.name}`}>
                        <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                          View Persona →
                        </Badge>
                      </Link>
                    ) : (
                      <Badge variant="destructive">Not Found</Badge>
                    )}
                  </div>
                </div>
                {referencedPersona && (
                  <>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Display Name</p>
                      <p className="text-sm">{referencedPersona.spec.displayName}</p>
                    </div>
                    {referencedPersona.spec.tone && (
                      <div>
                        <p className="text-sm font-medium text-muted-foreground">Tone</p>
                        <Badge variant="secondary">{referencedPersona.spec.tone}</Badge>
                      </div>
                    )}
                    {referencedPersona.spec.description && (
                      <div>
                        <p className="text-sm font-medium text-muted-foreground">Description</p>
                        <p className="text-sm">{referencedPersona.spec.description}</p>
                      </div>
                    )}
                  </>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No persona configured</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Tools Configuration ({agent.spec.tools?.length || 0})</CardTitle>
          </CardHeader>
          <CardContent>
            {agent.spec.tools && agent.spec.tools.length > 0 ? (
              <div className="space-y-3">
                {agent.spec.tools.map((toolRef, index) => {
                  const referencedTool = allTools.find((tool: LanguageTool) => tool.metadata.name === toolRef.name)
                  
                  return (
                    <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                      <div>
                        <div className="flex items-center space-x-2">
                          <p className="text-sm font-medium">{toolRef.name}</p>
                          {referencedTool ? (
                            <Link href={`/tools/${toolRef.name}`}>
                              <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer text-xs">
                                View →
                              </Badge>
                            </Link>
                          ) : (
                            <Badge variant="destructive" className="text-xs">Not Found</Badge>
                          )}
                        </div>
                        {referencedTool && (
                          <div className="mt-1">
                            <p className="text-xs text-muted-foreground">Type: {referencedTool.spec.type}</p>
                            <div className="flex items-center space-x-2 mt-1">
                              <Badge variant={referencedTool.status?.phase === 'Running' ? 'default' : 'secondary'} className="text-xs">
                                {referencedTool.status?.phase || 'Unknown'}
                              </Badge>
                            </div>
                          </div>
                        )}
                      </div>
                      <Badge variant="outline">Configured</Badge>
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No tools configured</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Cross-References Summary */}
      <Card>
        <CardHeader>
          <CardTitle>Resource Dependencies</CardTitle>
          <CardDescription>Overview of all referenced resources for this agent</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Language Model</p>
              <div className="flex items-center space-x-2">
                {referencedModel ? (
                  <>
                    <CheckCircle className="h-4 w-4 text-green-500" />
                    <Link href={`/models/${agent.spec.model.name}`} className="text-sm hover:underline">
                      {agent.spec.model.name}
                    </Link>
                    <Badge variant={referencedModel.status?.healthy ? 'default' : 'destructive'} className="text-xs">
                      {referencedModel.status?.healthy ? 'Healthy' : 'Unhealthy'}
                    </Badge>
                  </>
                ) : (
                  <>
                    <AlertCircle className="h-4 w-4 text-red-500" />
                    <span className="text-sm text-muted-foreground">{agent.spec.model.name} (Not Found)</span>
                  </>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Persona</p>
              <div className="flex items-center space-x-2">
                {agent.spec.persona?.name ? (
                  referencedPersona ? (
                    <>
                      <CheckCircle className="h-4 w-4 text-green-500" />
                      <Link href={`/personas/${agent.spec.persona.name}`} className="text-sm hover:underline">
                        {agent.spec.persona.name}
                      </Link>
                      <Badge variant="default" className="text-xs">Found</Badge>
                    </>
                  ) : (
                    <>
                      <AlertCircle className="h-4 w-4 text-red-500" />
                      <span className="text-sm text-muted-foreground">{agent.spec.persona.name} (Not Found)</span>
                    </>
                  )
                ) : (
                  <span className="text-sm text-muted-foreground">No persona configured</span>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Tools ({agent.spec.tools?.length || 0})</p>
              {agent.spec.tools && agent.spec.tools.length > 0 ? (
                <div className="space-y-1">
                  {agent.spec.tools.slice(0, 2).map((toolRef, index) => {
                    const tool = allTools.find((t: LanguageTool) => t.metadata.name === toolRef.name)
                    return (
                      <div key={index} className="flex items-center space-x-2">
                        {tool ? (
                          <>
                            <CheckCircle className="h-3 w-3 text-green-500" />
                            <Link href={`/tools/${toolRef.name}`} className="text-xs hover:underline">
                              {toolRef.name}
                            </Link>
                          </>
                        ) : (
                          <>
                            <AlertCircle className="h-3 w-3 text-red-500" />
                            <span className="text-xs text-muted-foreground">{toolRef.name} (Not Found)</span>
                          </>
                        )}
                      </div>
                    )
                  })}
                  {agent.spec.tools.length > 2 && (
                    <p className="text-xs text-muted-foreground">+{agent.spec.tools.length - 2} more...</p>
                  )}
                </div>
              ) : (
                <span className="text-sm text-muted-foreground">No tools configured</span>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Status and Conditions */}
      <Card>
        <CardHeader>
          <CardTitle>Status & Conditions</CardTitle>
        </CardHeader>
        <CardContent>
          {agent.status?.conditions && agent.status.conditions.length > 0 ? (
            <div className="space-y-3">
              {agent.status.conditions.map((condition, index) => (
                <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                  <div>
                    <p className="text-sm font-medium">{condition.type}</p>
                    {condition.message && (
                      <p className="text-xs text-muted-foreground">{condition.message}</p>
                    )}
                  </div>
                  <div className="flex items-center space-x-2">
                    {condition.status === 'True' ? (
                      <CheckCircle className="h-4 w-4 text-green-500" />
                    ) : condition.status === 'False' ? (
                      <AlertCircle className="h-4 w-4 text-red-500" />
                    ) : (
                      <Clock className="h-4 w-4 text-yellow-500" />
                    )}
                    <Badge variant={condition.status === 'True' ? 'default' : 'secondary'}>
                      {condition.status}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">No status conditions available</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

interface AgentMetricsProps {
  agent: LanguageAgent
}

function AgentMetrics({ agent }: AgentMetricsProps) {
  const metrics = agent.status?.metrics

  return (
    <div className="space-y-6">
      {/* Key Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Execution Count</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {agent.status?.executionCount?.toLocaleString() ?? 0}
            </div>
            <p className="text-xs text-muted-foreground">
              Total executions
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
            <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.averageLatency ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Average response time
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.costMetrics?.totalCost ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              {metrics?.costMetrics?.currency ?? 'USD'} total
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
              <p className="text-sm font-medium text-muted-foreground">Error Rate</p>
              <p className="text-sm">{metrics?.errorRate ?? 'N/A'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Total Requests</p>
              <p className="text-sm">{metrics?.totalRequests?.toLocaleString() ?? 'N/A'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Average Latency</p>
              <p className="text-sm">{metrics?.averageLatency ?? 'N/A'}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Cost Metrics</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {metrics?.costMetrics ? (
              <>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Total Cost</p>
                  <p className="text-sm">
                    {metrics.costMetrics.totalCost} {metrics.costMetrics.currency}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Cost per Execution</p>
                  <p className="text-sm">
                    {metrics.costMetrics.costPerExecution} {metrics.costMetrics.currency}
                  </p>
                </div>
                {metrics.costMetrics.billingPeriod && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Billing Period</p>
                    <p className="text-sm">{metrics.costMetrics.billingPeriod}</p>
                  </div>
                )}
              </>
            ) : (
              <p className="text-sm text-muted-foreground">No cost metrics available</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default function AgentDetailPage() {
  const params = useParams()
  const agentName = params.name as string
  const [activeTab, setActiveTab] = useState('overview')
  
  const { data: agentResponse, isLoading, error } = useAgent(agentName)
  const deleteAgent = useDeleteAgent()

  const agent = agentResponse?.data

  const getStatusIcon = (agent: LanguageAgent) => {
    const phase = agent.status?.phase || 'Unknown'
    
    if (phase === 'Running') {
      return <CheckCircle className="h-5 w-5 text-green-500" />
    } else if (phase === 'Pending') {
      return <Clock className="h-5 w-5 text-yellow-500" />
    } else if (phase === 'Failed') {
      return <AlertCircle className="h-5 w-5 text-red-500" />
    } else if (phase === 'Succeeded') {
      return <CheckCircle className="h-5 w-5 text-blue-500" />
    } else {
      return <AlertCircle className="h-5 w-5 text-gray-500" />
    }
  }

  const getStatusColor = (agent: LanguageAgent) => {
    const phase = agent.status?.phase || 'Unknown'
    
    if (phase === 'Running') {
      return 'bg-green-100 text-green-800'
    } else if (phase === 'Pending') {
      return 'bg-yellow-100 text-yellow-800'
    } else if (phase === 'Failed') {
      return 'bg-red-100 text-red-800'
    } else if (phase === 'Succeeded') {
      return 'bg-blue-100 text-blue-800'
    } else {
      return 'bg-gray-100 text-gray-800'
    }
  }

  const handleDeleteAgent = async () => {
    if (!agent || !agent.metadata.name) return
    
    if (confirm(`Are you sure you want to delete agent "${agent.metadata.name}"?`)) {
      try {
        await deleteAgent.mutateAsync(agent.metadata.name)
        // Redirect to agents list after successful deletion
        window.location.href = '/agents'
      } catch (error) {
        console.error('Failed to delete agent:', error)
        alert('Failed to delete agent. Please try again.')
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

  if (error || !agent) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Agent not found</h3>
            <p className="text-muted-foreground mb-4">
              The agent "{agentName}" could not be found.
            </p>
            <Link href="/agents">
              <Button variant="outline">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Agents
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
            <Link href="/agents">
              <Button variant="outline" size="icon">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <Bot className="h-8 w-8 text-blue-500" />
            <div>
              <div className="flex items-center space-x-3">
                <h1 className="text-3xl font-bold">{agent.metadata.name}</h1>
                <div className="flex items-center space-x-2">
                  {getStatusIcon(agent)}
                  <Badge className={getStatusColor(agent)}>
                    {agent.status?.phase || 'Unknown'}
                  </Badge>
                </div>
              </div>
              <p className="text-muted-foreground">
                Language Agent • {agent.metadata.namespace}
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Link href={`/agents/${agent.metadata.name}/logs`}>
              <Button variant="outline">
                <FileText className="h-4 w-4 mr-2" />
                View Logs
              </Button>
            </Link>
            <Link href={`/agents/${agent.metadata.name}/edit`}>
              <Button variant="outline">
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </Button>
            </Link>
            <Button 
              variant="destructive"
              onClick={handleDeleteAgent}
              disabled={deleteAgent.isPending}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              {deleteAgent.isPending ? 'Deleting...' : 'Delete'}
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
            <AgentOverview agent={agent} />
          </TabsContent>

          <TabsContent value="metrics" className="space-y-6">
            <AgentMetrics agent={agent} />
          </TabsContent>
        </Tabs>
      </div>
    </AuthenticatedLayout>
  )
}