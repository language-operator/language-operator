'use client'

import { useState, useEffect, useRef, useMemo } from 'react'
import { useRouter, useParams } from 'next/navigation'
import Link from 'next/link'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Bot, AlertCircle, CheckCircle, Clock, ArrowLeft, 
  Edit, FileText, Trash2, Activity, Zap, DollarSign, TrendingUp, Code, Terminal
} from 'lucide-react'
import { useAgent, useDeleteAgent } from '@/hooks/use-agents'
import { useModels } from '@/hooks/use-models'
import { useTools } from '@/hooks/use-tools'
import { usePersonas } from '@/hooks/use-personas'
import { LanguageAgent } from '@/types/agent'
import { Skeleton } from '@/components/ui/skeleton'
import { NotFound } from '@/components/ui/not-found'
// Simple ANSI code converter
function convertAnsiToHtml(text: string): string {
  if (!text) return text;
  
  // Basic ANSI color mappings
  const ansiToClass: { [key: string]: string } = {
    '\x1b[30m': '<span class="ansi-black-fg">',      // Black
    '\x1b[31m': '<span class="ansi-red-fg">',        // Red  
    '\x1b[32m': '<span class="ansi-green-fg">',      // Green
    '\x1b[33m': '<span class="ansi-yellow-fg">',     // Yellow
    '\x1b[34m': '<span class="ansi-blue-fg">',       // Blue
    '\x1b[35m': '<span class="ansi-magenta-fg">',    // Magenta
    '\x1b[36m': '<span class="ansi-cyan-fg">',       // Cyan
    '\x1b[37m': '<span class="ansi-white-fg">',      // White
    '\x1b[90m': '<span class="ansi-bright-black-fg">', // Bright Black (Gray)
    '\x1b[91m': '<span class="ansi-bright-red-fg">',   // Bright Red
    '\x1b[92m': '<span class="ansi-bright-green-fg">', // Bright Green
    '\x1b[93m': '<span class="ansi-bright-yellow-fg">', // Bright Yellow
    '\x1b[94m': '<span class="ansi-bright-blue-fg">',  // Bright Blue
    '\x1b[95m': '<span class="ansi-bright-magenta-fg">', // Bright Magenta
    '\x1b[96m': '<span class="ansi-bright-cyan-fg">',  // Bright Cyan
    '\x1b[97m': '<span class="ansi-bright-white-fg">', // Bright White
    '\x1b[1;36m': '<span class="ansi-bold ansi-cyan-fg">', // Bold Cyan
    '\x1b[1m': '<span class="ansi-bold">',             // Bold
    '\x1b[0m': '</span>',                              // Reset
  };
  
  let result = text;
  
  // Replace ANSI codes with HTML spans
  Object.entries(ansiToClass).forEach(([code, replacement]) => {
    result = result.split(code).join(replacement);
  });
  
  return result;
}

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

interface AgentOverviewProps {
  agent: LanguageAgent
  clusterName: string
}

function AgentOverview({ agent, clusterName }: AgentOverviewProps) {
  const { data: modelsResponse } = useModels({})
  const { data: toolsResponse } = useTools({})
  const { data: personasResponse } = usePersonas({})

  const allModels = modelsResponse?.data || []
  const allTools = toolsResponse?.data || []
  const allPersonas = personasResponse?.data || []

  const referencedModel = agent.spec.model?.name 
    ? allModels.find((model: any) => model.metadata.name === agent.spec.model?.name)
    : null

  const referencedTools = agent.spec.tools
    ? agent.spec.tools.map((toolRef) => 
        allTools.find((tool: any) => tool.metadata.name === toolRef.name)
      ).filter(Boolean)
    : []

  const referencedPersona = agent.spec.persona?.name
    ? allPersonas.find((persona: any) => persona.metadata.name === agent.spec.persona?.name)
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
              <p className="text-sm font-medium text-muted-foreground">Cluster</p>
              <p className="text-sm">{clusterName}</p>
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
                <p className="text-sm">{agent.spec.model?.name}</p>
                {referencedModel ? (
                  <Link href={`/clusters/${clusterName}/models/${agent.spec.model?.name}`}>
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
                      <Link href={`/clusters/${clusterName}/personas/${agent.spec.persona.name}`}>
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
                  const referencedTool = allTools.find((tool: any) => tool.metadata.name === toolRef.name)
                  
                  return (
                    <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                      <div>
                        <div className="flex items-center space-x-2">
                          <p className="text-sm font-medium">{toolRef.name}</p>
                          {referencedTool ? (
                            <Link href={`/clusters/${clusterName}/tools/${toolRef.name}`}>
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

interface AgentCodeProps {
  agent: LanguageAgent
  clusterName: string
}

function AgentCode({ agent, clusterName }: AgentCodeProps) {
  const [agentVersion, setAgentVersion] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch the LanguageAgentVersion referenced by the agent
  useEffect(() => {
    const fetchAgentVersion = async () => {
      if (!agent.spec.agentVersionRef) {
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        setError(null)
        
        const url = `/api/clusters/${clusterName}/agent-versions/${agent.spec.agentVersionRef.name}`
        console.log('Fetching agent version from:', url)
        
        const response = await fetch(url)
        console.log('Response status:', response.status, response.statusText)
        
        if (!response.ok) {
          const errorData = await response.text()
          console.error('Error response:', errorData)
          throw new Error(`Failed to fetch agent version: ${response.status} ${response.statusText}`)
        }
        
        const data = await response.json()
        console.log('Agent version data:', data)
        setAgentVersion(data.data)
      } catch (err) {
        console.error('Error fetching agent version:', err)
        setError(err instanceof Error ? err.message : 'Failed to load agent version')
      } finally {
        setLoading(false)
      }
    }

    fetchAgentVersion()
  }, [agent.spec.agentVersionRef, clusterName])

  const synthesisInfo = agent.status?.synthesisInfo
  const isSynthesized = agent.status?.conditions?.some(
    (condition: any) => condition.type === 'Synthesized' && condition.status === 'True'
  )

  if (loading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading synthesized code...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Code Synthesis Status */}
      <Card>
        <CardHeader>
          <CardTitle>Code Synthesis Status</CardTitle>
          <CardDescription>
            Status and metadata of code synthesis for this agent
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Synthesis Status</p>
              <Badge variant={isSynthesized ? 'default' : 'secondary'}>
                {isSynthesized ? 'Code Synthesized' : 'Not Synthesized'}
              </Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Last Synthesis</p>
              <p className="text-sm">
                {synthesisInfo?.lastSynthesisTime 
                  ? formatTimeAgo(synthesisInfo.lastSynthesisTime)
                  : 'Never'
                }
              </p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Synthesis Model</p>
              <p className="text-sm">
                {synthesisInfo?.synthesisModel || 'N/A'}
              </p>
            </div>
          </div>
          
          {synthesisInfo && (
            <div className="mt-4 pt-4 border-t">
              <div className="grid gap-4 md:grid-cols-4 text-sm">
                <div>
                  <p className="font-medium text-muted-foreground">Duration</p>
                  <p>{synthesisInfo.synthesisDuration ? `${synthesisInfo.synthesisDuration.toFixed(2)}s` : 'N/A'}</p>
                </div>
                <div>
                  <p className="font-medium text-muted-foreground">Attempts</p>
                  <p>{synthesisInfo.synthesisAttempts || 0}</p>
                </div>
                <div>
                  <p className="font-medium text-muted-foreground">Code Hash</p>
                  <p className="font-mono text-xs">{synthesisInfo.codeHash?.substring(0, 12) || 'N/A'}...</p>
                </div>
                <div>
                  <p className="font-medium text-muted-foreground">Instructions Hash</p>
                  <p className="font-mono text-xs">{synthesisInfo.instructionsHash?.substring(0, 12) || 'N/A'}...</p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Synthesized Code */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Code className="h-5 w-5" />
            Synthesized Agent Code
          </CardTitle>
          <CardDescription>
            Ruby DSL code synthesized for this agent's execution logic
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error ? (
            <div className="text-center py-16">
              <AlertCircle className="h-16 w-16 text-red-400 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2 text-red-600">Error Loading Code</h3>
              <p className="text-muted-foreground max-w-md mx-auto">{error}</p>
            </div>
          ) : agentVersion?.spec?.code ? (
            <div className="border rounded-lg">
              <div className="bg-muted p-3 border-b">
                <div className="flex items-center justify-between">
                  <p className="font-medium text-sm">Agent Definition (Ruby DSL)</p>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">Ruby</Badge>
                    <Badge variant="outline">v{agentVersion.spec.version || 1}</Badge>
                  </div>
                </div>
              </div>
              <div className="p-4">
                <pre className="text-sm bg-gray-50 p-4 rounded overflow-x-auto whitespace-pre-wrap">
                  <code>{agentVersion.spec.code}</code>
                </pre>
              </div>
            </div>
          ) : (
            <div className="text-center py-16">
              <Code className="h-16 w-16 text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">No Code Synthesized</h3>
              <p className="text-muted-foreground max-w-md mx-auto">
                This agent hasn't been synthesized yet. Code will appear here
                after the synthesis process completes successfully.
              </p>
              {agent.spec.agentVersionRef && (
                <p className="text-sm text-muted-foreground mt-2">
                  Agent Version Reference: {agent.spec.agentVersionRef.name}
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Agent Version Details */}
      {agentVersion && (
        <Card>
          <CardHeader>
            <CardTitle>Agent Version Details</CardTitle>
            <CardDescription>
              Metadata and information about the synthesized agent version
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Version Name</p>
                <p className="text-sm font-mono">{agentVersion.metadata.name}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Created</p>
                <p className="text-sm">{formatTimeAgo(agentVersion.metadata.creationTimestamp)}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Source Type</p>
                <Badge variant="outline">{agentVersion.spec.sourceType || 'manual'}</Badge>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Status</p>
                <Badge variant={agentVersion.status?.phase === 'Ready' ? 'default' : 'secondary'}>
                  {agentVersion.status?.phase || 'Unknown'}
                </Badge>
              </div>
            </div>
            
            {agentVersion.spec.description && (
              <div className="mt-4">
                <p className="text-sm font-medium text-muted-foreground">Description</p>
                <p className="text-sm">{agentVersion.spec.description}</p>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

interface AgentLogsProps {
  agent: LanguageAgent
  clusterName: string
}

function AgentLogs({ agent, clusterName }: AgentLogsProps) {
  const [logs, setLogs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  
  // Memoize ANSI converter for performance
  const convertLogsToHtml = useMemo(() => {
    return (logLines: string[]): string[] => {
      return logLines.map(line => convertAnsiToHtml(line))
    }
  }, [])

  const scrollToBottom = () => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [logs])

  useEffect(() => {
    fetchInitialLogs()
    return () => {
      // Cleanup on unmount
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
    }
  }, [agent.metadata.name, clusterName])

  const fetchInitialLogs = async () => {
    try {
      setLoading(true)
      setError(null)
      
      const response = await fetch(`/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs`)
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.status} ${response.statusText}`)
      }
      
      const data = await response.json()
      const logLines = data.logs ? data.logs.split('\n').filter((line: string) => line.trim()) : []
      setLogs(logLines)
    } catch (err) {
      console.error('Error fetching initial logs:', err)
      setError(err instanceof Error ? err.message : 'Failed to load logs')
    } finally {
      setLoading(false)
    }
  }

  const startStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    setIsStreaming(true)
    setError(null)
    
    const eventSource = new EventSource(`/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs/stream`)
    eventSourceRef.current = eventSource

    eventSource.onmessage = (event) => {
      const newLog = event.data
      if (newLog && newLog.trim()) {
        setLogs(prev => [...prev, newLog])
      }
    }

    eventSource.onerror = (error) => {
      console.error('EventSource error:', error)
      setError('Connection lost. Click "Start Streaming" to reconnect.')
      setIsStreaming(false)
      eventSource.close()
    }
  }

  const stopStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setIsStreaming(false)
  }

  const clearLogs = () => {
    setLogs([])
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading logs...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Log Controls */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Terminal className="h-5 w-5" />
                Agent Logs
              </CardTitle>
              <CardDescription>
                Real-time logs from the agent pod
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={clearLogs}
                disabled={logs.length === 0}
              >
                Clear
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={fetchInitialLogs}
              >
                Refresh
              </Button>
              {isStreaming ? (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={stopStreaming}
                >
                  Stop Streaming
                </Button>
              ) : (
                <Button
                  variant="default"
                  size="sm"
                  onClick={startStreaming}
                >
                  Start Streaming
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Error Display */}
      {error && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Log Output */}
      <Card>
        <CardContent className="p-0">
          <div className="bg-black text-white font-mono text-sm h-96 overflow-y-auto p-4">
            {logs.length === 0 ? (
              <div className="flex items-center justify-center h-full text-gray-500">
                No logs available
              </div>
            ) : (
              <div>
                {logs.map((log, index) => (
                  <div 
                    key={index} 
                    className="whitespace-pre-wrap break-words"
                    dangerouslySetInnerHTML={{ 
                      __html: convertAnsiToHtml(log) 
                    }}
                  />
                ))}
                <div ref={logsEndRef} />
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Streaming Status */}
      {isStreaming && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-green-600">
              <div className="animate-pulse w-2 h-2 bg-green-500 rounded-full" />
              <span className="text-sm">Streaming logs in real-time</span>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
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

export default function ClusterAgentDetailPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const agentName = params?.agentName as string
  const [activeTab, setActiveTab] = useState('overview')
  
  const { data: agentResponse, isLoading, error } = useAgent(agentName, clusterName)
  const deleteAgent = useDeleteAgent(clusterName)

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
        router.push(`/clusters/${clusterName}/agents`)
      } catch (error) {
        console.error('Failed to delete agent:', error)
        alert('Failed to delete agent. Please try again.')
      }
    }
  }

  const handleBack = () => {
    router.push(`/clusters/${clusterName}/agents`)
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
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
    // Determine if it's a 404 error or other error type
    const is404Error = error && (error as any)?.status === 404
    const errorMessage = error ? (error as Error).message : `Agent "${agentName}" could not be found in cluster "${clusterName}".`
    
    return (
      <AuthenticatedLayout>
        <NotFound
          title={is404Error ? 'Agent Not Found' : 'Error Loading Agent'}
          message={errorMessage}
          onBack={handleBack}
          backLabel="Back to Agents"
        />
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
                Language Agent • {clusterName} cluster
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Button 
              variant="outline"
              onClick={() => router.push(`/clusters/${clusterName}/agents/${agentName}/edit`)}
            >
              <Edit className="h-4 w-4 mr-2" />
              Edit
            </Button>
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
            <TabsTrigger value="code">Code</TabsTrigger>
            <TabsTrigger value="logs">Logs</TabsTrigger>
            <TabsTrigger value="metrics">Metrics</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            <AgentOverview agent={agent} clusterName={clusterName} />
          </TabsContent>

          <TabsContent value="code" className="space-y-6">
            <AgentCode agent={agent} clusterName={clusterName} />
          </TabsContent>

          <TabsContent value="logs" className="space-y-6">
            <AgentLogs agent={agent} clusterName={clusterName} />
          </TabsContent>

          <TabsContent value="metrics" className="space-y-6">
            <AgentMetrics agent={agent} />
          </TabsContent>
        </Tabs>
      </div>
    </AuthenticatedLayout>
  )
}