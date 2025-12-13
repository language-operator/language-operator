'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ClusterForm, ClusterFormData } from '@/components/forms/cluster-form'
import { ArrowLeft } from 'lucide-react'
import Link from 'next/link'

export default function CreateClusterPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (formData: ClusterFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch('/api/clusters', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
          domain: formData.domain || undefined,
          gatewayName: formData.gatewayName || undefined,
          ingressClassName: formData.ingressClassName || undefined,
          enableTLS: formData.enableTLS,
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create cluster')
      }

      const result = await response.json()
      
      console.log('Create cluster response:', result)
      
      // Redirect to cluster details page
      const clusterName = result.data?.metadata?.name || formData.name
      router.push(`/clusters/${clusterName}`)
    } catch (err: any) {
      console.error('Error creating cluster:', err)
      setError(err.message || 'Failed to create cluster')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push('/clusters')
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center space-x-4">
          <Link href="/clusters">
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Clusters
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Create Language Cluster</h1>
            <p className="text-muted-foreground">
              Set up a new cluster for deploying language agents
            </p>
          </div>
        </div>

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
    </AuthenticatedLayout>
  )
}