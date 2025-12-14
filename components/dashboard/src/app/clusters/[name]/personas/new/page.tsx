'use client'

import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { PersonaForm, PersonaFormData } from '@/components/forms/persona-form'

export default function CreateClusterPersonaPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (formData: PersonaFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const payload = {
        spec: {
          displayName: formData.displayName,
          description: formData.description,
          systemPrompt: formData.systemPrompt,
          ...(formData.tone && { tone: formData.tone }),
          ...(formData.language && { language: formData.language }),
          ...(formData.version && { version: formData.version }),
          ...(formData.capabilities && formData.capabilities.length > 0 && { capabilities: formData.capabilities }),
          ...(formData.limitations && formData.limitations.length > 0 && { limitations: formData.limitations }),
          ...(formData.instructions && formData.instructions.length > 0 && { instructions: formData.instructions }),
          ...(formData.examples && formData.examples.length > 0 && { 
            examples: formData.examples.map(ex => ({
              input: ex.input,
              output: ex.output,
              ...(ex.context && { context: ex.context }),
              ...(ex.tags && ex.tags.length > 0 && { tags: ex.tags })
            }))
          }),
        },
        name: formData.name,
      }
      
      console.log('Sending payload:', payload)
      
      const response = await fetch('/api/personas', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        const errorData = await response.json()
        console.error('API Error Response:', errorData)
        
        if (errorData.details && Array.isArray(errorData.details)) {
          const detailMessages = errorData.details.map((d: any) => `${d.path}: ${d.message}`).join(', ')
          throw new Error(`${errorData.error || 'Validation failed'}: ${detailMessages}`)
        }
        
        throw new Error(errorData.error || 'Failed to create persona')
      }

      const result = await response.json()
      console.log('Create persona result:', result)
      
      // Redirect to cluster personas list page
      router.push(`/clusters/${clusterName}/personas`)
    } catch (err: any) {
      console.error('Error creating persona:', err)
      setError(err.message || 'Failed to create persona')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/personas`)
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold">Create Language Persona</h1>
          <p className="text-muted-foreground mt-1">
            Add a new language persona to the {clusterName} cluster
          </p>
        </div>

        {/* Form */}
        <div className="max-w-4xl">
          <PersonaForm
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