'use client'

import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { useOrganization } from '@/components/organization-provider'
import { PersonaFormSimple, PersonaFormData } from '@/components/forms/persona-form-simple'
import { fetchWithOrganization } from '@/lib/api-client'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Button } from '@/components/ui/button'
import { Users, Sparkles } from 'lucide-react'
import { PersonaAutofillDialog } from '@/components/dialogs/persona-autofill-dialog'

export default function CreateClusterPersonaPage() {
  const router = useRouter()
  const params = useParams()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [autofillOpen, setAutofillOpen] = useState(false)
  const [generatedData, setGeneratedData] = useState<Partial<PersonaFormData> | null>(null)

  const handleSubmit = async (formData: PersonaFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const payload = {
        name: formData.name,
        displayName: formData.displayName,
        description: formData.description,
        ...(formData.systemPrompt && { systemPrompt: formData.systemPrompt }),
        ...(formData.tone && { tone: formData.tone }),
        ...(formData.language && { language: formData.language }),
        ...(formData.instructions && formData.instructions.length > 0 && { instructions: formData.instructions }),
      }
      
      console.log('Sending payload:', payload)
      
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/personas`, {
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
      router.push(getOrgUrl(`/clusters/${clusterName}/personas`))
    } catch (err: any) {
      console.error('Error creating persona:', err)
      setError(err.message || 'Failed to create persona')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(getOrgUrl(`/clusters/${clusterName}/personas`))
  }

  const handleAutofillGenerated = (data: Partial<PersonaFormData>) => {
    setGeneratedData(data)
  }

  return (
    <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          backHref={getOrgUrl(`/clusters/${clusterName}/personas`)}
          backLabel="Back to Personas"
          icon={Users}
          title="Create Persona"
          subtitle="Define specific personality traits and custom instructions"
          actions={
            <Button
              variant="outline"
              onClick={() => setAutofillOpen(true)}
              className="gap-2"
            >
              <Sparkles className="h-4 w-4" />
              Auto Create
            </Button>
          }
        />

        {/* Form */}
        {error && (
          <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
            <p className="text-sm text-red-800">{error}</p>
          </div>
        )}
        <PersonaFormSimple
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          submitLabel="Create Persona"
          clusterName={clusterName}
          initialData={generatedData || undefined}
        />

        {/* Auto-fill Dialog */}
        <PersonaAutofillDialog
          open={autofillOpen}
          onOpenChange={setAutofillOpen}
          onGenerated={handleAutofillGenerated}
          clusterName={clusterName}
        />
    </div>
  )
}