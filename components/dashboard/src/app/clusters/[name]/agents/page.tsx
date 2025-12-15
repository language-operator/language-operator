'use client'

import React, { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Bot, Plus, Activity, Clock, Zap } from 'lucide-react'
import Link from 'next/link'
import { LanguageAgent } from '@/types/agent'

export default function ClusterAgents() {
  const params = useParams()
  const router = useRouter()
  const clusterName = params?.name as string
  
  const [agents, setAgents] = useState<LanguageAgent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchAgents()
  }, [clusterName])

  const fetchAgents = async () => {
    try {
      setLoading(true)
      setError(null)

      // Fetch agents for this cluster using cluster-scoped API
      const agentsResponse = await fetch(`/api/clusters/${clusterName}/agents`)
      if (!agentsResponse.ok) {
        throw new Error('Failed to fetch agents')
      }
      const agentsData = await agentsResponse.json()
      setAgents(agentsData.data || [])
    } catch (err) {
      console.error('Error fetching agents:', err)
      setError(err instanceof Error ? err.message : 'Failed to load agents')
    } finally {
      setLoading(false)
    }
  }

  const clusterAgents = agents

  const getStatusColor = (phase?: string) => {
    switch (phase) {
      case 'Ready':
      case 'Running':
        return 'bg-green-100 text-green-800'
      case 'Pending':
        return 'bg-yellow-100 text-yellow-800'
      case 'Failed':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getExecutionModeIcon = (mode?: string) => {
    switch (mode) {
      case 'autonomous':
        return <Zap className="h-4 w-4" />
      case 'interactive':
        return <Activity className="h-4 w-4" />
      case 'scheduled':
        return <Clock className="h-4 w-4" />
      default:
        return <Bot className="h-4 w-4" />
    }
  }

  const handleAgentCardClick = (agentName: string) => (event: React.MouseEvent) => {
    // Don't navigate if clicking on buttons or other interactive elements
    const target = event.target as HTMLElement
    if (target.closest('button') || target.closest('a')) {
      return
    }
    
    router.push(`/clusters/${clusterName}/agents/${agentName}`)
  }

  const handleAgentCardKeyDown = (agentName: string) => (event: React.KeyboardEvent) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      router.push(`/clusters/${clusterName}/agents/${agentName}`)
    }
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          icon={Bot}
          iconColor="text-blue-500"
          title="Agents"
          subtitle={`AI agents running in the ${clusterName} cluster (${clusterAgents.length} agents)`}
          actions={
            <Button asChild>
              <Link href={`/clusters/${clusterName}/agents/new`}>
                <Plus className="h-4 w-4 mr-2" />
                New Agent
              </Link>
            </Button>
          }
        />

        {/* Loading State */}
        {loading && (
          <Card>
            <CardContent className="flex items-center justify-center py-16">
              <div className="text-center">
                <Bot className="h-8 w-8 animate-pulse mx-auto mb-4 text-gray-400" />
                <p className="text-gray-600">Loading agents...</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Error State */}
        {error && (
          <Card>
            <CardContent className="flex items-center justify-center py-16">
              <div className="text-center">
                <Bot className="h-8 w-8 mx-auto mb-4 text-red-400" />
                <p className="text-red-600 mb-2">Failed to load agents</p>
                <p className="text-gray-600 text-sm">{error}</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Agents List */}
        {!loading && !error && (
          <>
            {clusterAgents.length === 0 ? (
              /* Empty State */
              <Card>
                <CardContent className="flex flex-col items-center justify-center py-16">
                  <Bot className="h-16 w-16 text-gray-400 mb-4" />
                  <CardTitle className="text-xl mb-2">No agents yet</CardTitle>
                  <CardDescription className="text-center max-w-md mb-6">
                    Agents combine models, personas, and tools to create intelligent 
                    assistants. Deploy your first agent to get started.
                  </CardDescription>
                  <Button asChild>
                    <Link href={`/clusters/${clusterName}/agents/new`}>
                      <Plus className="h-4 w-4 mr-2" />
                      Create Your First Agent
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            ) : (
              /* Agents Grid */
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {clusterAgents.map((agent: any) => (
                  <Card 
                    key={agent.metadata.name} 
                    className="hover:shadow-md hover:bg-gray-50/50 transition-all cursor-pointer"
                    onClick={handleAgentCardClick(agent.metadata.name)}
                    onKeyDown={handleAgentCardKeyDown(agent.metadata.name)}
                    tabIndex={0}
                    role="button"
                    aria-label={`View details for agent ${agent.metadata.name}`}
                  >
                    <CardHeader className="pb-3">
                      <div className="flex items-center justify-between">
                        <CardTitle className="text-lg">{agent.metadata.name}</CardTitle>
                        <Badge className={getStatusColor(agent.status?.phase)}>
                          {agent.status?.phase || 'Unknown'}
                        </Badge>
                      </div>
                      <CardDescription>
                        {agent.spec.instructions || agent.spec.goal || 'No goal specified'}
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        <div className="flex items-center space-x-2 text-sm text-gray-600">
                          {getExecutionModeIcon(agent.spec.executionMode)}
                          <span className="capitalize">{agent.spec.executionMode || 'autonomous'}</span>
                        </div>
                        
                        <div className="space-y-1">
                          <div className="text-sm text-gray-500">Models:</div>
                          <div className="flex flex-wrap gap-1">
                            {(agent.spec.modelRefs || []).map((modelRef: any) => (
                              <Badge key={modelRef.name} variant="outline" className="text-xs">
                                {modelRef.name}
                              </Badge>
                            ))}
                            {/* Legacy support */}
                            {agent.spec.model && (
                              <Badge variant="outline" className="text-xs">
                                {agent.spec.model.name}
                              </Badge>
                            )}
                          </div>
                        </div>

                        {(agent.spec.toolRefs?.length > 0 || agent.spec.tools?.length > 0) && (
                          <div className="space-y-1">
                            <div className="text-sm text-gray-500">Tools:</div>
                            <div className="flex flex-wrap gap-1">
                              {(agent.spec.toolRefs || agent.spec.tools || []).map((toolRef: any) => (
                                <Badge key={toolRef.name} variant="outline" className="text-xs">
                                  {toolRef.name}
                                </Badge>
                              ))}
                            </div>
                          </div>
                        )}

                        <div className="pt-2 border-t">
                          <div className="text-xs text-gray-500">
                            Created: {new Date(agent.metadata.creationTimestamp).toLocaleDateString()}
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </AuthenticatedLayout>
  )
}