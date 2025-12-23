'use client'

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Trash2, Plus } from 'lucide-react'
import { useState, useEffect } from 'react'

// Form data type matching implemented fields
export type PersonaFormData = {
  name: string
  displayName?: string
  description?: string
  systemPrompt?: string
  tone?: string
  language?: string
  instructions?: string[]
}

// Validation schema for persona form
const personaSchema = z.object({
  name: z.string().min(1, 'Name is required').regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/, 'Name must be lowercase alphanumeric with hyphens'),
  displayName: z.string().optional(),
  description: z.string().optional(),
  systemPrompt: z.string().optional(),
  tone: z.string().optional(),
  language: z.string().optional(),
  instructions: z.array(z.string()).optional(),
})

export interface PersonaFormProps {
  initialData?: Partial<PersonaFormData>
  onSubmit: (data: PersonaFormData) => Promise<void>
  onCancel: () => void
  submitLabel?: string
  isEdit?: boolean
  clusterName?: string
}

export function PersonaFormSimple({
  initialData,
  onSubmit,
  onCancel,
  submitLabel = 'Create Persona',
  isEdit = false,
  clusterName
}: PersonaFormProps) {
  const [instructions, setInstructions] = useState<string[]>(initialData?.instructions || [])

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setValue,
    watch,
    reset,
  } = useForm<PersonaFormData>({
    resolver: zodResolver(personaSchema),
    defaultValues: {
      name: initialData?.name || '',
      displayName: initialData?.displayName || '',
      description: initialData?.description || '',
      systemPrompt: initialData?.systemPrompt || '',
      tone: initialData?.tone || 'professional',
      language: initialData?.language || 'en',
      instructions: initialData?.instructions || [],
    },
  })

  const tone = watch('tone')

  // Update form when initialData changes (e.g., from AI generation)
  useEffect(() => {
    if (initialData) {
      reset({
        name: initialData.name || '',
        displayName: initialData.displayName || '',
        description: initialData.description || '',
        systemPrompt: initialData.systemPrompt || '',
        tone: initialData.tone || 'professional',
        language: initialData.language || 'en',
        instructions: initialData.instructions || [],
      })
      setInstructions(initialData.instructions || [])
    }
  }, [initialData, reset])

  const handleFormSubmit = async (data: PersonaFormData) => {
    const formData = {
      ...data,
      instructions: instructions.filter(i => i.trim() !== ''),
    }
    await onSubmit(formData)
  }

  const addInstruction = () => {
    setInstructions([...instructions, ''])
  }

  const removeInstruction = (index: number) => {
    setInstructions(instructions.filter((_, i) => i !== index))
  }

  const updateInstruction = (index: number, value: string) => {
    const updated = [...instructions]
    updated[index] = value
    setInstructions(updated)
    setValue('instructions', updated)
  }

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
          <CardDescription>Configure the basic settings for your persona</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Persona Name *</Label>
            <Input
              id="name"
              {...register('name')}
              placeholder="technical-architect"
              disabled={isEdit}
            />
            {errors.name && <p className="text-sm text-red-500">{errors.name.message}</p>}
            {isEdit && <p className="text-sm text-muted-foreground">Name cannot be changed after creation</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="displayName">Display Name</Label>
            <Input
              id="displayName"
              {...register('displayName')}
              placeholder="Technical Architect"
            />
            {errors.displayName && <p className="text-sm text-red-500">{errors.displayName.message}</p>}
            <p className="text-sm text-muted-foreground">Human-readable name for this persona (optional)</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="systemPrompt">Core Capabilities</Label>
            <Textarea
              id="systemPrompt"
              {...register('systemPrompt')}
              placeholder="You are a senior technical architect with deep expertise in distributed systems, cloud infrastructure, and software design patterns. You provide strategic technical guidance and help teams make informed architectural decisions..."
              rows={5}
            />
            {errors.systemPrompt && <p className="text-sm text-red-500">{errors.systemPrompt.message}</p>}
            <p className="text-sm text-muted-foreground">Core behavioral instructions that define how this persona should act</p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="tone">Tone</Label>
              <Select value={tone} onValueChange={(value) => setValue('tone', value)}>
                <SelectTrigger>
                  <SelectValue placeholder="Select tone" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="professional">Professional</SelectItem>
                  <SelectItem value="casual">Casual</SelectItem>
                  <SelectItem value="friendly">Friendly</SelectItem>
                  <SelectItem value="formal">Formal</SelectItem>
                  <SelectItem value="technical">Technical</SelectItem>
                  <SelectItem value="empathetic">Empathetic</SelectItem>
                  <SelectItem value="concise">Concise</SelectItem>
                  <SelectItem value="detailed">Detailed</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-sm text-muted-foreground">The overall tone and communication style</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="language">Language</Label>
              <Input
                id="language"
                {...register('language')}
                placeholder="en"
              />
              <p className="text-sm text-muted-foreground">Primary language for responses (e.g., en, es, fr)</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Additional Instructions */}
      <Card>
        <CardHeader>
          <CardTitle>Additional Instructions</CardTitle>
          <CardDescription>Additional specific instructions for this persona</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {instructions.map((instruction, index) => (
            <div key={index} className="flex gap-2">
              <Textarea
                value={instruction}
                onChange={(e) => updateInstruction(index, e.target.value)}
                placeholder="Enter an instruction..."
                rows={2}
                className="flex-1"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={() => removeInstruction(index)}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
          <Button type="button" variant="outline" onClick={addInstruction}>
            <Plus className="h-4 w-4 mr-2" />
            Add Instruction
          </Button>
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex justify-end gap-4">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Saving...' : submitLabel}
        </Button>
      </div>
    </form>
  )
}
