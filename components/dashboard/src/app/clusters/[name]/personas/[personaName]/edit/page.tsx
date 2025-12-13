'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { PersonaForm, PersonaFormData } from '@/components/forms/persona-form'
import { usePersonas } from '@/hooks/use-personas'

export default function EditClusterPersonaPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const personaName = params?.personaName as string
  
  const { data: personas, isLoading: isLoadingPersonas } = usePersonas()
  const persona = personas?.find((p: any) => p.metadata.name === personaName)
  
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [initialData, setInitialData] = useState<Partial<PersonaFormData>>()

  useEffect(() => {
    if (persona) {
      setInitialData({
        name: persona.metadata.name,
        role: persona.spec.role || '',
        customRole: persona.spec.customRole || '',
        description: persona.spec.description || '',
        systemPrompt: persona.spec.systemPrompt || '',
        traits: persona.spec.traits || [],
        examples: persona.spec.examples || [
          { input: '', output: '' },
          { input: '', output: '' }
        ],
        temperature: persona.spec.temperature || 0.7,
        maxTokens: persona.spec.maxTokens || 2048,
        enabled: persona.spec.enabled !== false,
        requireApproval: persona.spec.requireApproval || false,
      })
    }
  }, [persona])

  const handleSubmit = async (formData: PersonaFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const payload = {
        name: formData.name,
        role: formData.role,
        customRole: formData.customRole,
        description: formData.description,
        systemPrompt: formData.systemPrompt,
        traits: formData.traits,
        examples: formData.examples,
        temperature: formData.temperature,
        maxTokens: formData.maxTokens,
        enabled: formData.enabled,
        requireApproval: formData.requireApproval,
      }
      
      console.log('Updating persona with payload:', payload)
      
      const response = await fetch(`/api/personas/${personaName}`, {
        method: 'PUT',
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
      setError(err.message || 'Failed to update persona')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/personas/${personaName}`)
  }

  if (isLoadingPersonas) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div>
            <div className="h-8 w-48 bg-gray-200 rounded animate-pulse"></div>
            <div className="h-4 w-64 bg-gray-200 rounded mt-2 animate-pulse"></div>
          </div>
          <div className="h-96 bg-gray-200 rounded animate-pulse"></div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (!persona) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div>
            <h1 className="text-3xl font-bold">Persona Not Found</h1>
            <p className="text-muted-foreground mt-1">
              The persona "{personaName}" was not found in cluster "{clusterName}"
            </p>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold">Edit Language Persona</h1>
          <p className="text-muted-foreground mt-1">
            Edit "{personaName}" in the {clusterName} cluster
          </p>
        </div>

        {/* Form */}
        <div className="max-w-4xl">
          {initialData ? (
            <PersonaForm
              initialData={initialData}
              isLoading={isLoading}
              error={error}
              onSubmit={handleSubmit}
              onCancel={handleCancel}
              isEdit={true}
            />
          ) : (
            <div className="h-96 bg-gray-200 rounded animate-pulse"></div>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}