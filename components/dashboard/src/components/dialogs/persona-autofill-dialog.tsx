'use client'

import { useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Loader2, Sparkles } from 'lucide-react'
import { useGeneratePersona } from '@/hooks/use-personas'
import { useModels } from '@/hooks/use-models'
import { PersonaFormData } from '@/components/forms/persona-form-simple'

interface PersonaAutofillDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onGenerated: (data: Partial<PersonaFormData>) => void
  clusterName: string
}

export function PersonaAutofillDialog({
  open,
  onOpenChange,
  onGenerated,
  clusterName,
}: PersonaAutofillDialogProps) {
  const [idea, setIdea] = useState('')
  const [selectedModel, setSelectedModel] = useState('')

  const { data: modelsResponse, isLoading: modelsLoading } = useModels({ clusterName, limit: 100 })
  const generatePersona = useGeneratePersona()

  const models = modelsResponse?.data || []

  // Filter for ready models
  const availableModels = models.filter((model: any) =>
    model.status?.phase === 'Ready'
  )

  const handleGenerate = async () => {
    if (!idea || !selectedModel) return

    try {
      const result = await generatePersona.mutateAsync({
        idea,
        modelName: selectedModel,
      })

      if (result.success && result.data) {
        onGenerated(result.data)
        onOpenChange(false)
        setIdea('')
        setSelectedModel('')
      }
    } catch (error) {
      console.error('Failed to generate persona:', error)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-purple-500" />
            Auto Create with AI
          </DialogTitle>
          <DialogDescription>
            Describe your persona idea and select a model to generate a complete persona automatically.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="idea">Persona Idea</Label>
            <Input
              id="idea"
              placeholder="e.g., A friendly customer support agent for a SaaS product"
              value={idea}
              onChange={(e) => setIdea(e.target.value)}
              disabled={generatePersona.isPending}
            />
            <p className="text-sm text-muted-foreground">
              Describe what kind of persona you want to create
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="model">Select Model</Label>
            {modelsLoading ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span className="ml-2 text-sm text-muted-foreground">Loading models...</span>
              </div>
            ) : availableModels.length === 0 ? (
              <p className="text-sm text-muted-foreground py-2">
                No ready models available in this cluster. Please add a model first.
              </p>
            ) : (
              <Select value={selectedModel} onValueChange={setSelectedModel} disabled={generatePersona.isPending}>
                <SelectTrigger id="model">
                  <SelectValue placeholder="Choose a model" />
                </SelectTrigger>
                <SelectContent>
                  {availableModels.map((model: any) => (
                    <SelectItem key={model.metadata.name} value={model.metadata.name}>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{model.metadata.name}</span>
                        <span className="text-xs text-muted-foreground">
                          ({model.spec.provider} - {model.spec.modelName})
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <p className="text-sm text-muted-foreground">
              Different models may produce different results
            </p>
          </div>

          {generatePersona.isError && (
            <div className="rounded-lg bg-red-50 p-3 text-sm text-red-600">
              {generatePersona.error instanceof Error ? generatePersona.error.message : 'Failed to generate persona'}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              onOpenChange(false)
              setIdea('')
              setSelectedModel('')
            }}
            disabled={generatePersona.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleGenerate}
            disabled={!idea || !selectedModel || generatePersona.isPending || availableModels.length === 0}
          >
            {generatePersona.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <Sparkles className="mr-2 h-4 w-4" />
                Generate Persona
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
