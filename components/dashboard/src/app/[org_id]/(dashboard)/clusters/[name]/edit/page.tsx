'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { useOrganization } from '@/components/organization-provider'
import { Button } from '@/components/ui/button'
import { ClusterForm, ClusterFormData } from '@/components/forms/cluster-form'
import { ResourceNetworkPolicyForm } from '@/components/forms/resource-network-policy-form'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Form } from '@/components/ui/form'
import { ArrowLeft, Server, Settings, Network } from 'lucide-react'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchWithOrganization } from '@/lib/api-client'
import Link from 'next/link'

interface NetworkRule {
  description?: string
  to?: {
    dns?: string[]
    cidr?: string
    group?: string
    service?: {
      name: string
      namespace?: string
    }
    namespaceSelector?: {
      matchLabels: Record<string, string>
    }
    podSelector?: {
      matchLabels: Record<string, string>
    }
  }
  from?: {
    dns?: string[]
    cidr?: string
    group?: string
    service?: {
      name: string
      namespace?: string
    }
    namespaceSelector?: {
      matchLabels: Record<string, string>
    }
    podSelector?: {
      matchLabels: Record<string, string>
    }
  }
  ports?: Array<{
    port: number
    protocol: 'TCP' | 'UDP' | 'SCTP'
  }>
}

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
    networkPolicies?: NetworkRule[]
  }
  status: {
    phase: string
  }
}

// Form validation schema for network tab (uses legacy structure for compatibility with ResourceNetworkPolicyForm)
const networkFormSchema = z.object({
  egressRules: z.array(z.object({
    description: z.string().optional(),
    dns: z.array(z.string()).optional(),
    cidr: z.string().optional(),
    ports: z.array(z.object({
      port: z.number().min(1).max(65535),
      protocol: z.enum(['TCP', 'UDP'])
    })).optional()
  })).optional(),
})

type NetworkFormValues = z.infer<typeof networkFormSchema>

export default function EditClusterPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter()
  const { getOrgUrl } = useOrganization()
  const [cluster, setCluster] = useState<Cluster | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingCluster, setIsLoadingCluster] = useState(true)
  const [error, setError] = useState('')
  const [clusterName, setClusterName] = useState<string>('')
  const [activeTab, setActiveTab] = useState('general')

  // Form for network policy management
  const networkForm = useForm<NetworkFormValues>({
    resolver: zodResolver(networkFormSchema),
    defaultValues: {
      egressRules: []
    }
  })

  const watchedEgressRules = networkForm.watch('egressRules')

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
        
        // Initialize network form with existing network policies converted to legacy format
        const networkPolicies = data.cluster?.spec?.networkPolicies || []
        const egressRules = networkPolicies.map((rule: NetworkRule) => ({
          description: rule.description,
          dns: rule.to?.dns || [],
          cidr: rule.to?.cidr || '',
          ports: rule.ports || []
        }))
        networkForm.reset({ egressRules })
      } catch (err: any) {
        console.error('Error fetching cluster:', err)
        setError(err.message || 'Failed to load cluster')
      } finally {
        setIsLoadingCluster(false)
      }
    }

    fetchCluster()
  }, [clusterName])


  const handleSubmit = async (formData: ClusterFormData) => {
    setIsLoading(true)
    setError('')

    try {
      // Get current network form values and transform to CRD format
      const networkValues = networkForm.getValues()
      const networkPolicies = (networkValues.egressRules || []).map(rule => ({
        description: rule.description,
        to: {
          dns: rule.dns && rule.dns.length > 0 ? rule.dns : undefined,
          cidr: rule.cidr || undefined
        },
        ports: rule.ports && rule.ports.length > 0 ? rule.ports.map(port => ({
          port: port.port,
          protocol: port.protocol || 'TCP'
        })) : undefined
      })).filter(rule => 
        // Only include rules that have at least one target defined
        (rule.to?.dns && rule.to.dns.length > 0) || rule.to?.cidr
      )
      
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
            networkPolicies: networkPolicies,
          },
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update cluster')
      }

      // Redirect to cluster details page
      router.push(getOrgUrl(`/clusters/${clusterName}`))
    } catch (err: any) {
      console.error('Error updating cluster:', err)
      setError(err.message || 'Failed to update cluster')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(getOrgUrl(`/clusters/${clusterName}`))
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
    )
  }

  if (error && !cluster) {
    return (
      <div className="space-y-6">
        <ResourceHeader
          backHref={getOrgUrl("/clusters")}
          backLabel="Back to Clusters"
          icon={Server}
          title="Edit Cluster"
          subtitle="Failed to load cluster"
        />

        <div className="max-w-2xl">
          <div className="text-center py-12">
            <h3 className="text-lg font-medium mb-2">Error loading cluster</h3>
            <p className="text-muted-foreground mb-4">{error}</p>
            <Link href={getOrgUrl("/clusters")}>
              <Button>Back to Clusters</Button>
            </Link>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <ResourceHeader
        backHref={getOrgUrl(`/clusters/${clusterName}`)}
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
              <Form {...networkForm}>
                <ResourceNetworkPolicyForm
                  control={networkForm.control}
                  egressRulesFieldName="egressRules"
                  watchedEgressRules={watchedEgressRules}
                  setValue={networkForm.setValue}
                  getValues={networkForm.getValues}
                  title="Cluster Network Policy"
                  description="Control external network access for all agents in this cluster. Agents automatically have access to cluster resources and the Kubernetes API server."
                  resourceType="cluster"
                />

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
              </Form>
            </TabsContent>
          </Tabs>
        </div>
    </div>
  )
}