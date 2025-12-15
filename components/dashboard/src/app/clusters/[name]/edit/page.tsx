'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ClusterForm, ClusterFormData } from '@/components/forms/cluster-form'
import { ArrowLeft, Server } from 'lucide-react'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Skeleton } from '@/components/ui/skeleton'
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
        const response = await fetch(`/api/clusters/${clusterName}`)
        if (!response.ok) {
          throw new Error('Failed to fetch cluster')
        }
        const data = await response.json()
        setCluster(data.cluster)
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
      const response = await fetch(`/api/clusters/${clusterName}`, {
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

        {/* Form */}
        <div className="max-w-2xl">
          <ClusterForm
            initialData={getInitialFormData()}
            isLoading={isLoading}
            error={error}
            onSubmit={handleSubmit}
            onCancel={handleCancel}
            isEdit={true}
          />
        </div>
      </div>
    </AuthenticatedLayout>
  )
}