'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ModelForm, ModelFormData } from '@/components/forms/model-form'
import { ArrowLeft } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import Link from 'next/link'

interface Model {
  metadata: {
    name: string
    namespace: string
    creationTimestamp: string
  }
  spec: {
    provider: string
    model: string
    endpoint: string
    apiKey?: string
    description?: string
    parameters?: {
      maxTokens?: number
      temperature?: number
      topP?: number
      frequencyPenalty?: number
      presencePenalty?: number
    }
    contextWindow?: number
    cost?: {
      inputTokens?: number
      outputTokens?: number
      currency?: string
    }
    enabled?: boolean
    requireApproval?: boolean
  }
  status: {
    phase: string
  }
}

export default function EditModelPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter()
  const [model, setModel] = useState<Model | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingModel, setIsLoadingModel] = useState(true)
  const [error, setError] = useState('')
  const [modelName, setModelName] = useState<string>('')

  // Get the model name from params
  useEffect(() => {
    const getModelName = async () => {
      const resolvedParams = await params
      setModelName(resolvedParams.name)
    }
    getModelName()
  }, [params])

  // Fetch existing model data
  useEffect(() => {
    if (!modelName) return

    const fetchModel = async () => {
      setIsLoadingModel(true)
      try {
        const response = await fetch(`/api/models/${modelName}`)
        if (!response.ok) {
          throw new Error('Failed to fetch model')
        }
        const data = await response.json()
        setModel(data.model)
      } catch (err: any) {
        console.error('Error fetching model:', err)
        setError(err.message || 'Failed to load model')
      } finally {
        setIsLoadingModel(false)
      }
    }

    fetchModel()
  }, [modelName])

  const handleSubmit = async (formData: ModelFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch(`/api/models/${modelName}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
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

      // Redirect to model details page
      router.push(`/models/${modelName}`)
    } catch (err: any) {
      console.error('Error updating model:', err)
      setError(err.message || 'Failed to update model')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/models/${modelName}`)
  }

  // Convert model data to form format
  const getInitialFormData = (): Partial<ModelFormData> | undefined => {
    if (!model) return undefined

    return {
      name: model.metadata.name,
      provider: model.spec.provider,
      model: model.spec.model,
      endpoint: model.spec.endpoint,
      apiKey: model.spec.apiKey || '',
      description: model.spec.description || '',
      maxTokens: model.spec.parameters?.maxTokens || 4096,
      temperature: model.spec.parameters?.temperature || 0.7,
      topP: model.spec.parameters?.topP || 1.0,
      frequencyPenalty: model.spec.parameters?.frequencyPenalty || 0.0,
      presencePenalty: model.spec.parameters?.presencePenalty || 0.0,
      contextWindow: model.spec.contextWindow || 8192,
      costPerInputToken: model.spec.cost?.inputTokens || 0.0,
      costPerOutputToken: model.spec.cost?.outputTokens || 0.0,
      enabled: model.spec.enabled ?? true,
      requireApproval: model.spec.requireApproval ?? false,
    }
  }

  if (isLoadingModel) {
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
            <Skeleton className="h-48" />
            <Skeleton className="h-32" />
            <Skeleton className="h-32" />
            <Skeleton className="h-24" />
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error && !model) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Link href="/models">
              <Button variant="outline" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Models
              </Button>
            </Link>
            <div>
              <h1 className="text-3xl font-bold">Edit Model</h1>
              <p className="text-muted-foreground">Failed to load model</p>
            </div>
          </div>

          <div className="max-w-2xl">
            <div className="text-center py-12">
              <h3 className="text-lg font-medium mb-2">Error loading model</h3>
              <p className="text-muted-foreground mb-4">{error}</p>
              <Link href="/models">
                <Button>Back to Models</Button>
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
        <div className="flex items-center space-x-4">
          <Link href={`/models/${modelName}`}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Model
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Edit Model</h1>
            <p className="text-muted-foreground">
              Update settings for model "{modelName}"
            </p>
          </div>
        </div>

        {/* Form */}
        <div className="max-w-2xl">
          <ModelForm
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