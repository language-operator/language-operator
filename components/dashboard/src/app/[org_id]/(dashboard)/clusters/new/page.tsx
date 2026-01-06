'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { ClusterForm, ClusterFormData } from '@/components/forms/cluster-form'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Boxes } from 'lucide-react'
import { useApiClient } from '@/lib/api-client'
import { useOrganization } from '@/components/organization-provider'

export default function CreateClusterPage() {
  const router = useRouter()
  const { getOrgUrl } = useOrganization()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const apiClient = useApiClient()

  const handleSubmit = async (formData: ClusterFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await apiClient.post('/clusters', {
        name: formData.name,
        domain: formData.domain || undefined,
        gatewayName: formData.gatewayName || undefined,
        ingressClassName: formData.ingressClassName || undefined,
        enableTLS: formData.enableTLS,
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create cluster')
      }

      const result = await response.json()
      
      console.log('Create cluster response:', result)
      
      // Redirect to cluster details page
      const clusterName = result.data?.metadata?.name || formData.name
      router.push(getOrgUrl(`/clusters/${clusterName}`))
    } catch (err: any) {
      console.error('Error creating cluster:', err)
      setError(err.message || 'Failed to create cluster')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(getOrgUrl('/clusters'))
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <ResourceHeader
        backHref={getOrgUrl('/clusters')}
        backLabel="Back to Clusters"
        icon={Boxes}
        title="Create Language Cluster"
        subtitle="Set up a new cluster for deploying language agents"
      />

      {/* Form */}
      <div className="max-w-2xl">
        <ClusterForm
          isLoading={isLoading}
          error={error}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
        />
      </div>
    </div>
  )
}