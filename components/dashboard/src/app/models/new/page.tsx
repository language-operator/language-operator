'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ModelForm, ModelFormData } from '@/components/forms/model-form'
import { ArrowLeft } from 'lucide-react'
import Link from 'next/link'

export default function CreateModelPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (formData: ModelFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch('/api/models', {
        method: 'POST',
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
        throw new Error(errorData.error || 'Failed to create model')
      }

      const result = await response.json()
      
      // Redirect to model details page
      router.push(`/models/${result.model.metadata.name}`)
    } catch (err: any) {
      console.error('Error creating model:', err)
      setError(err.message || 'Failed to create model')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push('/models')
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center space-x-4">
          <Link href="/models">
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Models
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Create Language Model</h1>
            <p className="text-muted-foreground">
              Add a new language model to your organization
            </p>
          </div>
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