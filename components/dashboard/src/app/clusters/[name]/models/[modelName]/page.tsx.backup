'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { fetchWithOrganization } from '@/lib/api-client'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ResourceHeader } from '@/components/ui/resource-header'
import { ResourceEventsActivity } from '@/components/ui/events-activity'
import { 
  Cpu, AlertCircle, CheckCircle, Clock, ArrowLeft, 
  Edit, FileText, Trash2, Activity, Zap, DollarSign, 
  TrendingUp, Globe, Shield, Key, Settings, MoreVertical, FileCode, Copy, Check,
  Home, BarChart3, Info, RefreshCw, Database
} from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight, oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useTheme } from 'next-themes'
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
  clusterName: string
}

function ModelOverview({ model, clusterName }: ModelOverviewProps) {
  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Provider</p>
              <Badge variant="outline">{model.spec.provider}</Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Model Name</p>
              <p className="text-sm font-mono">{model.spec.modelName}</p>
            </div>
            {model.spec.endpoint && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Endpoint</p>
                <p className="text-sm font-mono text-xs break-all">{model.spec.endpoint}</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Status & Conditions */}
      <Card>
        <CardHeader>
          <CardTitle>Status & Conditions</CardTitle>
        </CardHeader>
        <CardContent>
          {model.status?.conditions && model.status.conditions.length > 0 ? (
            <div className="space-y-3">
              {model.status.conditions.map((condition, index) => (
                <div key={index} className="flex items-center justify-between p-3 border border-stone-200 dark:border-stone-700">
                  <div>
                    <p className="text-sm font-medium">{condition.type}</p>
                    {condition.message && (
                      <p className="text-xs text-stone-600 dark:text-stone-400">{condition.message}</p>
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
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-stone-600 dark:text-stone-400">No status conditions available</p>
          )}
        </CardContent>
      </Card>

      {/* Events */}
      <ResourceEventsActivity
        resourceType="model"
        resourceName={model.metadata.name!}
        namespace={model.metadata.namespace!}
        clusterName={clusterName}
      />

      {/* Cost Tracking */}
      {model.spec.costTracking?.enabled && (
        <Card>
          <CardHeader>
            <CardTitle>Cost Tracking Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Currency</p>
                <p className="text-sm">{model.spec.costTracking.currency || 'USD'}</p>
              </div>
              {model.spec.costTracking.inputTokenCost && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Input Token Cost</p>
                  <p className="text-sm">{formatCurrency(model.spec.costTracking.inputTokenCost, model.spec.costTracking.currency)}</p>
                </div>
              )}
              {model.spec.costTracking.outputTokenCost && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Output Token Cost</p>
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
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Requests per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.requestsPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.tokensPerMinute && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Tokens per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.tokensPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.concurrentRequests && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Concurrent Requests</p>
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

interface ModelDetailsProps {
  model: LanguageModel
}

function ModelDetails({ model }: ModelDetailsProps) {
  return (
    <div className="space-y-6">
      {/* Model Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            Model Configuration
          </CardTitle>
          <CardDescription>
            LLM parameters and behavior settings
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Temperature</p>
              <p className="text-sm">{model.spec.configuration?.temperature ?? 0.7}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Max Tokens</p>
              <p className="text-sm">{(model.spec.configuration?.maxTokens ?? 4096).toLocaleString()}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Top P</p>
              <p className="text-sm">{model.spec.configuration?.topP ?? 1.0}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Context Window</p>
              <p className="text-sm">{((model.spec.configuration as any)?.contextWindow ?? 8192).toLocaleString()} tokens</p>
            </div>
            {model.spec.configuration?.frequencyPenalty !== undefined && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Frequency Penalty</p>
                <p className="text-sm">{model.spec.configuration.frequencyPenalty}</p>
              </div>
            )}
            {model.spec.configuration?.presencePenalty !== undefined && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Presence Penalty</p>
                <p className="text-sm">{model.spec.configuration.presencePenalty}</p>
              </div>
            )}
            {(model.spec as any).timeout && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Request Timeout</p>
                <p className="text-sm">{(model.spec as any).timeout}</p>
              </div>
            )}
          </div>
          {model.spec.configuration?.stopSequences && model.spec.configuration.stopSequences.length > 0 && (
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Stop Sequences</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {model.spec.configuration.stopSequences.map((seq, index) => (
                  <Badge key={index} variant="outline" className="text-xs font-mono">
                    {seq}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Caching */}
      {model.spec.caching && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              Response Caching
            </CardTitle>
            <CardDescription>
              Caching configuration for improved performance
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Status</p>
                <Badge variant={model.spec.caching.enabled ? "default" : "secondary"}>
                  {model.spec.caching.enabled ? "Enabled" : "Disabled"}
                </Badge>
              </div>
              {model.spec.caching.ttl && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">TTL (Time to Live)</p>
                  <p className="text-sm">{model.spec.caching.ttl}</p>
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
            <CardTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Rate Limiting
            </CardTitle>
            <CardDescription>
              Request rate limits and throttling
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {model.spec.rateLimits.requestsPerMinute && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Requests per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.requestsPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.tokensPerMinute && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Tokens per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.tokensPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.concurrentRequests && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Concurrent Requests</p>
                  <p className="text-sm">{model.spec.rateLimits.concurrentRequests}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Retry Policy */}
      {model.spec.retryPolicy && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <RefreshCw className="h-5 w-5" />
              Retry Policy
            </CardTitle>
            <CardDescription>
              Automatic retry configuration for failed requests
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {(model.spec.retryPolicy as any).maxAttempts && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Max Attempts</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).maxAttempts}</p>
                </div>
              )}
              {(model.spec.retryPolicy as any).initialBackoff && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Initial Backoff</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).initialBackoff}</p>
                </div>
              )}
              {(model.spec.retryPolicy as any).maxBackoff && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Max Backoff</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).maxBackoff}</p>
                </div>
              )}
              {(model.spec.retryPolicy as any).backoffMultiplier && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Backoff Multiplier</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).backoffMultiplier}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Network Policy */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Network Policy
          </CardTitle>
          <CardDescription>
            External network access rules
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {model.spec.egress && model.spec.egress.length > 0 ? (
            <div className="space-y-3">
              {model.spec.egress.map((rule, index) => (
                <div key={index} className="border border-stone-200 p-3 dark:border-stone-700">
                  <div className="space-y-2">
                    {rule.description && (
                      <div>
                        <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Description</p>
                        <p className="text-sm">{rule.description}</p>
                      </div>
                    )}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                      {rule.to?.dns && rule.to.dns.length > 0 && (
                        <div>
                          <p className="text-sm font-medium text-stone-600 dark:text-stone-400">DNS Names</p>
                          <p className="text-sm font-mono">{rule.to.dns.join(', ')}</p>
                        </div>
                      )}
                      {rule.ports && rule.ports.length > 0 && (
                        <div>
                          <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Ports</p>
                          <p className="text-sm font-mono">{rule.ports.map(p => p.port).join(', ')}</p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-6">
              <Globe className="h-8 w-8 text-stone-500 dark:text-stone-400 mx-auto mb-2" />
              <p className="text-sm text-stone-600 dark:text-stone-400">No network egress rules configured</p>
              <p className="text-xs text-stone-500 dark:text-stone-500 mt-1">
                Network policies control external access from the model
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

interface ModelMetricsProps {
  model: LanguageModel
}

function ModelMetrics({ model }: ModelMetricsProps) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[400px] space-y-4">
      <div className="mx-auto w-24 h-24 bg-stone-100 border border-stone-200 flex items-center justify-center dark:bg-stone-800 dark:border-stone-700">
        <BarChart3 className="h-12 w-12 text-stone-500 dark:text-stone-400" />
      </div>
      <h3 className="text-xl font-semibold text-stone-600 dark:text-stone-400">Coming Soon</h3>
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
  const { theme } = useTheme()
  
  const { data: modelResponse, isLoading, error } = useModel(modelName, clusterName)
  const deleteModel = useDeleteModel(clusterName)

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
      return 'bg-stone-100 text-stone-800 dark:bg-stone-800 dark:text-stone-300'
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
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/models/${modelName}/yaml`)
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
            <p className="text-stone-600 dark:text-stone-400 mb-4">
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
        <ResourceHeader
          backHref={`/clusters/${clusterName}/models`}
          backLabel="Back to Models"
          icon={Cpu}
          title={model.metadata.name}
          subtitle="LanguageModel"
          actions={
            <>
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
            </>
          }
        />

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col">
          <TabsList>
            <TabsTrigger value="overview">
              <Home className="w-4 h-4 mr-2" />
              Overview
            </TabsTrigger>
            <TabsTrigger value="details">
              <Info className="w-4 h-4 mr-2" />
              Details
            </TabsTrigger>
            <TabsTrigger value="metrics">
              <BarChart3 className="w-4 h-4 mr-2" />
              Metrics
            </TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6 mt-3">
            <ModelOverview model={model} clusterName={clusterName} />
          </TabsContent>

          <TabsContent value="details" className="space-y-6 mt-3">
            <ModelDetails model={model} />
          </TabsContent>

          <TabsContent value="metrics" className="space-y-6 mt-3">
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
                <div className="flex-1 min-h-0 overflow-auto">
                  {yamlContent.startsWith('Error') ? (
                    <div className="p-4 text-red-600 font-mono text-sm">
                      {yamlContent}
                    </div>
                  ) : (
                    <SyntaxHighlighter
                      language="yaml"
                      style={theme === 'dark' ? oneDark : oneLight}
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