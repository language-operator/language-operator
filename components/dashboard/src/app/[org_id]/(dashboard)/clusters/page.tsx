'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useOrganization } from '@/components/organization-provider'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AnimatedStatus } from '@/components/ui/animated-status'
import { ResourceHeader } from '@/components/ui/resource-header'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { 
  Plus, Search, Server, CheckCircle, AlertCircle, 
  Clock, Globe, Shield, Users, MoreHorizontal, 
  Edit, Trash2, Eye, Activity, Bot, 
  Link as LinkIcon, Boxes
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useClusters, useDeleteCluster } from '@/hooks/use-clusters'
import { useWatchClusters } from '@/hooks/use-watch'
import { useResourceNotifications } from '@/components/ui/resource-notifications'
import { LanguageCluster } from '@/types/cluster'
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

interface ClusterTableProps {
  clusters: LanguageCluster[]
  onDelete: (name: string) => void
  isDeleting?: boolean
  getOrgUrl: (path: string) => string
}

function ClusterTable({ clusters, onDelete, isDeleting, getOrgUrl }: ClusterTableProps) {
  const getStatusBadge = (cluster: LanguageCluster) => {
    const phase = cluster.status?.phase || 'Unknown'
    return <AnimatedStatus status={phase} size="sm" />
  }


  return (
    <Card>
      <CardHeader>
        <CardTitle>Clusters ({clusters.length})</CardTitle>
        <CardDescription>Language clusters in your organization</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Domain</TableHead>
              <TableHead>Agents</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Age</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {clusters.map((cluster) => (
              <TableRow key={cluster.metadata.name}>
                <TableCell className="font-medium">
                  <Link 
                    href={getOrgUrl(`/clusters/${cluster.metadata.name}`)}
                    className="hover:underline"
                  >
                    {cluster.metadata.name}
                  </Link>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-2">
                    <LinkIcon className="h-3 w-3 text-blue-500" />
                    <span className="text-sm font-mono">
                      {cluster.spec.domain || 'No domain configured'}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}/agents`)}>
                    <div className="flex items-center space-x-1 hover:text-primary cursor-pointer">
                      <Bot className="h-3 w-3 text-purple-500" />
                      <span className="text-sm">
                        {cluster.status?.agentCount || 0}
                      </span>
                    </div>
                  </Link>
                </TableCell>
                <TableCell>
                  {getStatusBadge(cluster)}
                </TableCell>
                <TableCell>
                  <span className="text-sm text-muted-foreground">
                    {formatTimeAgo(cluster.metadata.creationTimestamp)}
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
                        <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}`)}>
                          <Eye className="h-4 w-4 mr-2" />
                          View Details
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}/edit`)}>
                          <Edit className="h-4 w-4 mr-2" />
                          Edit
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={() => {
                          if (confirm(`Are you sure you want to delete cluster "${cluster.metadata.name}"?`)) {
                            onDelete(cluster.metadata.name!)
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
        {clusters.length === 0 && (
          <div className="text-center py-8">
            <Boxes className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">No clusters found</h3>
            <p className="text-muted-foreground mb-4">
              Create your first language cluster to get started.
            </p>
            <Link href={getOrgUrl('/clusters/new')}>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Create Cluster
              </Button>
            </Link>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function ClustersPage() {
  const { getOrgUrl } = useOrganization()
  const [search, setSearch] = useState('')
  const [phaseFilter, setPhaseFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<'name' | 'domain' | 'agents' | 'status' | 'age'>('name')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')

  const { 
    data: clustersResponse, 
    isLoading, 
    error,
    refetch 
  } = useClusters({
    search: search || undefined,
    phase: phaseFilter !== 'all' ? [phaseFilter] : undefined,
    sortBy,
    sortOrder,
    limit: 100,
  })

  // Enable real-time cluster updates with notifications
  const handleWatchEvent = useResourceNotifications({
    onEvent: (event) => {
      console.log('🔄 Cluster update:', event.type, event.data?.metadata?.name)
      // React Query cache is automatically invalidated by the watch hook
    },
    enabled: true,
    showAllEvents: false // Only show important events like creation, deletion, status changes
  })

  useWatchClusters({
    enabled: process.env.NODE_ENV !== 'development', // Disable in dev to avoid loops
    onEvent: handleWatchEvent
  })

  const deleteCluster = useDeleteCluster()

  const clusters = clustersResponse?.data || []
  const total = clustersResponse?.total || 0

  // Stats calculations
  const readyClusters = clusters.filter((c: LanguageCluster) => c.status?.phase === 'Ready').length
  const totalAgents = clusters.reduce((sum: number, c: LanguageCluster) => sum + (c.status?.agentCount || 0), 0)

  const handleDelete = async (name: string) => {
    try {
      await deleteCluster.mutateAsync(name)
      refetch()
    } catch (error) {
      console.error('Failed to delete cluster:', error)
      alert('Failed to delete cluster. Please try again.')
    }
  }

  if (isLoading) {
    return (
      
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
      
    )
  }

  if (error) {
    return (
      
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Failed to load clusters</h3>
            <p className="text-muted-foreground mb-4">
              There was an error loading your language clusters.
            </p>
            <Button onClick={() => refetch()}>
              Try Again
            </Button>
          </div>
        </div>
      
    )
  }

  return (
    
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          icon={Boxes}
          title="Clusters"
          subtitle="Logical groups of agents, tools, and models"
          actions={
            <Link href={getOrgUrl('/clusters/new')}>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Create Cluster
              </Button>
            </Link>
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
                    placeholder="Search clusters..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>
              </div>
              
              <Select value={phaseFilter} onValueChange={setPhaseFilter}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="All Phases" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Phases</SelectItem>
                  <SelectItem value="Ready">Ready</SelectItem>
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
                  <SelectItem value="domain-asc">Domain (A-Z)</SelectItem>
                  <SelectItem value="domain-desc">Domain (Z-A)</SelectItem>
                  <SelectItem value="agents-desc">Agents (Most)</SelectItem>
                  <SelectItem value="agents-asc">Agents (Least)</SelectItem>
                  <SelectItem value="status-desc">Status (Best First)</SelectItem>
                  <SelectItem value="status-asc">Status (Worst First)</SelectItem>
                  <SelectItem value="age-desc">Newest</SelectItem>
                  <SelectItem value="age-asc">Oldest</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Clusters Table */}
        <ClusterTable 
          clusters={clusters} 
          onDelete={handleDelete}
          isDeleting={deleteCluster.isPending}
          getOrgUrl={getOrgUrl}
        />
      </div>
    
  )
}