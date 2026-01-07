'use client'

import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { ModelForm, ModelFormData } from '@/components/forms/model-form'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Cpu } from 'lucide-react'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganization } from '@/components/organization-provider'

export default function CreateClusterModelPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const { getOrgUrl } = useOrganization()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (formData: ModelFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const payload = {
        name: formData.name,
        provider: formData.provider,
        modelName: formData.model,
        endpoint: formData.endpoint,
        ...(formData.apiKey && { 
          apiKeySecretName: `${formData.name}-api-key`,
          apiKeySecretKey: 'api-key' 
        }),
        temperature: formData.temperature,
        maxTokens: formData.maxTokens,
        topP: formData.topP,
      }
      
      console.log('Sending payload:', payload)
      
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/models`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        const errorData = await response.json()
        console.error('API Error Response:', errorData)
        
        // Show detailed validation errors if available
        if (errorData.details && Array.isArray(errorData.details)) {
          const detailMessages = errorData.details.map((d: any) => `${d.path}: ${d.message}`).join(', ')
          throw new Error(`${errorData.error || 'Validation failed'}: ${detailMessages}`)
        }
        
        throw new Error(errorData.error || 'Failed to create model')
      }

      const result = await response.json()
      console.log('Create model result:', result)
      
      // Redirect to cluster models list page (since model detail pages may not exist yet)
      router.push(getOrgUrl(`/clusters/${clusterName}/models`))
    } catch (err: any) {
      console.error('Error creating model:', err)
      setError(err.message || 'Failed to create model')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/models`)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <ResourceHeader
        backHref={`/clusters/${clusterName}/models`}
        backLabel="Back to Models"
        icon={Cpu}
        title="Create Language Model"
        subtitle={`Add a new language model to the ${clusterName} cluster`}
      />

      {/* Form */}
      <div className="max-w-2xl">
        <ModelForm
          isLoading={isLoading}
          error={error}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          clusterName={clusterName}
        />
      </div>
    </div>
  )
}