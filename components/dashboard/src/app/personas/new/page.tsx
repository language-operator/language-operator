'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { PersonaForm, PersonaFormData } from '@/components/forms/persona-form'
import { ArrowLeft } from 'lucide-react'
import Link from 'next/link'

export default function CreatePersonaPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (formData: PersonaFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch('/api/personas', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
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
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create persona')
      }

      const result = await response.json()
      
      // Redirect to persona details page
      router.push(`/personas/${result.data.metadata.name}`)
    } catch (err: any) {
      console.error('Error creating persona:', err)
      setError(err.message || 'Failed to create persona')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push('/personas')
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center space-x-4">
          <Link href="/personas">
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Personas
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Create Language Persona</h1>
            <p className="text-muted-foreground">
              Define a personality and behavior pattern for agents
            </p>
          </div>
        </div>

        {/* Form */}
        <div className="max-w-2xl">
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