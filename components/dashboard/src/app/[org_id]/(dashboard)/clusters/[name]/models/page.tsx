'use client'

import React from 'react'
import { useParams } from 'next/navigation'
import { useOrganization } from '@/components/organization-provider'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Cpu, Plus, ExternalLink, MoreHorizontal, Eye, Edit, Trash2, Search } from 'lucide-react'
import Link from 'next/link'
import { useModels } from '@/hooks/use-models'
import { useWatchModels } from '@/hooks/use-watch'
import { EventsActivity } from '@/components/ui/events-activity'

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

export default function ClusterModels() {
  const params = useParams()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const [search, setSearch] = React.useState('')
  const [providerFilter, setProviderFilter] = React.useState<string>('all')

  // Use cluster-specific API endpoint with real-time updates
  const { data: modelsResponse, isLoading, error } = useModels({
    clusterName,
    limit: 100
  })

  // Enable real-time updates via SSE watch
  useWatchModels()

  const allModels = modelsResponse?.data || []
  
  
  
  // Filter models based on search and provider
  const filteredModels = allModels.filter((model: any) => {
    const searchQuery = search.toLowerCase()
    const matchesSearch = !search || 
      model.metadata.name.toLowerCase().includes(searchQuery) ||
      (model.spec.provider || '').toLowerCase().includes(searchQuery) ||
      (model.spec.modelName || '').toLowerCase().includes(searchQuery)
    
    const matchesProvider = providerFilter === 'all' || 
      (model.spec.provider || '').toLowerCase() === providerFilter.toLowerCase()
    
    return matchesSearch && matchesProvider
  })
  
  // For now, show filtered models (TODO: implement proper cluster-scoped filtering)
  const clusterModels = filteredModels
  
  // Get unique providers for filter dropdown
  const providers = React.useMemo(() => {
    const uniqueProviders = Array.from(new Set(
      allModels.map((model: any) => model.spec.provider).filter(Boolean)
    )) as string[]
    return uniqueProviders.sort()
  }, [allModels])

  const getStatusColor = (phase?: string) => {
    switch (phase) {
      case 'Ready':
      case 'Available':
        return 'bg-green-100 text-green-800'
      case 'Pending':
        return 'bg-yellow-100 text-yellow-800'
      case 'Failed':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading models...</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          icon={Cpu}
          title="Models"
          subtitle="Large language models used for reasoning"
          actions={
            <Button asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/models/new`)}>
                <Plus className="h-4 w-4 mr-2" />
                New Model
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
                    placeholder="Search models..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>
              </div>
              
              <Select value={providerFilter} onValueChange={setProviderFilter}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="All Providers" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Providers</SelectItem>
                  {providers.map((provider) => (
                    <SelectItem key={provider} value={provider}>
                      {provider}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Models List or Empty State */}
        {clusterModels.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-16">
              <Cpu className="h-16 w-16 text-gray-400 mb-4" />
              <CardTitle className="text-xl mb-2">No models yet</CardTitle>
              <CardDescription className="text-center max-w-md mb-6">
                Language models define the AI capabilities available in this cluster. 
                Add your first model to get started.
              </CardDescription>
              <Button asChild>
                <Link href={getOrgUrl(`/clusters/${clusterName}/models/new`)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Your First Model
                </Link>
              </Button>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>Models ({clusterModels.length})</CardTitle>
              <CardDescription>Language models in this cluster</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Model</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Age</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {clusterModels.map((model: any) => (
                    <TableRow key={model.metadata.name}>
                      <TableCell className="font-medium">
                        <Link 
                          href={getOrgUrl(`/clusters/${clusterName}/models/${model.metadata.name}`)}
                          className="hover:underline"
                        >
                          {model.metadata.name}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{model.spec.provider}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm font-mono">{model.spec.modelName}</span>
                      </TableCell>
                      <TableCell>
                        <Badge className={getStatusColor(model.status?.phase)}>
                          {model.status?.phase || 'Unknown'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-muted-foreground">
                          {formatTimeAgo(model.metadata.creationTimestamp)}
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
                              <Link href={getOrgUrl(`/clusters/${clusterName}/models/${model.metadata.name}`)}>
                                <Eye className="h-4 w-4 mr-2" />
                                View Details
                              </Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem asChild>
                              <Link href={getOrgUrl(`/clusters/${clusterName}/models/${model.metadata.name}/edit`)}>
                                <Edit className="h-4 w-4 mr-2" />
                                Edit
                              </Link>
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              className="text-destructive"
                              onClick={() => {
                                if (confirm(`Are you sure you want to delete model "${model.metadata.name}"?`)) {
                                  // TODO: Add delete functionality
                                  console.log('Delete model:', model.metadata.name)
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

        {/* Model Events */}
        <EventsActivity
          title="Model Events"
          description="Recent events for models in this cluster"
          clusterName={clusterName}
          resourceType="model"
          limit={10}
          showNamespace={false}
        />
    </div>
  )
}