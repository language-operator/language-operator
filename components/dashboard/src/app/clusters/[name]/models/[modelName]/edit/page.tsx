'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ModelForm, ModelFormData } from '@/components/forms/model-form'
import { ArrowLeft } from 'lucide-react'
import { useModel } from '@/hooks/use-models'
import { Skeleton } from '@/components/ui/skeleton'
import Link from 'next/link'

export default function ClusterEditModelPage() {
  const params = useParams()
  const router = useRouter()
  const clusterName = params.name as string
  const modelName = params.modelName as string
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const { data: modelResponse, isLoading: isLoadingModel } = useModel(modelName)
  const model = modelResponse?.data

  const handleSubmit = async (formData: ModelFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch(`/api/models/${modelName}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
          provider: formData.provider,
          model: formData.model,
          endpoint: formData.endpoint,
          apiKey: formData.apiKey || undefined,
          description: formData.description || undefined,
          spec: {
            provider: formData.provider,
            model: formData.model,
            endpoint: formData.endpoint,
            apiKey: formData.apiKey || undefined,
            parameters: {
              maxTokens: formData.maxTokens,
              temperature: formData.temperature,
              topP: formData.topP,
              frequencyPenalty: formData.frequencyPenalty,
              presencePenalty: formData.presencePenalty,
            },
            contextWindow: formData.contextWindow,
            cost: {
              inputTokens: formData.costPerInputToken,
              outputTokens: formData.costPerOutputToken,
              currency: 'USD'
            },
            enabled: formData.enabled,
            requireApproval: formData.requireApproval,
          },
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update model')
      }

      // Redirect to model detail page
      router.push(`/clusters/${clusterName}/models/${modelName}`)
    } catch (err: any) {
      console.error('Error updating model:', err)
      setError(err.message || 'Failed to update model')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/models/${modelName}`)
  }

  // Convert model data to form data format
  const initialData: Partial<ModelFormData> | undefined = model ? {
    name: model.metadata.name,
    provider: model.spec.provider,
    model: model.spec.modelName,
    endpoint: model.spec.endpoint,
    description: model.spec.description || '',
    maxTokens: model.spec.configuration?.maxTokens || 4096,
    temperature: model.spec.configuration?.temperature || 0.7,
    topP: model.spec.configuration?.topP || 1.0,
    frequencyPenalty: model.spec.configuration?.frequencyPenalty || 0.0,
    presencePenalty: model.spec.configuration?.presencePenalty || 0.0,
    contextWindow: model.spec.configuration?.contextWindow || 8192,
    costPerInputToken: model.spec.costTracking?.inputTokenCost || 0.0,
    costPerOutputToken: model.spec.costTracking?.outputTokenCost || 0.0,
    enabled: model.spec.enabled !== false,
    requireApproval: model.spec.requireApproval || false,
    // Note: We don't populate apiKey for security reasons
  } : undefined

  if (isLoadingModel) {
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

  if (!model) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
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
        <div className="flex items-center space-x-4">
          <Link href={`/clusters/${clusterName}/models/${modelName}`}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Model
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Edit Model: {model.metadata.name}</h1>
            <p className="text-muted-foreground">
              Update the configuration for this model in the {clusterName} cluster
            </p>
          </div>
        </div>

        {/* Form */}
        <div className="max-w-2xl">
          <ModelForm
            initialData={initialData}
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