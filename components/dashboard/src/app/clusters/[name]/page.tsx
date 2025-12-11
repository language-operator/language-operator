'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  ArrowLeft, Server, Globe, Shield, Network, Users, 
  CheckCircle, AlertCircle, Clock, Activity, Edit,
  Trash2, ExternalLink, Copy, Settings, Database,
  Terminal, FileText
} from 'lucide-react'
import { useCluster, useDeleteCluster } from '@/hooks/use-clusters'
import { Skeleton } from '@/components/ui/skeleton'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'

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

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}

export default function ClusterDetailPage() {
  const params = useParams()
  const router = useRouter()
  const clusterName = params?.name as string
  
  const { data: cluster, isLoading, error, refetch } = useCluster(clusterName)
  const deleteCluster = useDeleteCluster()

  const handleDelete = async () => {
    if (!cluster?.metadata.name) return
    
    if (confirm(`Are you sure you want to delete cluster "${cluster.metadata.name}"? This action cannot be undone.`)) {
      try {
        await deleteCluster.mutateAsync(cluster.metadata.name)
        router.push('/clusters')
      } catch (error) {
        console.error('Failed to delete cluster:', error)
        alert('Failed to delete cluster. Please try again.')
      }
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Skeleton className="h-8 w-8" />
              <Skeleton className="h-8 w-64" />
            </div>
            <Skeleton className="h-9 w-32" />
          </div>
          <div className="grid gap-6 md:grid-cols-3">
            <div className="md:col-span-2 space-y-6">
              <Skeleton className="h-64 w-full" />
              <Skeleton className="h-96 w-full" />
            </div>
            <div className="space-y-6">
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-32 w-full" />
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error || !cluster) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Cluster not found</h3>
            <p className="text-muted-foreground mb-4">
              The cluster "{clusterName}" could not be found or you don't have permission to view it.
            </p>
            <Link href="/clusters">
              <Button>
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Clusters
              </Button>
            </Link>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  const getStatusIcon = () => {
    const phase = cluster.status?.phase
    if (phase === 'Ready') {
      return <CheckCircle className="h-5 w-5 text-green-500" />
    } else if (phase === 'Pending') {
      return <Clock className="h-5 w-5 text-yellow-500" />
    } else if (phase === 'Failed') {
      return <AlertCircle className="h-5 w-5 text-red-500" />
    } else {
      return <AlertCircle className="h-5 w-5 text-gray-500" />
    }
  }

  const getStatusBadge = () => {
    const phase = cluster.status?.phase || 'Unknown'
    if (phase === 'Ready') {
      return <Badge variant="default" className="bg-green-100 text-green-800">Ready</Badge>
    } else if (phase === 'Pending') {
      return <Badge variant="secondary">Pending</Badge>
    } else if (phase === 'Failed') {
      return <Badge variant="destructive">Failed</Badge>
    } else {
      return <Badge variant="secondary">{phase}</Badge>
    }
  }

  const getIngressStatusBadge = () => {
    const ingressReady = cluster.status?.ingress?.ready
    if (ingressReady === true) {
      return <Badge variant="default" className="bg-green-100 text-green-800">Healthy</Badge>
    } else if (ingressReady === false) {
      return <Badge variant="destructive">Failed</Badge>
    } else {
      return <Badge variant="secondary">Configuring</Badge>
    }
  }

  const getTLSStatusBadge = () => {
    const tlsEnabled = cluster.spec.tls?.enabled
    const tlsReady = cluster.status?.tls?.ready
    
    if (!tlsEnabled) {
      return <Badge variant="outline">Disabled</Badge>
    } else if (tlsReady === true) {
      return <Badge variant="default" className="bg-green-100 text-green-800">Valid Certificate</Badge>
    } else if (tlsReady === false) {
      return <Badge variant="destructive">Certificate Error</Badge>
    } else {
      return <Badge variant="secondary">Provisioning</Badge>
    }
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Link href="/clusters">
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back
              </Button>
            </Link>
            <div className="flex items-center space-x-3">
              <Server className="h-8 w-8 text-orange-500" />
              <div>
                <h1 className="text-3xl font-bold">{cluster.metadata.name}</h1>
                <div className="flex items-center space-x-2 mt-1">
                  {getStatusIcon()}
                  {getStatusBadge()}
                </div>
              </div>
            </div>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button>
                <Settings className="h-4 w-4 mr-2" />
                Actions
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem asChild>
                <Link href={`/clusters/${cluster.metadata.name}/edit`}>
                  <Edit className="h-4 w-4 mr-2" />
                  Edit Cluster
                </Link>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive"
                onClick={handleDelete}
                disabled={deleteCluster.isPending}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete Cluster
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          {/* Main Content */}
          <div className="md:col-span-2 space-y-6">
            {/* Overview Cards */}
            <div className="grid gap-4 md:grid-cols-3">
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Domain</CardTitle>
                  <Globe className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-mono">
                      {cluster.spec.domain || 'Not configured'}
                    </span>
                    {cluster.spec.domain && (
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => copyToClipboard(cluster.spec.domain!)}
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Agents</CardTitle>
                  <Users className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {cluster.status?.agentCount || 0}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Deployed agents
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Ingress</CardTitle>
                  <Network className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="flex items-center space-x-2">
                    {getIngressStatusBadge()}
                  </div>
                  {cluster.status?.ingress?.endpoint && (
                    <p className="text-xs text-muted-foreground mt-1">
                      {cluster.status.ingress.endpoint}
                    </p>
                  )}
                </CardContent>
              </Card>
            </div>

            {/* Detailed Information */}
            <Tabs defaultValue="config" className="w-full">
              <TabsList>
                <TabsTrigger value="config">Configuration</TabsTrigger>
                <TabsTrigger value="agents">Member Agents</TabsTrigger>
                <TabsTrigger value="status">Status & Events</TabsTrigger>
              </TabsList>

              <TabsContent value="config" className="space-y-4">
                {/* Domain & DNS Configuration */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Globe className="h-5 w-5" />
                      <span>Domain & DNS Configuration</span>
                    </CardTitle>
                    <CardDescription>
                      Network and domain settings for the cluster
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Domain</label>
                        <div className="flex items-center space-x-2 mt-1">
                          <span className="text-sm font-mono bg-muted px-2 py-1 rounded">
                            {cluster.spec.domain || 'Not configured'}
                          </span>
                          {cluster.spec.domain && (
                            <>
                              <Button 
                                variant="ghost" 
                                size="sm"
                                onClick={() => copyToClipboard(cluster.spec.domain!)}
                              >
                                <Copy className="h-3 w-3" />
                              </Button>
                              <Button 
                                variant="ghost" 
                                size="sm" 
                                asChild
                              >
                                <a href={`https://${cluster.spec.domain}`} target="_blank" rel="noopener noreferrer">
                                  <ExternalLink className="h-3 w-3" />
                                </a>
                              </Button>
                            </>
                          )}
                        </div>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Subdomain</label>
                        <p className="text-sm font-mono bg-muted px-2 py-1 rounded mt-1">
                          {cluster.spec.subdomain || 'Not configured'}
                        </p>
                      </div>
                    </div>
                    
                    {cluster.status?.ingress?.dnsRecords && cluster.status.ingress.dnsRecords.length > 0 && (
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">DNS Records</label>
                        <div className="mt-1 space-y-1">
                          {cluster.status.ingress.dnsRecords.map((record, index) => (
                            <div key={index} className="text-sm font-mono bg-muted px-2 py-1 rounded">
                              {record.type}: {record.name} → {record.value}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* TLS Configuration */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Shield className="h-5 w-5" />
                      <span>TLS Configuration</span>
                    </CardTitle>
                    <CardDescription>
                      SSL/TLS certificate and encryption settings
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium">TLS Status</p>
                        <p className="text-xs text-muted-foreground">Certificate status and configuration</p>
                      </div>
                      {getTLSStatusBadge()}
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Enabled</label>
                        <p className="text-sm mt-1">
                          {cluster.spec.tls?.enabled ? 'Yes' : 'No'}
                        </p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Auto-provision</label>
                        <p className="text-sm mt-1">
                          {cluster.spec.tls?.autoProvision ? 'Yes' : 'No'}
                        </p>
                      </div>
                    </div>

                    {cluster.spec.tls?.secretName && (
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Certificate Secret</label>
                        <p className="text-sm font-mono bg-muted px-2 py-1 rounded mt-1">
                          {cluster.spec.tls.secretName}
                        </p>
                      </div>
                    )}

                    {cluster.status?.tls?.certificateExpiry && (
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Certificate Expires</label>
                        <p className="text-sm mt-1">
                          {new Date(cluster.status.tls.certificateExpiry).toLocaleString()}
                        </p>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* Gateway Configuration */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Network className="h-5 w-5" />
                      <span>Gateway Configuration</span>
                    </CardTitle>
                    <CardDescription>
                      Load balancer and ingress gateway settings
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Gateway Class</label>
                        <p className="text-sm font-mono bg-muted px-2 py-1 rounded mt-1">
                          {cluster.spec.gateway?.className || 'Default'}
                        </p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Load Balancer Type</label>
                        <p className="text-sm font-mono bg-muted px-2 py-1 rounded mt-1">
                          {cluster.spec.gateway?.loadBalancerType || 'Standard'}
                        </p>
                      </div>
                    </div>

                    {cluster.status?.gateway?.externalIP && (
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">External IP</label>
                        <div className="flex items-center space-x-2 mt-1">
                          <span className="text-sm font-mono bg-muted px-2 py-1 rounded">
                            {cluster.status.gateway.externalIP}
                          </span>
                          <Button 
                            variant="ghost" 
                            size="sm"
                            onClick={() => copyToClipboard(cluster.status.gateway.externalIP!)}
                          >
                            <Copy className="h-3 w-3" />
                          </Button>
                        </div>
                      </div>
                    )}

                    {cluster.spec.gateway?.annotations && Object.keys(cluster.spec.gateway.annotations).length > 0 && (
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Gateway Annotations</label>
                        <div className="mt-1 space-y-1">
                          {Object.entries(cluster.spec.gateway.annotations).map(([key, value]) => (
                            <div key={key} className="text-xs font-mono bg-muted px-2 py-1 rounded">
                              {key}: {value}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="agents" className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Users className="h-5 w-5" />
                      <span>Member Agents ({cluster.status?.agentCount || 0})</span>
                    </CardTitle>
                    <CardDescription>
                      Language agents deployed in this cluster
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    {cluster.status?.agents && cluster.status.agents.length > 0 ? (
                      <div className="space-y-3">
                        {cluster.status.agents.map((agent, index) => (
                          <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                            <div className="flex items-center space-x-3">
                              <Users className="h-4 w-4 text-blue-500" />
                              <div>
                                <p className="text-sm font-medium">{agent.name}</p>
                                <p className="text-xs text-muted-foreground">
                                  Namespace: {agent.namespace}
                                </p>
                              </div>
                            </div>
                            <div className="flex items-center space-x-2">
                              {agent.status === 'Ready' ? (
                                <Badge variant="default" className="bg-green-100 text-green-800">Ready</Badge>
                              ) : agent.status === 'Pending' ? (
                                <Badge variant="secondary">Pending</Badge>
                              ) : (
                                <Badge variant="destructive">Failed</Badge>
                              )}
                              <Link href={`/agents/${agent.name}`}>
                                <Button variant="ghost" size="sm">
                                  <ExternalLink className="h-3 w-3" />
                                </Button>
                              </Link>
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="text-center py-8">
                        <Users className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                        <h3 className="text-lg font-medium mb-2">No agents deployed</h3>
                        <p className="text-muted-foreground mb-4">
                          This cluster doesn't have any language agents deployed yet.
                        </p>
                        <Link href="/agents/new">
                          <Button>Deploy Agent</Button>
                        </Link>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="status" className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Activity className="h-5 w-5" />
                      <span>Status Information</span>
                    </CardTitle>
                    <CardDescription>
                      Current status and recent events
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Phase</label>
                        <div className="flex items-center space-x-2 mt-1">
                          {getStatusIcon()}
                          {getStatusBadge()}
                        </div>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Last Updated</label>
                        <p className="text-sm mt-1">
                          {formatTimeAgo(cluster.status?.lastUpdateTime || cluster.metadata.creationTimestamp)}
                        </p>
                      </div>
                    </div>

                    {cluster.status?.conditions && cluster.status.conditions.length > 0 && (
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Conditions</label>
                        <div className="mt-1 space-y-2">
                          {cluster.status.conditions.map((condition, index) => (
                            <div key={index} className="flex items-center justify-between p-2 border rounded">
                              <div className="flex items-center space-x-2">
                                {condition.status === 'True' ? (
                                  <CheckCircle className="h-4 w-4 text-green-500" />
                                ) : condition.status === 'False' ? (
                                  <AlertCircle className="h-4 w-4 text-red-500" />
                                ) : (
                                  <Clock className="h-4 w-4 text-yellow-500" />
                                )}
                                <span className="text-sm font-medium">{condition.type}</span>
                              </div>
                              <div className="text-right">
                                <Badge variant={condition.status === 'True' ? 'default' : condition.status === 'False' ? 'destructive' : 'secondary'}>
                                  {condition.status}
                                </Badge>
                                {condition.reason && (
                                  <p className="text-xs text-muted-foreground mt-1">
                                    {condition.reason}
                                  </p>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {cluster.status?.message && (
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Status Message</label>
                        <p className="text-sm bg-muted p-3 rounded mt-1">
                          {cluster.status.message}
                        </p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Metadata */}
            <Card>
              <CardHeader>
                <CardTitle>Metadata</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Name</label>
                  <p className="text-sm font-mono">{cluster.metadata.name}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Namespace</label>
                  <p className="text-sm font-mono">{cluster.metadata.namespace}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Created</label>
                  <p className="text-sm">{formatTimeAgo(cluster.metadata.creationTimestamp)}</p>
                </div>
                {cluster.metadata.uid && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">UID</label>
                    <div className="flex items-center space-x-2">
                      <p className="text-xs font-mono truncate">{cluster.metadata.uid}</p>
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => copyToClipboard(cluster.metadata.uid!)}
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Labels & Annotations */}
            {((cluster.metadata.labels && Object.keys(cluster.metadata.labels).length > 0) ||
              (cluster.metadata.annotations && Object.keys(cluster.metadata.annotations).length > 0)) && (
              <Card>
                <CardHeader>
                  <CardTitle>Labels & Annotations</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {cluster.metadata.labels && Object.keys(cluster.metadata.labels).length > 0 && (
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Labels</label>
                      <div className="mt-1 space-y-1">
                        {Object.entries(cluster.metadata.labels).map(([key, value]) => (
                          <div key={key} className="text-xs font-mono bg-muted px-2 py-1 rounded">
                            {key}: {value}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                  {cluster.metadata.annotations && Object.keys(cluster.metadata.annotations).length > 0 && (
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Annotations</label>
                      <div className="mt-1 space-y-1">
                        {Object.entries(cluster.metadata.annotations).map(([key, value]) => (
                          <div key={key} className="text-xs font-mono bg-muted px-2 py-1 rounded">
                            {key}: {value}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}