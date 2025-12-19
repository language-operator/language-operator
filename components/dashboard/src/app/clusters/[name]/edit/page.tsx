'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { X, Plus, Globe, Cloud, Zap } from 'lucide-react'
import { ClusterForm, ClusterFormData } from '@/components/forms/cluster-form'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ArrowLeft, Server, Settings, Network } from 'lucide-react'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchWithOrganization } from '@/lib/api-client'
import Link from 'next/link'

interface Cluster {
  metadata: {
    name: string
    namespace: string
    creationTimestamp: string
  }
  spec: {
    domain?: string
    description?: string
    ingress?: {
      enabled: boolean
    }
    networkPolicies?: {
      enabled: boolean
      allowedDomains?: string[]
    }
  }
  status: {
    phase: string
  }
}

export default function EditClusterPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter()
  const [cluster, setCluster] = useState<Cluster | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingCluster, setIsLoadingCluster] = useState(true)
  const [error, setError] = useState('')
  const [clusterName, setClusterName] = useState<string>('')
  const [activeTab, setActiveTab] = useState('general')
  const [allowedDomains, setAllowedDomains] = useState<string[]>([])
  const [newDomain, setNewDomain] = useState('')

  // Get the cluster name from params
  useEffect(() => {
    const getClusterName = async () => {
      const resolvedParams = await params
      setClusterName(resolvedParams.name)
    }
    getClusterName()
  }, [params])

  // Fetch existing cluster data
  useEffect(() => {
    if (!clusterName) return

    const fetchCluster = async () => {
      setIsLoadingCluster(true)
      try {
        const response = await fetchWithOrganization(`/api/clusters/${clusterName}`)
        if (!response.ok) {
          throw new Error('Failed to fetch cluster')
        }
        const data = await response.json()
        setCluster(data.cluster)
        
        // Initialize allowed domains from cluster data
        const domains = data.cluster?.spec?.networkPolicies?.allowedDomains || []
        setAllowedDomains(domains)
      } catch (err: any) {
        console.error('Error fetching cluster:', err)
        setError(err.message || 'Failed to load cluster')
      } finally {
        setIsLoadingCluster(false)
      }
    }

    fetchCluster()
  }, [clusterName])

  const handleAddDomain = () => {
    if (!newDomain.trim()) return
    
    // Validate domain format
    const domainRegex = /^[a-zA-Z0-9*]([a-zA-Z0-9\-.*]{0,61}[a-zA-Z0-9*])?(\.[a-zA-Z0-9*]([a-zA-Z0-9\-.*]{0,61}[a-zA-Z0-9*])?)*$/
    if (!domainRegex.test(newDomain.trim())) {
      setError('Invalid domain format')
      return
    }
    
    // Check for duplicates
    if (allowedDomains.includes(newDomain.trim())) {
      setError('Domain already exists')
      return
    }
    
    setAllowedDomains(prev => [...prev, newDomain.trim()])
    setNewDomain('')
    setError('')
  }

  const handleRemoveDomain = (domain: string) => {
    setAllowedDomains(prev => prev.filter(d => d !== domain))
  }

  const handleAddQuickDomain = (domain: string) => {
    if (!allowedDomains.includes(domain)) {
      setAllowedDomains(prev => [...prev, domain])
    }
  }

  const handleSubmit = async (formData: ClusterFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          domain: formData.domain || undefined,
          spec: {
            domain: formData.domain || undefined,
            ingress: {
              enabled: formData.enableTLS, // Use enableTLS as a proxy for ingress
            },
            networkPolicies: {
              enabled: true, // Default to enabled
              allowedDomains: allowedDomains,
            },
          },
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update cluster')
      }

      // Redirect to cluster details page
      router.push(`/clusters/${clusterName}`)
    } catch (err: any) {
      console.error('Error updating cluster:', err)
      setError(err.message || 'Failed to update cluster')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}`)
  }

  // Convert cluster data to form format
  const getInitialFormData = (): Partial<ClusterFormData> | undefined => {
    if (!cluster) return undefined

    return {
      name: cluster.metadata.name,
      domain: cluster.spec.domain || '',
      enableTLS: cluster.spec.ingress?.enabled ?? true,
    }
  }

  if (isLoadingCluster) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          {/* Header Skeleton */}
          <div className="flex items-center space-x-4">
            <Skeleton className="h-8 w-32" />
            <div>
              <Skeleton className="h-8 w-64 mb-2" />
              <Skeleton className="h-4 w-48" />
            </div>
          </div>

          {/* Form Skeleton */}
          <div className="max-w-2xl space-y-6">
            <Skeleton className="h-48" />
            <Skeleton className="h-32" />
            <Skeleton className="h-24" />
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error && !cluster) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <ResourceHeader
            backHref="/clusters"
            backLabel="Back to Clusters"
            icon={Server}
            title="Edit Cluster"
            subtitle="Failed to load cluster"
          />

          <div className="max-w-2xl">
            <div className="text-center py-12">
              <h3 className="text-lg font-medium mb-2">Error loading cluster</h3>
              <p className="text-muted-foreground mb-4">{error}</p>
              <Link href="/clusters">
                <Button>Back to Clusters</Button>
              </Link>
            </div>
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
          backHref={`/clusters/${clusterName}`}
          backLabel="Back to Cluster"
          icon={Server}
          title="Edit Cluster"
          subtitle={`Update settings for cluster "${clusterName}"`}
        />

        {/* Tabbed Form */}
        <div className="max-w-4xl">
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="general">
                <Settings className="h-4 w-4 mr-2" />
                General
              </TabsTrigger>
              <TabsTrigger value="network">
                <Network className="h-4 w-4 mr-2" />
                Network
              </TabsTrigger>
            </TabsList>

            {/* General Configuration */}
            <TabsContent value="general" className="space-y-6 mt-6">
              <ClusterForm
                initialData={getInitialFormData()}
                isLoading={isLoading}
                error={activeTab === 'general' ? error : ''}
                onSubmit={handleSubmit}
                onCancel={handleCancel}
                isEdit={true}
              />
            </TabsContent>

            {/* Network Configuration */}
            <TabsContent value="network" className="space-y-6 mt-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <Globe className="h-5 w-5" />
                    <span>Allowed Domains</span>
                  </CardTitle>
                  <CardDescription>
                    Configure domains that agents in this cluster can access. This creates a NetworkPolicy to allow egress traffic to these domains.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  {/* Current Domains */}
                  {allowedDomains.length > 0 && (
                    <div className="space-y-3">
                      <Label>Current Domains</Label>
                      <div className="flex flex-wrap gap-2">
                        {allowedDomains.map((domain) => (
                          <Badge key={domain} variant="outline" className="flex items-center gap-2">
                            {domain}
                            <Button
                              type="button"
                              size="sm"
                              variant="ghost"
                              className="h-4 w-4 p-0 hover:bg-destructive hover:text-destructive-foreground"
                              onClick={() => handleRemoveDomain(domain)}
                              disabled={isLoading}
                            >
                              <X className="h-3 w-3" />
                            </Button>
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Add Domain */}
                  <div className="space-y-3">
                    <Label htmlFor="newDomain">Add Domain</Label>
                    <div className="flex gap-2">
                      <Input
                        id="newDomain"
                        value={newDomain}
                        onChange={(e) => setNewDomain(e.target.value)}
                        placeholder="*.company.com or api.openai.com"
                        className="flex-1"
                        disabled={isLoading}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault()
                            handleAddDomain()
                          }
                        }}
                      />
                      <Button
                        type="button"
                        onClick={handleAddDomain}
                        disabled={isLoading || !newDomain.trim()}
                      >
                        <Plus className="h-4 w-4" />
                      </Button>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      Use wildcards like *.company.com for subdomains. IPv4 CIDR blocks are also supported.
                    </p>
                  </div>

                  {/* Quick Add Buttons */}
                  <div className="space-y-3">
                    <Label>Quick Add Common Providers</Label>
                    <div className="flex flex-wrap gap-2">
                      {[
                        { name: 'OpenAI', domain: 'api.openai.com' },
                        { name: 'Anthropic', domain: 'api.anthropic.com' },
                        { name: 'Google AI', domain: '*.googleapis.com' },
                        { name: 'Azure OpenAI', domain: '*.openai.azure.com' },
                        { name: 'AWS Bedrock', domain: '*.amazonaws.com' },
                      ].map((provider) => (
                        <Button
                          key={provider.domain}
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => handleAddQuickDomain(provider.domain)}
                          disabled={isLoading || allowedDomains.includes(provider.domain)}
                        >
                          {provider.name}
                        </Button>
                      ))}
                    </div>
                  </div>

                  {/* System Access Info */}
                  <div className="rounded-lg border border-green-200 bg-green-50 p-4">
                    <div className="flex items-start space-x-3">
                      <Zap className="h-5 w-5 text-green-600 mt-0.5" />
                      <div className="space-y-1">
                        <p className="text-sm font-medium text-green-800">
                          System Access (Auto-managed)
                        </p>
                        <p className="text-sm text-green-700">
                          Agents automatically have access to the Kubernetes API server for event emission and system operations.
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Error Display */}
                  {activeTab === 'network' && error && (
                    <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                      <p className="text-sm text-red-700">{error}</p>
                    </div>
                  )}

                  {/* Network Tab Actions */}
                  <div className="flex justify-end space-x-4 pt-4">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleCancel}
                      disabled={isLoading}
                    >
                      Cancel
                    </Button>
                    <Button
                      type="button"
                      onClick={() => handleSubmit(getInitialFormData() as ClusterFormData)}
                      disabled={isLoading}
                    >
                      {isLoading ? 'Updating...' : 'Update Cluster'}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}