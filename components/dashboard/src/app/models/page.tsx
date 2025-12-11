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
  Plus, Search, Filter, Brain, CheckCircle, AlertCircle, 
  Clock, DollarSign, Zap, Activity, MoreHorizontal, 
  Edit, Trash2, Eye
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useModels, useDeleteModel } from '@/hooks/use-models'
import { LanguageModel } from '@/types/model'
import { Skeleton } from '@/components/ui/skeleton'

function formatCurrency(amount?: number, currency = 'USD') {
  if (!amount) return 'N/A'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
  }).format(amount)
}

function formatLatency(latency?: number) {
  if (!latency) return 'N/A'
  if (latency < 1000) return `${latency.toFixed(0)}ms`
  return `${(latency / 1000).toFixed(1)}s`
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

interface ModelTableProps {
  models: LanguageModel[]
  onDelete: (name: string) => void
  isDeleting?: boolean
}

function ModelTable({ models, onDelete, isDeleting }: ModelTableProps) {
  const getHealthIcon = (model: LanguageModel) => {
    if (model.status?.healthy === true) {
      return <CheckCircle className="h-4 w-4 text-green-500" />
    } else if (model.status?.healthy === false) {
      return <AlertCircle className="h-4 w-4 text-red-500" />
    } else {
      return <Clock className="h-4 w-4 text-yellow-500" />
    }
  }

  const getHealthBadge = (model: LanguageModel) => {
    if (model.status?.healthy === true) {
      return <Badge variant="default" className="bg-green-100 text-green-800">Healthy</Badge>
    } else if (model.status?.healthy === false) {
      return <Badge variant="destructive">Unhealthy</Badge>
    } else {
      return <Badge variant="secondary">Unknown</Badge>
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Models ({models.length})</CardTitle>
        <CardDescription>Language models in your organization</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Provider</TableHead>
              <TableHead>Model</TableHead>
              <TableHead>Health</TableHead>
              <TableHead>Latency</TableHead>
              <TableHead>Cost</TableHead>
              <TableHead>Age</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {models.map((model) => (
              <TableRow key={model.metadata.name}>
                <TableCell className="font-medium">
                  <div className="flex items-center space-x-2">
                    <Brain className="h-4 w-4 text-blue-500" />
                    <Link 
                      href={`/models/${model.metadata.name}`}
                      className="hover:underline"
                    >
                      {model.metadata.name}
                    </Link>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{model.spec.provider}</Badge>
                </TableCell>
                <TableCell>
                  <span className="font-mono text-sm">{model.spec.modelName}</span>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-2">
                    {getHealthIcon(model)}
                    {getHealthBadge(model)}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-1">
                    <Zap className="h-3 w-3 text-yellow-500" />
                    <span className="text-sm">
                      {formatLatency(model.status?.metrics?.averageLatency)}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-1">
                    <DollarSign className="h-3 w-3 text-green-500" />
                    <span className="text-sm">
                      {formatCurrency(
                        typeof model.status?.metrics?.costMetrics?.totalCost === 'string' 
                          ? parseFloat(model.status.metrics.costMetrics.totalCost) 
                          : model.status?.metrics?.costMetrics?.totalCost,
                        model.spec.costTracking?.currency
                      )}
                    </span>
                  </div>
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
                        <Link href={`/models/${model.metadata.name}`}>
                          <Eye className="h-4 w-4 mr-2" />
                          View Details
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <Link href={`/models/${model.metadata.name}/edit`}>
                          <Edit className="h-4 w-4 mr-2" />
                          Edit
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={() => {
                          if (confirm(`Are you sure you want to delete model "${model.metadata.name}"?`)) {
                            onDelete(model.metadata.name!)
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
        {models.length === 0 && (
          <div className="text-center py-8">
            <Brain className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">No models found</h3>
            <p className="text-muted-foreground mb-4">
              Create your first language model to get started.
            </p>
            <Link href="/models/new">
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Create Model
              </Button>
            </Link>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function ModelsPage() {
  const [search, setSearch] = useState('')
  const [providerFilter, setProviderFilter] = useState<string>('all')
  const [healthFilter, setHealthFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<'name' | 'provider' | 'health' | 'latency' | 'cost' | 'age'>('name')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')

  const { 
    data: modelsResponse, 
    isLoading, 
    error,
    refetch 
  } = useModels({
    search: search || undefined,
    provider: providerFilter !== 'all' ? [providerFilter] : undefined,
    healthy: healthFilter === 'healthy' ? true : healthFilter === 'unhealthy' ? false : undefined,
    sortBy,
    sortOrder,
    limit: 100,
  })

  const deleteModel = useDeleteModel()

  const models = modelsResponse?.data || []
  const total = modelsResponse?.total || 0

  // Get unique providers for filter dropdown
  const providers = Array.from(new Set(models.map((model: LanguageModel) => model.spec.provider))).sort() as string[]

  // Stats calculations
  const healthyModels = models.filter((m: LanguageModel) => m.status?.healthy === true).length
  const totalCost = models.reduce((sum: number, m: LanguageModel) => {
    const cost = m.status?.metrics?.costMetrics?.totalCost
    return sum + (typeof cost === 'string' ? parseFloat(cost) : cost || 0)
  }, 0)
  const avgLatency = models.length > 0 
    ? models.reduce((sum: number, m: LanguageModel) => sum + (m.status?.metrics?.averageLatency || 0), 0) / models.length 
    : 0

  const handleDelete = async (name: string) => {
    try {
      await deleteModel.mutateAsync(name)
      refetch()
    } catch (error) {
      console.error('Failed to delete model:', error)
      alert('Failed to delete model. Please try again.')
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
            <h3 className="text-lg font-medium mb-2">Failed to load models</h3>
            <p className="text-muted-foreground mb-4">
              There was an error loading your language models.
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
            <h1 className="text-3xl font-bold">Language Models</h1>
            <p className="text-muted-foreground">
              Manage language models and their configurations
            </p>
          </div>
          <Link href="/models/new">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Model
            </Button>
          </Link>
        </div>

        {/* Stats Cards */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Models</CardTitle>
              <Brain className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{total}</div>
              <p className="text-xs text-muted-foreground">
                Across all providers
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Healthy Models</CardTitle>
              <CheckCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{healthyModels}</div>
              <p className="text-xs text-muted-foreground">
                {total > 0 ? Math.round((healthyModels / total) * 100) : 0}% success rate
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
              <Zap className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatLatency(avgLatency)}</div>
              <p className="text-xs text-muted-foreground">
                Response time
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
              <DollarSign className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatCurrency(totalCost)}</div>
              <p className="text-xs text-muted-foreground">
                This month
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
                    placeholder="Search models..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-8"
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

              <Select value={healthFilter} onValueChange={setHealthFilter}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="All Health States" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Health States</SelectItem>
                  <SelectItem value="healthy">Healthy</SelectItem>
                  <SelectItem value="unhealthy">Unhealthy</SelectItem>
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
                  <SelectItem value="provider-asc">Provider (A-Z)</SelectItem>
                  <SelectItem value="provider-desc">Provider (Z-A)</SelectItem>
                  <SelectItem value="health-desc">Health (Best First)</SelectItem>
                  <SelectItem value="health-asc">Health (Worst First)</SelectItem>
                  <SelectItem value="latency-asc">Latency (Fastest)</SelectItem>
                  <SelectItem value="latency-desc">Latency (Slowest)</SelectItem>
                  <SelectItem value="cost-desc">Cost (Highest)</SelectItem>
                  <SelectItem value="cost-asc">Cost (Lowest)</SelectItem>
                  <SelectItem value="age-desc">Newest</SelectItem>
                  <SelectItem value="age-asc">Oldest</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Models Table */}
        <ModelTable 
          models={models} 
          onDelete={handleDelete}
          isDeleting={deleteModel.isPending}
        />
      </div>
    </AuthenticatedLayout>
  )
}