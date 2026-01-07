'use client'

import { useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Loader2, Sparkles, AlertCircle, RefreshCw, ExternalLink } from 'lucide-react'
import { useGeneratePersona } from '@/hooks/use-personas'
import { useModels } from '@/hooks/use-models'
import { PersonaFormData } from '@/components/forms/persona-form-simple'
import { useOrganization } from '@/components/organization-provider'

interface PersonaAutofillDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onGenerated: (data: Partial<PersonaFormData>) => void
  clusterName: string
}

interface ErrorDisplayProps {
  error: any
  onRetry?: () => void
  clusterName?: string
  getOrgUrl: (path: string) => string
}

function ErrorDisplay({ error, onRetry, clusterName, getOrgUrl }: ErrorDisplayProps) {
  if (!error) return null
  
  let errorMessage = 'An unexpected error occurred'
  let errorCode: string | undefined
  let details: string | undefined
  let canRetry = false
  let showModelLink = false
  
  // Handle different error structures from React Query
  let apiError: any = error
  
  // React Query may wrap the error or store response in different places
  if (error.response) {
    // Axios-style error
    apiError = error.response.data
  } else if (error.message && typeof error.message === 'string' && error.message.startsWith('{')) {
    // JSON string in message
    try {
      apiError = JSON.parse(error.message)
    } catch {
      errorMessage = error.message
    }
  } else if (error instanceof Error) {
    errorMessage = error.message
  }
  
  // Extract error details if it's our API error format
  if (apiError && typeof apiError === 'object') {
    errorCode = apiError.code
    details = apiError.details || apiError.error
    errorMessage = apiError.error || errorMessage
  }
  
  // Determine error type and messaging
  let icon = <AlertCircle className="h-4 w-4 text-red-500" />
  let title = 'Generation Failed'
  let actionButtons: React.ReactNode = null
  
  switch (errorCode) {
    case 'MODEL_NOT_AVAILABLE':
      icon = <AlertCircle className="h-4 w-4 text-amber-500" />
      title = 'Model Not Available'
      showModelLink = true
      break
      
    case 'MODEL_ENDPOINT_ERROR':
      icon = <AlertCircle className="h-4 w-4 text-red-500" />
      title = 'Model Service Unavailable'
      canRetry = true
      break
      
    case 'MODEL_RESPONSE_ERROR':
      const is500Error = details?.includes('500') ?? false
      icon = <AlertCircle className="h-4 w-4 text-red-500" />
      title = 'Model Error'
      canRetry = is500Error
      break
      
    case 'GENERATION_TIMEOUT':
      icon = <AlertCircle className="h-4 w-4 text-amber-500" />
      title = 'Generation Timeout'
      canRetry = true
      break
      
    case 'GENERATION_PARSING_ERROR':
      icon = <AlertCircle className="h-4 w-4 text-orange-500" />
      title = 'Invalid Response Format'
      canRetry = true
      break
      
    default:
      canRetry = true
      break
  }
  
  if (canRetry && onRetry) {
    actionButtons = (
      <Button variant="outline" size="sm" onClick={onRetry} className="mt-2">
        <RefreshCw className="mr-2 h-3 w-3" />
        Try Again
      </Button>
    )
  }
  
  if (showModelLink && clusterName) {
    actionButtons = (
      <Button variant="outline" size="sm" asChild className="mt-2">
        <a href={getOrgUrl(`/clusters/${clusterName}/models`)} target="_blank" rel="noopener noreferrer">
          <ExternalLink className="mr-2 h-3 w-3" />
          Check Models
        </a>
      </Button>
    )
  }
  
  return (
    <div className="rounded-lg bg-red-50 border border-red-200 p-3">
      <div className="flex items-start gap-2">
        {icon}
        <div className="flex-1 min-w-0">
          <div className="font-medium text-red-900 text-sm">{title}</div>
          <div className="text-red-700 text-sm mt-1">
            {details || errorMessage}
          </div>
          {actionButtons}
        </div>
      </div>
    </div>
  )
}

export function PersonaAutofillDialog({
  open,
  onOpenChange,
  onGenerated,
  clusterName,
}: PersonaAutofillDialogProps) {
  const [idea, setIdea] = useState('')
  const [selectedModel, setSelectedModel] = useState('')
  const { getOrgUrl } = useOrganization()

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
      // Error will be displayed by the ErrorDisplay component
    }
  }

  const handleRetry = () => {
    generatePersona.reset() // Clear the error state
    handleGenerate()
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
            <ErrorDisplay 
              error={generatePersona.error} 
              onRetry={handleRetry}
              clusterName={clusterName}
              getOrgUrl={getOrgUrl}
            />
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
