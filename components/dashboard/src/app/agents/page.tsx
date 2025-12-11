'use client'

import { useState } from 'react'
import Link from 'next/link'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { 
  Plus, Bot, AlertCircle, CheckCircle, Clock, Loader2, Search, 
  Filter, MoreVertical, Eye, Edit, Trash2, FileText 
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useAgents, useDeleteAgent } from '@/hooks/use-agents'
import { LanguageAgent } from '@/types/agent'

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

export default function AgentsPage() {
  const [search, setSearch] = useState('')
  const [phaseFilter, setPhaseFilter] = useState('all')
  const [modeFilter, setModeFilter] = useState('all')
  const [sortBy, setSortBy] = useState<'name' | 'phase' | 'executions' | 'age'>('name')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')

  // Use our enhanced useAgents hook with filters
  const { data: agentsResponse, isLoading, error } = useAgents({
    search: search || undefined,
    phase: phaseFilter !== 'all' ? [phaseFilter] : undefined,
    executionMode: modeFilter !== 'all' ? [modeFilter] : undefined,
    sortBy,
    sortOrder,
    limit: 100,
  })

  const agents = agentsResponse?.data || []
  const deleteAgent = useDeleteAgent()

  const getStatusIcon = (agent: LanguageAgent) => {
    const phase = agent.status?.phase || 'Unknown'
    
    if (phase === 'Running') {
      return <CheckCircle className="h-4 w-4 text-green-500" />
    } else if (phase === 'Pending') {
      return <Clock className="h-4 w-4 text-yellow-500" />
    } else if (phase === 'Failed') {
      return <AlertCircle className="h-4 w-4 text-red-500" />
    } else if (phase === 'Succeeded') {
      return <CheckCircle className="h-4 w-4 text-blue-500" />
    } else {
      return <AlertCircle className="h-4 w-4 text-gray-500" />
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

  const getDisplayStatus = (agent: LanguageAgent) => {
    return agent.status?.phase || 'Unknown'
  }

  const handleDeleteAgent = async (agentName: string) => {
    if (confirm(`Are you sure you want to delete agent "${agentName}"?`)) {
      try {
        await deleteAgent.mutateAsync(agentName)
      } catch (error) {
        console.error('Failed to delete agent:', error)
        alert('Failed to delete agent. Please try again.')
      }
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin" />
          <span className="ml-2">Loading agents...</span>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <AlertCircle className="h-8 w-8 text-red-500" />
          <span className="ml-2">Failed to load agents</span>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Language Agents</h1>
            <p className="text-muted-foreground mt-2">
              Manage your AI agents and their configurations
            </p>
          </div>
          <Link href="/agents/new">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Agent
            </Button>
          </Link>
        </div>

        {/* Search and Filters */}
        <Card>
          <CardContent className="p-6">
            <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <div className="flex flex-1 gap-4">
                <div className="relative flex-1 max-w-sm">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    placeholder="Search agents..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>
                
                <Select value={phaseFilter} onValueChange={setPhaseFilter}>
                  <SelectTrigger className="w-[150px]">
                    <Filter className="h-4 w-4 mr-2" />
                    <SelectValue placeholder="Phase" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Phases</SelectItem>
                    <SelectItem value="Pending">Pending</SelectItem>
                    <SelectItem value="Running">Running</SelectItem>
                    <SelectItem value="Succeeded">Succeeded</SelectItem>
                    <SelectItem value="Failed">Failed</SelectItem>
                    <SelectItem value="Unknown">Unknown</SelectItem>
                  </SelectContent>
                </Select>

                <Select value={modeFilter} onValueChange={setModeFilter}>
                  <SelectTrigger className="w-[150px]">
                    <SelectValue placeholder="Mode" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Modes</SelectItem>
                    <SelectItem value="worker">Worker</SelectItem>
                    <SelectItem value="server">Server</SelectItem>
                    <SelectItem value="hybrid">Hybrid</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="text-sm text-muted-foreground">
                {agentsResponse?.total || 0} agents
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Agents</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{agents.length}</div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Running</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {agents.filter(a => a.status?.phase === 'Running').length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Pending</CardTitle>
              <Clock className="h-4 w-4 text-yellow-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {agents.filter(a => a.status?.phase === 'Pending').length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Failed</CardTitle>
              <AlertCircle className="h-4 w-4 text-red-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {agents.filter(a => a.status?.phase === 'Failed').length}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Agents Table */}
        <Card>
          <CardContent className="p-0">
            {agents.length === 0 ? (
              <div className="flex items-center justify-center h-64">
                <div className="text-center">
                  <Bot className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <h3 className="text-lg font-medium mb-2">No agents found</h3>
                  <p className="text-muted-foreground mb-4">Get started by creating your first language agent.</p>
                  <Link href="/agents/new">
                    <Button>
                      <Plus className="h-4 w-4 mr-2" />
                      Create Agent
                    </Button>
                  </Link>
                </div>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[250px]">Name</TableHead>
                    <TableHead>Mode</TableHead>
                    <TableHead>Phase</TableHead>
                    <TableHead>Replicas</TableHead>
                    <TableHead>Executions</TableHead>
                    <TableHead>Success Rate</TableHead>
                    <TableHead>Age</TableHead>
                    <TableHead className="w-[70px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {agents.map((agent) => (
                    <TableRow key={agent.metadata.name}>
                      <TableCell>
                        <div className="flex items-center space-x-3">
                          <Bot className="h-4 w-4 text-muted-foreground" />
                          <div>
                            <Link 
                              href={`/agents/${agent.metadata.name}`}
                              className="font-medium hover:underline"
                            >
                              {agent.metadata.name}
                            </Link>
                            <p className="text-sm text-muted-foreground">
                              {agent.metadata.namespace}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">
                          {agent.spec.executionMode}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          {getStatusIcon(agent)}
                          <Badge variant="outline" className={getStatusColor(agent)}>
                            {getDisplayStatus(agent)}
                          </Badge>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {agent.status?.activeReplicas ?? agent.spec.replicas ?? 0}
                          {agent.spec.replicas ? ` / ${agent.spec.replicas}` : ''}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {agent.status?.executionCount?.toLocaleString() ?? 0}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {agent.status?.metrics?.successRate ?? 'N/A'}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-muted-foreground">
                          {formatTimeAgo(agent.metadata.creationTimestamp)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <MoreVertical className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem asChild>
                              <Link href={`/agents/${agent.metadata.name}`}>
                                <Eye className="h-4 w-4 mr-2" />
                                View Details
                              </Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem asChild>
                              <Link href={`/agents/${agent.metadata.name}/edit`}>
                                <Edit className="h-4 w-4 mr-2" />
                                Edit
                              </Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem asChild>
                              <Link href={`/agents/${agent.metadata.name}/logs`}>
                                <FileText className="h-4 w-4 mr-2" />
                                View Logs
                              </Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => handleDeleteAgent(agent.metadata.name || '')}
                              className="text-red-600"
                              disabled={deleteAgent.isPending}
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
            )}
          </CardContent>
        </Card>
      </div>
    </AuthenticatedLayout>
  )
}