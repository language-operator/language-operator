'use client'

import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ModelForm, ModelFormData } from '@/components/forms/model-form'

export default function CreateClusterModelPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
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
      
      const response = await fetch('/api/models', {
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
      router.push(`/clusters/${clusterName}/models`)
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
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold">Create Language Model</h1>
          <p className="text-muted-foreground mt-1">
            Add a new language model to the {clusterName} cluster
          </p>
        </div>

        {/* Form */}
        <div className="max-w-2xl">
          <ModelForm
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