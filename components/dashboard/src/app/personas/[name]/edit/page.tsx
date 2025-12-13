'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { PersonaForm, PersonaFormData } from '@/components/forms/persona-form'
import { ArrowLeft } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import Link from 'next/link'

interface Persona {
  metadata: {
    name: string
    namespace: string
    creationTimestamp: string
  }
  spec: {
    role: string
    systemPrompt: string
    traits: string[]
    examples?: Array<{input: string, output: string}>
    parameters?: {
      temperature?: number
      maxTokens?: number
    }
    enabled?: boolean
    requireApproval?: boolean
  }
  status: {
    phase: string
  }
}

export default function EditPersonaPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter()
  const [persona, setPersona] = useState<Persona | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingPersona, setIsLoadingPersona] = useState(true)
  const [error, setError] = useState('')
  const [personaName, setPersonaName] = useState<string>('')

  // Get the persona name from params
  useEffect(() => {
    const getPersonaName = async () => {
      const resolvedParams = await params
      setPersonaName(resolvedParams.name)
    }
    getPersonaName()
  }, [params])

  // Fetch existing persona data
  useEffect(() => {
    if (!personaName) return

    const fetchPersona = async () => {
      setIsLoadingPersona(true)
      try {
        const response = await fetch(`/api/personas/${personaName}`)
        if (!response.ok) {
          throw new Error('Failed to fetch persona')
        }
        const data = await response.json()
        setPersona(data.persona)
      } catch (err: any) {
        console.error('Error fetching persona:', err)
        setError(err.message || 'Failed to load persona')
      } finally {
        setIsLoadingPersona(false)
      }
    }

    fetchPersona()
  }, [personaName])

  const handleSubmit = async (formData: PersonaFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch(`/api/personas/${personaName}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          role: formData.role === 'custom' ? formData.customRole : formData.role,
          description: formData.description || undefined,
          spec: {
            role: formData.role === 'custom' ? formData.customRole : formData.role,
            systemPrompt: formData.systemPrompt,
            traits: formData.traits,
            examples: formData.examples.filter(ex => ex.input.trim() && ex.output.trim()),
            parameters: {
              temperature: formData.temperature,
              maxTokens: formData.maxTokens,
            },
            enabled: formData.enabled,
            requireApproval: formData.requireApproval,
          },
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update persona')
      }

      // Redirect to persona details page
      router.push(`/personas/${personaName}`)
    } catch (err: any) {
      console.error('Error updating persona:', err)
      setError(err.message || 'Failed to update persona')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/personas/${personaName}`)
  }

  // Convert persona data to form format
  const getInitialFormData = (): Partial<PersonaFormData> | undefined => {
    if (!persona) return undefined

    // Determine if it's a custom role by checking against known roles
    const knownRoles = ['assistant', 'analyst', 'developer', 'writer', 'researcher', 'teacher', 'consultant', 'support']
    const isCustomRole = !knownRoles.includes(persona.spec.role)

    return {
      name: persona.metadata.name,
      role: isCustomRole ? 'custom' : persona.spec.role,
      customRole: isCustomRole ? persona.spec.role : '',
      description: '', // Personas don't have description in current schema
      systemPrompt: persona.spec.systemPrompt,
      traits: persona.spec.traits || [],
      examples: persona.spec.examples || [
        { input: '', output: '' },
        { input: '', output: '' }
      ],
      temperature: persona.spec.parameters?.temperature || 0.7,
      maxTokens: persona.spec.parameters?.maxTokens || 2048,
      enabled: persona.spec.enabled ?? true,
      requireApproval: persona.spec.requireApproval ?? false,
    }
  }

  if (isLoadingPersona) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Skeleton className="h-8 w-32" />
            <div>
              <Skeleton className="h-8 w-64 mb-2" />
              <Skeleton className="h-4 w-48" />
            </div>
          </div>
          <div className="max-w-2xl space-y-6">
            <Skeleton className="h-48" />
            <Skeleton className="h-64" />
            <Skeleton className="h-48" />
            <Skeleton className="h-32" />
            <Skeleton className="h-24" />
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error && !persona) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Link href="/personas">
              <Button variant="outline" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Personas
              </Button>
            </Link>
            <div>
              <h1 className="text-3xl font-bold">Edit Persona</h1>
              <p className="text-muted-foreground">Failed to load persona</p>
            </div>
          </div>
          <div className="max-w-2xl">
            <div className="text-center py-12">
              <h3 className="text-lg font-medium mb-2">Error loading persona</h3>
              <p className="text-muted-foreground mb-4">{error}</p>
              <Link href="/personas">
                <Button>Back to Personas</Button>
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
        <div className="flex items-center space-x-4">
          <Link href={`/personas/${personaName}`}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Persona
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Edit Persona</h1>
            <p className="text-muted-foreground">
              Update settings for persona "{personaName}"
            </p>
          </div>
        </div>

        <div className="max-w-2xl">
          <PersonaForm
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