'use client'

import React, { useState, useMemo } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useOrganization } from '@/components/organization-provider'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ResourceHeader } from '@/components/ui/resource-header'
import { EventsActivity } from '@/components/ui/events-activity'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Bot, Plus, Activity, Clock, Zap, MoreHorizontal, Eye, Edit, Trash2, Search, MessageCircle } from 'lucide-react'
import Link from 'next/link'
import { LanguageAgent } from '@/types/agent'
import { useAgents } from '@/hooks/use-agents'
import { useWatchAgents } from '@/hooks/use-watch'

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

export default function ClusterAgents() {
  const params = useParams()
  const router = useRouter()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string

  const [search, setSearch] = useState('')
  const [executionModeFilter, setExecutionModeFilter] = useState<string>('all')

  // Use the agents hook for real-time updates
  const { data: agentsData, isLoading: loading, error: agentsError } = useAgents({
    clusterName,
    page: 1,
    limit: 1000, // Get all agents for list view
  })

  // Enable real-time updates via SSE watch
  useWatchAgents({ clusterName })

  const agents = agentsData?.data || []
  const error = agentsError ? (agentsError as Error).message : null

  // Filter agents based on search and execution mode
  const filteredAgents = useMemo(() => {
    return agents.filter((agent: any) => {
      const searchQuery = search.toLowerCase()
      const matchesSearch = !search ||
        agent.metadata.name.toLowerCase().includes(searchQuery) ||
        (agent.spec.displayName || '').toLowerCase().includes(searchQuery) ||
        (agent.spec.description || '').toLowerCase().includes(searchQuery)

      const matchesExecutionMode = executionModeFilter === 'all' ||
        (agent.spec.executionMode || 'autonomous').toLowerCase() === executionModeFilter.toLowerCase()

      return matchesSearch && matchesExecutionMode
    })
  }, [agents, search, executionModeFilter])

  const clusterAgents = filteredAgents
  
  // Get unique execution modes for filter dropdown
  const executionModes = React.useMemo(() => {
    const uniqueModes = Array.from(new Set(
      agents.map((agent: any) => agent.spec.executionMode || 'autonomous')
    )) as string[]
    return uniqueModes.sort()
  }, [agents])

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


  return (
    <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          icon={Bot}
          title="Agents"
          subtitle="Natural language-based goals and automations"
          actions={
            <Button asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/agents/new`)}>
                <Plus className="h-4 w-4 mr-2" />
                New Agent
              </Link>
            </Button>
          }
        />

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col md:flex-row gap-4">
              <div className="flex-1">
                <div className="relative">
                  <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search agents..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>
              </div>
              
              <Select value={executionModeFilter} onValueChange={setExecutionModeFilter}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="All Modes" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Modes</SelectItem>
                  {executionModes.map((mode) => (
                    <SelectItem key={mode} value={mode}>
                      {mode.charAt(0).toUpperCase() + mode.slice(1)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

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
                    <Link href={getOrgUrl(`/clusters/${clusterName}/agents/new`)}>
                      <Plus className="h-4 w-4 mr-2" />
                      Create Your First Agent
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            ) : (
              /* Agents Table */
              <Card>
                <CardHeader>
                  <CardTitle>Agents ({clusterAgents.length})</CardTitle>
                  <CardDescription>Language agents in this cluster</CardDescription>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>Mode</TableHead>
                        <TableHead>Models</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Age</TableHead>
                        <TableHead className="text-right">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {clusterAgents.map((agent: any) => (
                        <TableRow key={agent.metadata.name}>
                          <TableCell className="font-medium">
                            <Link 
                              href={getOrgUrl(`/clusters/${clusterName}/agents/${agent.metadata.name}`)}
                              className="hover:underline"
                            >
                              {agent.metadata.name}
                            </Link>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center space-x-1">
                              {getExecutionModeIcon(agent.spec.executionMode)}
                              <span className="text-sm capitalize">
                                {agent.spec.executionMode || 'autonomous'}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-wrap gap-1">
                              {(agent.spec.modelRefs || []).slice(0, 2).map((modelRef: any) => (
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
                              {(agent.spec.modelRefs?.length || 0) > 2 && (
                                <Badge variant="outline" className="text-xs">
                                  +{(agent.spec.modelRefs?.length || 0) - 2}
                                </Badge>
                              )}
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge className={getStatusColor(agent.status?.phase)}>
                              {agent.status?.phase || 'Unknown'}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <span className="text-sm text-muted-foreground">
                              {formatTimeAgo(agent.metadata.creationTimestamp)}
                            </span>
                          </TableCell>
                          <TableCell className="text-right">
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" className="h-8 w-8 p-0">
                                  <MoreHorizontal className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem asChild>
                                  <Link href={getOrgUrl(`/clusters/${clusterName}/console?agent=${agent.metadata.name}`)}>
                                    <MessageCircle className="h-4 w-4 mr-2" />
                                    Open in Console
                                  </Link>
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem asChild>
                                  <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agent.metadata.name}`)}>
                                    <Eye className="h-4 w-4 mr-2" />
                                    View Details
                                  </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                  <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agent.metadata.name}/edit`)}>
                                    <Edit className="h-4 w-4 mr-2" />
                                    Edit
                                  </Link>
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem
                                  className="text-destructive"
                                  onClick={() => {
                                    if (confirm(`Are you sure you want to delete agent "${agent.metadata.name}"?`)) {
                                      // TODO: Add delete functionality
                                      console.log('Delete agent:', agent.metadata.name)
                                    }
                                  }}
                                >
                                  <Trash2 className="h-4 w-4 mr-2" />
                                  Delete
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            )}
            
            {/* Agent Events - Show events related to agents in this cluster */}
            {!loading && !error && clusterAgents.length > 0 && (
              <EventsActivity
                title="Agent Events"
                description="Real-time events for agents in this cluster"
                clusterName={clusterName}
                resourceType="agent"
                limit={6}
                showNamespace={false}
              />
            )}
          </>
        )}
    </div>
  )
}