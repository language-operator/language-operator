'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Brain, Zap, DollarSign, AlertCircle, Server, RefreshCw } from 'lucide-react'

const PROVIDERS = [
  { id: 'openai', name: 'OpenAI', endpoint: 'https://api.openai.com/v1', requiresEndpoint: false },
  { id: 'anthropic', name: 'Anthropic', endpoint: 'https://api.anthropic.com/v1', requiresEndpoint: false },
  { id: 'openai-compatible', name: 'Local/OpenAI Compatible', endpoint: '', requiresEndpoint: true },
]

const KNOWN_MODELS = {
  openai: ['gpt-4', 'gpt-4-turbo', 'gpt-3.5-turbo', 'gpt-4o', 'gpt-4o-mini'],
  anthropic: ['claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307', 'claude-3-5-sonnet-20241022'],
  'openai-compatible': ['llama3:8b', 'llama3:70b', 'phi3:14b', 'codellama:7b', 'mistral:7b'],
}

export interface ModelFormData {
  name: string
  provider: string
  model: string
  endpoint: string
  apiKey: string
  description: string
  maxTokens: number
  temperature: number
  topP: number
  frequencyPenalty: number
  presencePenalty: number
  contextWindow: number
  costPerInputToken: number
  costPerOutputToken: number
  enabled: boolean
  requireApproval: boolean
}

interface ModelFormProps {
  initialData?: Partial<ModelFormData>
  isLoading?: boolean
  error?: string
  onSubmit: (data: ModelFormData) => Promise<void>
  onCancel: () => void
  isEdit?: boolean
}

export function ModelForm({ 
  initialData, 
  isLoading = false, 
  error, 
  onSubmit, 
  onCancel,
  isEdit = false 
}: ModelFormProps) {
  const [step, setStep] = useState(1)
  const [availableModels, setAvailableModels] = useState<string[]>([])
  const [fetchingModels, setFetchingModels] = useState(false)
  const [formData, setFormData] = useState<ModelFormData>({
    name: '',
    provider: '',
    model: '',
    endpoint: '',
    apiKey: '',
    description: '',
    maxTokens: 4096,
    temperature: 0.7,
    topP: 1.0,
    frequencyPenalty: 0.0,
    presencePenalty: 0.0,
    contextWindow: 8192,
    costPerInputToken: 0.0,
    costPerOutputToken: 0.0,
    enabled: true,
    requireApproval: false,
    ...initialData
  })

  const [validationError, setValidationError] = useState('')

  // Update form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      setFormData(prev => ({ ...prev, ...initialData }))
      if (initialData.provider) {
        setStep(2) // Skip to model selection if we have provider
      }
    }
  }, [initialData])

  const handleProviderChange = (providerId: string) => {
    const provider = PROVIDERS.find(p => p.id === providerId)
    if (!provider) return

    setFormData(prev => ({
      ...prev,
      provider: providerId,
      endpoint: provider.endpoint,
      model: '', // Reset model when provider changes
      apiKey: '' // Reset API key when provider changes
    }))

    // Reset available models - will be fetched after credentials are provided
    setAvailableModels([])
    
    // Clear any validation errors when switching providers
    setValidationError('')

    // All providers now go to step 2 (credentials) first
    setStep(2)
  }

  const handleCredentialsSet = async () => {
    setValidationError('')
    
    // Validate based on provider type
    if (formData.provider === 'openai-compatible') {
      if (!formData.endpoint.trim()) {
        setValidationError('Endpoint is required')
        return
      }
      try {
        new URL(formData.endpoint)
      } catch {
        setValidationError('Invalid endpoint URL')
        return
      }
    } else {
      // OpenAI and Anthropic require API key
      if (!formData.apiKey.trim()) {
        setValidationError('API key is required')
        return
      }
    }

    // Try to fetch available models using the provided credentials
    await fetchAvailableModels()
    setStep(3)
  }

  const fetchAvailableModels = async () => {
    // For all providers, we need either endpoint or default endpoints
    const endpoint = formData.endpoint || 
      (formData.provider === 'openai' ? 'https://api.openai.com/v1' :
       formData.provider === 'anthropic' ? 'https://api.anthropic.com/v1' : '')

    if (!endpoint) return

    setFetchingModels(true)
    try {
      const response = await fetch(`/api/models/discover`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          endpoint: endpoint,
          provider: formData.provider,
          apiKey: formData.apiKey
        })
      })

      if (response.ok) {
        const data = await response.json()
        setAvailableModels(data.models || [])
        console.log(`Found ${data.models?.length || 0} models for ${formData.provider}`)
      } else {
        const errorData = await response.json()
        console.warn('Model discovery failed:', errorData.error)
        setAvailableModels([])
        
        if (response.status === 401) {
          setValidationError('Invalid API key or credentials')
        } else {
          setValidationError(`Failed to discover models: ${errorData.error || 'Unknown error'}`)
        }
        return
      }
    } catch (err) {
      console.warn('Failed to fetch models:', err)
      setAvailableModels([])
      setValidationError('Failed to connect to API endpoint. Please check the URL and try again.')
    } finally {
      setFetchingModels(false)
    }
  }

  const validateForm = () => {
    if (!formData.name.trim()) {
      setValidationError('Model name is required')
      return false
    }
    
    // Validate model name (DNS-compatible)
    const nameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/
    if (!nameRegex.test(formData.name)) {
      setValidationError('Model name must be lowercase alphanumeric with hyphens')
      return false
    }
    
    if (formData.name.length > 63) {
      setValidationError('Model name must be 63 characters or less')
      return false
    }

    if (!formData.provider) {
      setValidationError('Provider is required')
      return false
    }

    if (!formData.model.trim()) {
      setValidationError('Model identifier is required')
      return false
    }

    if (!formData.endpoint.trim()) {
      setValidationError('API endpoint is required')
      return false
    }

    // Validate endpoint URL
    try {
      new URL(formData.endpoint)
    } catch {
      setValidationError('Invalid endpoint URL')
      return false
    }

    setValidationError('')
    return true
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!validateForm()) {
      return
    }

    await onSubmit(formData)
  }

  const displayError = error || validationError

  const renderProviderStep = () => (
    <Card>
      <CardHeader>
        <CardTitle>Choose Provider</CardTitle>
        <CardDescription>
          Select your AI model provider
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4">
          {PROVIDERS.map((provider) => (
            <Button
              key={provider.id}
              variant="outline"
              className="justify-start h-16 text-left"
              onClick={() => handleProviderChange(provider.id)}
            >
              <div>
                <div className="font-semibold">{provider.name}</div>
                <div className="text-sm text-muted-foreground">
                  {provider.id === 'openai' && 'GPT-4, GPT-3.5, and more'}
                  {provider.id === 'anthropic' && 'Claude 3 models'}
                  {provider.id === 'openai-compatible' && 'Ollama, vLLM, and other compatible APIs'}
                </div>
              </div>
            </Button>
          ))}
        </div>
      </CardContent>
    </Card>
  )

  const renderCredentialsStep = () => {
    const provider = PROVIDERS.find(p => p.id === formData.provider)
    
    return (
      <Card>
        <CardHeader>
          <CardTitle>Configure {provider?.name}</CardTitle>
          <CardDescription>
            Enter your {provider?.name} credentials to discover available models
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* API Key field - required for OpenAI and Anthropic */}
          {(formData.provider === 'openai' || formData.provider === 'anthropic') && (
            <div className="space-y-2">
              <Label htmlFor="apiKey">API Key *</Label>
              <Input
                id="apiKey"
                type="password"
                value={formData.apiKey}
                onChange={(e) => setFormData(prev => ({ ...prev, apiKey: e.target.value }))}
                placeholder={
                  formData.provider === 'openai' ? 'sk-...' :
                  formData.provider === 'anthropic' ? 'sk-ant-...' : ''
                }
                className="font-mono"
              />
              <p className="text-sm text-muted-foreground">
                {formData.provider === 'openai' && 'Get your API key from platform.openai.com'}
                {formData.provider === 'anthropic' && 'Get your API key from console.anthropic.com'}
              </p>
            </div>
          )}

          {/* Optional endpoint override for OpenAI/Anthropic */}
          {(formData.provider === 'openai' || formData.provider === 'anthropic') && (
            <div className="space-y-2">
              <Label htmlFor="endpoint">Endpoint (Optional)</Label>
              <Input
                id="endpoint"
                value={formData.endpoint}
                onChange={(e) => setFormData(prev => ({ ...prev, endpoint: e.target.value }))}
                placeholder={
                  formData.provider === 'openai' ? 'https://api.openai.com/v1' :
                  formData.provider === 'anthropic' ? 'https://api.anthropic.com/v1' : ''
                }
                className="font-mono"
              />
              <p className="text-sm text-muted-foreground">
                Leave empty to use the default endpoint
              </p>
            </div>
          )}

          {/* Endpoint field - required for openai-compatible */}
          {formData.provider === 'openai-compatible' && (
            <>
              <div className="space-y-2">
                <Label htmlFor="endpoint">API Endpoint *</Label>
                <Input
                  id="endpoint"
                  value={formData.endpoint}
                  onChange={(e) => setFormData(prev => ({ ...prev, endpoint: e.target.value }))}
                  placeholder="http://localhost:11434/v1"
                  className="font-mono"
                />
                <p className="text-sm text-muted-foreground">
                  Common endpoints: Ollama (http://localhost:11434/v1), vLLM (/v1), LM Studio (/v1)
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="apiKey">API Key (Optional)</Label>
                <Input
                  id="apiKey"
                  type="password"
                  value={formData.apiKey}
                  onChange={(e) => setFormData(prev => ({ ...prev, apiKey: e.target.value }))}
                  placeholder="Leave empty if not required"
                  className="font-mono"
                />
                <p className="text-sm text-muted-foreground">
                  Most local providers don't require an API key
                </p>
              </div>
            </>
          )}
          
          {displayError && (
            <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{displayError}</AlertDescription>
          </Alert>
        )}

        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setStep(1)}>
            Back
          </Button>
          <Button onClick={handleCredentialsSet} disabled={fetchingModels}>
            {fetchingModels ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                Discovering Models...
              </>
            ) : (
              'Continue'
            )}
          </Button>
        </div>
      </CardContent>
    </Card>
  )}

  const handleModelSelect = (modelName: string) => {
    // Generate a default name from the model name
    const defaultName = modelName
      .toLowerCase()
      .replace(/[^a-z0-9-]/g, '-')  // Replace invalid chars with hyphens
      .replace(/-+/g, '-')          // Collapse multiple hyphens
      .replace(/^-|-$/g, '')        // Remove leading/trailing hyphens
      .slice(0, 63)                 // Kubernetes name limit
    
    setFormData(prev => ({ 
      ...prev, 
      model: modelName,
      name: prev.name || defaultName  // Only set if name is empty
    }))
  }

  const renderModelStep = () => (
    <Card>
      <CardHeader>
        <CardTitle>Select Model</CardTitle>
        <CardDescription>
          Choose from available models
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {availableModels.length > 0 ? (
          <>
            <div className="space-y-2">
              <Label htmlFor="model">Available Models ({availableModels.length})</Label>
              <Select 
                value={formData.model} 
                onValueChange={handleModelSelect}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a model" />
                </SelectTrigger>
                <SelectContent>
                  {availableModels.map((model) => (
                    <SelectItem key={model} value={model}>
                      {model}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="custom-model">Or enter custom model name</Label>
              <Input
                id="custom-model"
                value={formData.model}
                onChange={(e) => handleModelSelect(e.target.value)}
                placeholder="custom-model-name"
                className="font-mono"
              />
            </div>
          </>
        ) : (
          <div className="space-y-4">
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                No models were discovered from the API endpoint. Please enter a model name manually below.
              </AlertDescription>
            </Alert>
            <div className="space-y-2">
              <Label htmlFor="manual-model">Model Name *</Label>
              <Input
                id="manual-model"
                value={formData.model}
                onChange={(e) => handleModelSelect(e.target.value)}
                placeholder="Enter the exact model name (e.g., gpt-4, llama3:8b)"
                className="font-mono"
                required
              />
              <p className="text-sm text-muted-foreground">
                Enter the exact model name as expected by your API endpoint
              </p>
            </div>
          </div>
        )}

        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setStep(2)}>
            Back
          </Button>
          <Button onClick={() => setStep(4)} disabled={!formData.model.trim()}>
            Continue
          </Button>
        </div>
      </CardContent>
    </Card>
  )

  const renderDetailsStep = () => (
    <form onSubmit={handleSubmit} className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Model Details</CardTitle>
          <CardDescription>
            Configure the final details for your model
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Model Name *</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
              placeholder="my-model"
              className="font-mono"
              disabled={isEdit || isLoading}
              required
            />
            <p className="text-sm text-muted-foreground">
              Must be lowercase alphanumeric with hyphens, max 63 characters
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              placeholder="Description of this model's capabilities..."
              rows={3}
            />
          </div>
        </CardContent>
      </Card>

      {/* Error Display */}
      {displayError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{displayError}</AlertDescription>
        </Alert>
      )}

      {/* Actions */}
      <div className="flex justify-between">
        <Button variant="outline" onClick={() => setStep(3)}>
          Back
        </Button>
        <div className="flex gap-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" disabled={isLoading}>
            {isLoading ? 'Creating...' : 'Create Model'}
          </Button>
        </div>
      </div>
    </form>
  )

  if (isEdit) {
    return renderDetailsStep() // For editing, show full form
  }

  return (
    <div className="space-y-6">
      {step === 1 && renderProviderStep()}
      {step === 2 && renderCredentialsStep()}
      {step === 3 && renderModelStep()}
      {step === 4 && renderDetailsStep()}
    </div>
  )
}