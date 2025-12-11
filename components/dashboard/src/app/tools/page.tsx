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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { 
  Plus, Search, Wrench, CheckCircle, AlertCircle, 
  Clock, Activity, TrendingUp, MoreHorizontal, 
  Edit, Trash2, Eye, Settings, Code, Globe, Database, Mail
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useTools, useDeleteTool } from '@/hooks/use-tools'
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

interface ToolTableProps {
  tools: LanguageTool[]
  onDelete: (name: string) => void
  isDeleting?: boolean
}

function ToolTable({ tools, onDelete, isDeleting }: ToolTableProps) {
  const getStatusIcon = (tool: LanguageTool) => {
    const phase = tool.status?.phase
    if (phase === 'Running') {
      return <CheckCircle className="h-4 w-4 text-green-500" />
    } else if (phase === 'Pending') {
      return <Clock className="h-4 w-4 text-yellow-500" />
    } else if (phase === 'Failed') {
      return <AlertCircle className="h-4 w-4 text-red-500" />
    } else {
      return <AlertCircle className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusBadge = (tool: LanguageTool) => {
    const phase = tool.status?.phase || 'Unknown'
    if (phase === 'Running') {
      return <Badge variant="default" className="bg-green-100 text-green-800">Running</Badge>
    } else if (phase === 'Pending') {
      return <Badge variant="secondary">Pending</Badge>
    } else if (phase === 'Failed') {
      return <Badge variant="destructive">Failed</Badge>
    } else {
      return <Badge variant="secondary">{phase}</Badge>
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tools ({tools.length})</CardTitle>
        <CardDescription>Language tools in your organization</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Deployment Mode</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Invocation Count</TableHead>
              <TableHead>Success Rate</TableHead>
              <TableHead>Age</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tools.map((tool) => (
              <TableRow key={tool.metadata.name}>
                <TableCell className="font-medium">
                  <div className="flex items-center space-x-2">
                    {getToolTypeIcon(tool.spec.type)}
                    <Link 
                      href={`/tools/${tool.metadata.name}`}
                      className="hover:underline"
                    >
                      {tool.metadata.name}
                    </Link>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{tool.spec.type}</Badge>
                </TableCell>
                <TableCell>
                  <span className="text-sm">
                    {tool.spec.scaling?.replicas ? `Replicated (${tool.spec.scaling.replicas})` : 'Single'}
                  </span>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-2">
                    {getStatusIcon(tool)}
                    {getStatusBadge(tool)}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-1">
                    <Activity className="h-3 w-3 text-blue-500" />
                    <span className="text-sm">
                      {tool.status?.metrics?.invocationCount?.toLocaleString() || 'N/A'}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-1">
                    <TrendingUp className="h-3 w-3 text-green-500" />
                    <span className="text-sm">
                      {tool.status?.metrics?.successRate || 'N/A'}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-sm text-muted-foreground">
                    {formatTimeAgo(tool.metadata.creationTimestamp)}
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
                        <Link href={`/tools/${tool.metadata.name}`}>
                          <Eye className="h-4 w-4 mr-2" />
                          View Details
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <Link href={`/tools/${tool.metadata.name}/edit`}>
                          <Edit className="h-4 w-4 mr-2" />
                          Edit
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={() => {
                          if (confirm(`Are you sure you want to delete tool "${tool.metadata.name}"?`)) {
                            onDelete(tool.metadata.name!)
                          }
                        }}
                        disabled={isDeleting}
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
        {tools.length === 0 && (
          <div className="text-center py-8">
            <Wrench className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">No tools found</h3>
            <p className="text-muted-foreground mb-4">
              Create your first language tool to get started.
            </p>
            <Link href="/tools/new">
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Create Tool
              </Button>
            </Link>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function ToolsPage() {
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('all')
  const [phaseFilter, setPhaseFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<'name' | 'type' | 'phase' | 'invocations' | 'age'>('name')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')

  const { 
    data: toolsResponse, 
    isLoading, 
    error,
    refetch 
  } = useTools({
    search: search || undefined,
    type: typeFilter !== 'all' ? [typeFilter] : undefined,
    phase: phaseFilter !== 'all' ? [phaseFilter] : undefined,
    sortBy,
    sortOrder,
    limit: 100,
  })

  const deleteTool = useDeleteTool()

  const tools = toolsResponse?.data || []
  const total = toolsResponse?.total || 0

  // Get unique types for filter dropdown
  const types = Array.from(new Set(tools.map(tool => tool.spec.type))).sort()

  // Stats calculations
  const runningTools = tools.filter(t => t.status?.phase === 'Running').length
  const totalInvocations = tools.reduce((sum, t) => sum + (t.status?.metrics?.invocationCount || 0), 0)
  const avgSuccessRate = tools.length > 0 
    ? tools.reduce((sum, t) => {
        const rate = t.status?.metrics?.successRate
        return sum + (rate ? parseFloat(rate.replace('%', '')) : 0)
      }, 0) / tools.length 
    : 0

  const handleDelete = async (name: string) => {
    try {
      await deleteTool.mutateAsync(name)
      refetch()
    } catch (error) {
      console.error('Failed to delete tool:', error)
      alert('Failed to delete tool. Please try again.')
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-9 w-32" />
          </div>
          <div className="grid gap-4 md:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24" />
            ))}
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Failed to load tools</h3>
            <p className="text-muted-foreground mb-4">
              There was an error loading your language tools.
            </p>
            <Button onClick={() => refetch()}>
              Try Again
            </Button>
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
          <div>
            <h1 className="text-3xl font-bold">Language Tools</h1>
            <p className="text-muted-foreground">
              Manage tools and capabilities available to your agents
            </p>
          </div>
          <Link href="/tools/new">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Tool
            </Button>
          </Link>
        </div>

        {/* Stats Cards */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Tools</CardTitle>
              <Wrench className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{total}</div>
              <p className="text-xs text-muted-foreground">
                All tool types
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Running Tools</CardTitle>
              <CheckCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{runningTools}</div>
              <p className="text-xs text-muted-foreground">
                {total > 0 ? Math.round((runningTools / total) * 100) : 0}% operational
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Invocations</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalInvocations.toLocaleString()}</div>
              <p className="text-xs text-muted-foreground">
                All time usage
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Avg Success Rate</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{avgSuccessRate.toFixed(1)}%</div>
              <p className="text-xs text-muted-foreground">
                Average success rate
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col md:flex-row gap-4">
              <div className="flex-1">
                <div className="relative">
                  <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search tools..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-8"
                  />
                </div>
              </div>
              
              <Select value={typeFilter} onValueChange={setTypeFilter}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="All Types" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Types</SelectItem>
                  {types.map((type) => (
                    <SelectItem key={type} value={type}>
                      {type}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={phaseFilter} onValueChange={setPhaseFilter}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="All Phases" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Phases</SelectItem>
                  <SelectItem value="Running">Running</SelectItem>
                  <SelectItem value="Pending">Pending</SelectItem>
                  <SelectItem value="Failed">Failed</SelectItem>
                </SelectContent>
              </Select>

              <Select value={`${sortBy}-${sortOrder}`} onValueChange={(value) => {
                const [newSortBy, newSortOrder] = value.split('-') as [typeof sortBy, typeof sortOrder]
                setSortBy(newSortBy)
                setSortOrder(newSortOrder)
              }}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="Sort by" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="name-asc">Name (A-Z)</SelectItem>
                  <SelectItem value="name-desc">Name (Z-A)</SelectItem>
                  <SelectItem value="type-asc">Type (A-Z)</SelectItem>
                  <SelectItem value="type-desc">Type (Z-A)</SelectItem>
                  <SelectItem value="phase-desc">Phase (Best First)</SelectItem>
                  <SelectItem value="phase-asc">Phase (Worst First)</SelectItem>
                  <SelectItem value="invocations-desc">Invocations (Most)</SelectItem>
                  <SelectItem value="invocations-asc">Invocations (Least)</SelectItem>
                  <SelectItem value="age-desc">Newest</SelectItem>
                  <SelectItem value="age-asc">Oldest</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Tools Table */}
        <ToolTable 
          tools={tools} 
          onDelete={handleDelete}
          isDeleting={deleteTool.isPending}
        />
      </div>
    </AuthenticatedLayout>
  )
}