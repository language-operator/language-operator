'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { PersonaForm, PersonaFormData } from '@/components/forms/persona-form'
import { usePersona } from '@/hooks/use-personas'
import { Button } from '@/components/ui/button'
import { ArrowLeft, Users } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { ResourceHeader } from '@/components/ui/resource-header'

export default function EditClusterPersonaPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const personaName = params?.personaName as string
  
  const { data: personaResponse, isLoading: isLoadingPersona, error } = usePersona(personaName)
  const persona = personaResponse?.persona
  
  const [isLoading, setIsLoading] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const [initialData, setInitialData] = useState<Partial<PersonaFormData>>()

  useEffect(() => {
    if (persona) {
      setInitialData({
        name: persona.metadata.name,
        displayName: persona.spec.displayName || '',
        description: persona.spec.description || '',
        systemPrompt: persona.spec.systemPrompt || '',
        tone: persona.spec.tone || '',
        language: persona.spec.language || '',
        version: persona.spec.version || '',
        capabilities: persona.spec.capabilities || [],
        limitations: persona.spec.limitations || [],
        instructions: persona.spec.instructions || [],
        examples: persona.spec.examples || [],
      })
    }
  }, [persona])

  const handleSubmit = async (formData: PersonaFormData) => {
    setIsLoading(true)
    setSubmitError('')

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
        }
      }
      
      console.log('Updating persona with payload:', payload)
      
      const response = await fetch(`/api/personas/${personaName}`, {
        method: 'PATCH',
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
        
        throw new Error(errorData.error || 'Failed to update persona')
      }

      const result = await response.json()
      console.log('Update persona result:', result)
      
      // Redirect to persona detail page
      router.push(`/clusters/${clusterName}/personas/${personaName}`)
    } catch (err: any) {
      console.error('Error updating persona:', err)
      setSubmitError(err.message || 'Failed to update persona')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/personas/${personaName}`)
  }


  if (isLoadingPersona) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <ResourceHeader
            backHref={`/clusters/${clusterName}/personas`}
            backLabel="Back to Personas"
            icon={Users}
            iconColor="text-blue-500"
            title={<Skeleton className="h-6 w-48" />}
            subtitle={<Skeleton className="h-4 w-64" />}
          />
          <Skeleton className="h-96 w-full" />
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error || !persona) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <ResourceHeader
            backHref={`/clusters/${clusterName}/personas`}
            backLabel="Back to Personas"
            icon={Users}
            iconColor="text-blue-500"
            title="Persona Not Found"
            subtitle={`The persona "${personaName}" was not found in cluster "${clusterName}"`}
          />
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          backHref={`/clusters/${clusterName}/personas`}
          backLabel="Back to Personas"
          icon={Users}
          iconColor="text-blue-500"
          title="Edit Language Persona"
          subtitle={`Edit "${persona.spec.displayName || persona.metadata.name}" in the ${clusterName} cluster`}
        />

        {/* Form */}
        <div className="max-w-4xl">
          {initialData ? (
            <PersonaForm
              initialData={initialData}
              isLoading={isLoading}
              error={submitError}
              onSubmit={handleSubmit}
              onCancel={handleCancel}
              isEdit={true}
            />
          ) : (
            <Skeleton className="h-96 w-full" />
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}