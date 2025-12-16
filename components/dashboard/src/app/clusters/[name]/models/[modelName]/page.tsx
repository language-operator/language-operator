'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { 
  Cpu, AlertCircle, CheckCircle, Clock, ArrowLeft, 
  Edit, FileText, Trash2, Activity, Zap, DollarSign, 
  TrendingUp, Globe, Shield, Key, Settings, MoreVertical, FileCode, Copy, Check
} from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useModel, useDeleteModel } from '@/hooks/use-models'
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

interface ModelOverviewProps {
  model: LanguageModel
}

function ModelOverview({ model }: ModelOverviewProps) {
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
              <p className="text-sm">{model.metadata.name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Namespace</p>
              <p className="text-sm">{model.metadata.namespace}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Provider</p>
              <Badge variant="outline">{model.spec.provider}</Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Model Name</p>
              <p className="text-sm font-mono">{model.spec.modelName}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Created</p>
              <p className="text-sm">{formatTimeAgo(model.metadata.creationTimestamp)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Health Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Phase</p>
              <p className="text-sm">{model.status?.phase || 'Unknown'}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Configuration */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Model Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {model.spec.endpoint && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Endpoint</p>
                <p className="text-sm font-mono text-xs break-all">{model.spec.endpoint}</p>
              </div>
            )}
            {model.spec.configuration && (
              <>
                {model.spec.configuration.temperature && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Temperature</p>
                    <p className="text-sm">{model.spec.configuration.temperature}</p>
                  </div>
                )}
                {model.spec.configuration.maxTokens && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Max Tokens</p>
                    <p className="text-sm">{model.spec.configuration.maxTokens.toLocaleString()}</p>
                  </div>
                )}
                {model.spec.configuration.topP && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Top P</p>
                    <p className="text-sm">{model.spec.configuration.topP}</p>
                  </div>
                )}
                {model.spec.configuration.timeoutMs && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Timeout</p>
                    <p className="text-sm">{model.spec.configuration.timeoutMs}ms</p>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Security & Authentication</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {model.spec.apiKeySecretRef ? (
              <div>
                <p className="text-sm font-medium text-muted-foreground">API Key Secret</p>
                <div className="flex items-center space-x-2">
                  <Key className="h-4 w-4 text-blue-500" />
                  <p className="text-sm font-mono">{model.spec.apiKeySecretRef.name}</p>
                  <Badge variant="outline" className="text-xs">
                    {model.spec.apiKeySecretRef.key || 'api-key'}
                  </Badge>
                </div>
              </div>
            ) : (
              <div>
                <p className="text-sm font-medium text-muted-foreground">API Key Secret</p>
                <p className="text-sm text-muted-foreground">Not configured</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Cost Tracking */}
      {model.spec.costTracking?.enabled && (
        <Card>
          <CardHeader>
            <CardTitle>Cost Tracking Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Currency</p>
                <p className="text-sm">{model.spec.costTracking.currency || 'USD'}</p>
              </div>
              {model.spec.costTracking.inputTokenCost && (
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Input Token Cost</p>
                  <p className="text-sm">{formatCurrency(model.spec.costTracking.inputTokenCost, model.spec.costTracking.currency)}</p>
                </div>
              )}
              {model.spec.costTracking.outputTokenCost && (
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Output Token Cost</p>
                  <p className="text-sm">{formatCurrency(model.spec.costTracking.outputTokenCost, model.spec.costTracking.currency)}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Rate Limits */}
      {model.spec.rateLimits && (
        <Card>
          <CardHeader>
            <CardTitle>Rate Limits</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              {model.spec.rateLimits.requestsPerMinute && (
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Requests per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.requestsPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.tokensPerMinute && (
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Tokens per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.tokensPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.concurrentRequests && (
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Concurrent Requests</p>
                  <p className="text-sm">{model.spec.rateLimits.concurrentRequests}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

interface ModelMetricsProps {
  model: LanguageModel
}

function ModelMetrics({ model }: ModelMetricsProps) {
  const metrics = model.status?.metrics

  return (
    <div className="space-y-6">
      {/* Key Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Requests</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.totalRequests?.toLocaleString() ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              All time requests
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
              {formatLatency(metrics?.averageLatency)}
            </div>
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
            <div className="text-2xl font-bold">
              {formatCurrency(typeof metrics?.costMetrics?.totalCost === 'string' ? parseFloat(metrics.costMetrics.totalCost) : metrics?.costMetrics?.totalCost, model.spec.costTracking?.currency)}
            </div>
            <p className="text-xs text-muted-foreground">
              {model.spec.costTracking?.currency || 'USD'} total
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
              <p className="text-sm font-medium text-muted-foreground">P95 Latency</p>
              <p className="text-sm">{formatLatency(metrics?.p95Latency)}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">P99 Latency</p>
              <p className="text-sm">{formatLatency(metrics?.p99Latency)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Cost Breakdown</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {metrics ? (
              <>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Total Cost</p>
                  <p className="text-sm">{formatCurrency(typeof metrics?.costMetrics?.totalCost === 'string' ? parseFloat(metrics.costMetrics.totalCost) : metrics?.costMetrics?.totalCost, model.spec.costTracking?.currency)}</p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Input Tokens</p>
                  <p className="text-sm">{metrics.costMetrics?.totalInputTokens?.toLocaleString() ?? 'N/A'}</p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Output Tokens</p>
                  <p className="text-sm">{metrics.costMetrics?.totalOutputTokens?.toLocaleString() ?? 'N/A'}</p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Cost per Request</p>
                  <p className="text-sm">
                    {metrics.totalRequests && metrics.costMetrics?.totalCost 
                      ? formatCurrency(
                          (typeof metrics.costMetrics.totalCost === 'string' 
                            ? parseFloat(metrics.costMetrics.totalCost) 
                            : metrics.costMetrics.totalCost) / metrics.totalRequests, 
                          model.spec.costTracking?.currency
                        )
                      : 'N/A'
                    }
                  </p>
                </div>
              </>
            ) : (
              <p className="text-sm text-muted-foreground">No cost metrics available</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Regional Health (if available) */}
      {metrics?.regionHealth && (
        <Card>
          <CardHeader>
            <CardTitle>Regional Endpoint Health</CardTitle>
            <CardDescription>Health status across different regions</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {Object.entries(metrics.regionHealth).map(([region, health]) => (
                <div key={region} className="flex items-center justify-between p-3 border rounded-lg">
                  <div className="flex items-center space-x-2">
                    <Globe className="h-4 w-4 text-blue-500" />
                    <span className="text-sm font-medium">{region}</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    {health ? (
                      <>
                        <CheckCircle className="h-4 w-4 text-green-500" />
                        <Badge variant="default" className="bg-green-100 text-green-800 text-xs">
                          Healthy
                        </Badge>
                      </>
                    ) : (
                      <>
                        <AlertCircle className="h-4 w-4 text-red-500" />
                        <Badge variant="destructive" className="text-xs">
                          Unhealthy
                        </Badge>
                      </>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default function ClusterModelDetailPage() {
  const params = useParams()
  const clusterName = params.name as string
  const modelName = params.modelName as string
  const [activeTab, setActiveTab] = useState('overview')
  const [yamlModalOpen, setYamlModalOpen] = useState(false)
  const [yamlContent, setYamlContent] = useState('')
  const [yamlLoading, setYamlLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  
  const { data: modelResponse, isLoading, error } = useModel(modelName)
  const deleteModel = useDeleteModel()

  const model = modelResponse?.data

  const getStatusIcon = (model: LanguageModel) => {
    if (model.status?.healthy === true) {
      return <CheckCircle className="h-5 w-5 text-green-500" />
    } else if (model.status?.healthy === false) {
      return <AlertCircle className="h-5 w-5 text-red-500" />
    } else {
      return <Clock className="h-5 w-5 text-yellow-500" />
    }
  }

  const getStatusColor = (model: LanguageModel) => {
    if (model.status?.healthy === true) {
      return 'bg-green-100 text-green-800'
    } else if (model.status?.healthy === false) {
      return 'bg-red-100 text-red-800'
    } else {
      return 'bg-gray-100 text-gray-800'
    }
  }

  const handleDeleteModel = async () => {
    if (!model || !model.metadata.name) return
    
    if (confirm(`Are you sure you want to delete model "${model.metadata.name}"?`)) {
      try {
        await deleteModel.mutateAsync(model.metadata.name)
        // Redirect to cluster models list after successful deletion
        window.location.href = `/clusters/${clusterName}/models`
      } catch (error) {
        console.error('Failed to delete model:', error)
        alert('Failed to delete model. Please try again.')
      }
    }
  }

  const handleViewYaml = async () => {
    setYamlModalOpen(true)
    setYamlLoading(true)
    try {
      const response = await fetch(`/api/clusters/${clusterName}/models/${modelName}/yaml`)
      if (!response.ok) {
        throw new Error('Failed to fetch YAML')
      }
      const yaml = await response.text()
      setYamlContent(yaml)
    } catch (error) {
      console.error('Error fetching YAML:', error)
      setYamlContent('Error loading YAML content')
    } finally {
      setYamlLoading(false)
    }
  }

  const handleCopyYaml = async () => {
    try {
      await navigator.clipboard.writeText(yamlContent)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy YAML:', error)
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

  if (error || !model) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Model not found</h3>
            <p className="text-muted-foreground mb-4">
              The model "{modelName}" could not be found in cluster "{clusterName}".
            </p>
            <Link href={`/clusters/${clusterName}/models`}>
              <Button variant="outline">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Models
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
            <Link href={`/clusters/${clusterName}/models`}>
              <Button variant="outline" size="icon">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <Cpu className="h-8 w-8 text-blue-500" />
            <div>
              <div>
                <h1 className="text-3xl font-bold">{model.metadata.name}</h1>
              </div>
              <p className="text-muted-foreground">
                LanguageModel
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Link href={`/clusters/${clusterName}/models/${model.metadata.name}/edit`}>
              <Button variant="outline">
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </Button>
            </Link>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={handleViewYaml}>
                  <FileCode className="h-4 w-4 mr-2" />
                  View YAML
                </DropdownMenuItem>
                <DropdownMenuItem 
                  onClick={handleDeleteModel}
                  disabled={deleteModel.isPending}
                  className="text-red-600 focus:text-red-600 focus:bg-red-50"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  {deleteModel.isPending ? 'Deleting...' : 'Delete Model'}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="metrics">Metrics</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            <ModelOverview model={model} />
          </TabsContent>

          <TabsContent value="metrics" className="space-y-6">
            <ModelMetrics model={model} />
          </TabsContent>
        </Tabs>
      </div>

      {/* YAML Modal */}
      <Dialog open={yamlModalOpen} onOpenChange={setYamlModalOpen}>
        <DialogContent className="w-[85vw] !max-w-[85vw] max-h-[85vh] flex flex-col sm:!max-w-[85vw] md:!max-w-[85vw] lg:!max-w-[85vw]">
          <DialogHeader>
            <DialogTitle>LanguageModel YAML</DialogTitle>
          </DialogHeader>
          
          <div className="flex-1 min-h-0 flex flex-col">
            {yamlLoading ? (
              <div className="flex items-center justify-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></div>
              </div>
            ) : (
              <div className="border rounded-lg flex-1 min-h-0 flex flex-col">
                <div className="bg-muted p-3 border-b flex justify-between items-center">
                  <span className="font-medium text-sm">
                    {model?.metadata.name || modelName}.yaml
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopyYaml}
                    disabled={!yamlContent || yamlContent.startsWith('Error')}
                  >
                    {copied ? (
                      <>
                        <Check className="h-4 w-4 mr-2" />
                        Copied!
                      </>
                    ) : (
                      <>
                        <Copy className="h-4 w-4 mr-2" />
                        Copy
                      </>
                    )}
                  </Button>
                </div>
                <div className="flex-1 min-h-0 overflow-hidden">
                  {yamlContent.startsWith('Error') ? (
                    <div className="p-4 text-red-600 font-mono text-sm">
                      {yamlContent}
                    </div>
                  ) : (
                    <SyntaxHighlighter
                      language="yaml"
                      style={oneLight}
                      customStyle={{
                        margin: 0,
                        padding: '1rem',
                        background: 'transparent',
                        fontSize: '0.875rem',
                        lineHeight: '1.5',
                        height: '100%',
                        overflow: 'auto'
                      }}
                      codeTagProps={{
                        style: {
                          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
                        }
                      }}
                    >
                      {yamlContent}
                    </SyntaxHighlighter>
                  )}
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </AuthenticatedLayout>
  )
}