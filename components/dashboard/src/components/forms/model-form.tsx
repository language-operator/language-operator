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
import { Brain, Zap, DollarSign, AlertCircle, Server } from 'lucide-react'

const PROVIDERS = [
  { id: 'openai', name: 'OpenAI', endpoint: 'https://api.openai.com/v1' },
  { id: 'anthropic', name: 'Anthropic', endpoint: 'https://api.anthropic.com/v1' },
  { id: 'ollama', name: 'Ollama', endpoint: 'http://localhost:11434/v1' },
  { id: 'openai-compatible', name: 'OpenAI Compatible', endpoint: '' },
  { id: 'custom', name: 'Custom Provider', endpoint: '' }
]

const SAMPLE_MODELS = {
  openai: ['gpt-4', 'gpt-4-turbo', 'gpt-3.5-turbo'],
  anthropic: ['claude-3-opus', 'claude-3-sonnet', 'claude-3-haiku'],
  ollama: ['llama3', 'codellama', 'mistral'],
  'openai-compatible': ['qwen3-coder:30b', 'llama3:8b', 'phi3:14b'],
  custom: []
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
    }
  }, [initialData])

  const handleInputChange = (field: keyof ModelFormData, value: string | number | boolean) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }))
    
    // Auto-set endpoint when provider changes
    if (field === 'provider' && typeof value === 'string') {
      const provider = PROVIDERS.find(p => p.id === value)
      if (provider?.endpoint) {
        setFormData(prev => ({
          ...prev,
          endpoint: provider.endpoint
        }))
      }
    }
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
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

  const selectedProvider = PROVIDERS.find(p => p.id === formData.provider)
  const availableModels = formData.provider ? SAMPLE_MODELS[formData.provider as keyof typeof SAMPLE_MODELS] || [] : []
  const displayError = error || validationError

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Brain className="h-5 w-5" />
            <span>Basic Information</span>
          </CardTitle>
          <CardDescription>
            Configure the basic settings for your language model
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Model Name *</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="my-model"
              className="font-mono"
              disabled={isEdit || isLoading}
              required
            />
            {isEdit && (
              <p className="text-sm text-muted-foreground">
                Name cannot be changed after creation
              </p>
            )}
            {!isEdit && (
              <p className="text-sm text-muted-foreground">
                Must be lowercase alphanumeric with hyphens, max 63 characters
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              placeholder="Description of this model's capabilities..."
              rows={3}
              disabled={isLoading}
            />
          </div>
        </CardContent>
      </Card>

      {/* Provider Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Server className="h-5 w-5" />
            <span>Provider Configuration</span>
          </CardTitle>
          <CardDescription>
            Configure the model provider and endpoint
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="provider">Provider *</Label>
            <Select 
              value={formData.provider} 
              onValueChange={(value) => handleInputChange('provider', value)}
              disabled={isLoading}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a provider" />
              </SelectTrigger>
              <SelectContent>
                {PROVIDERS.map((provider) => (
                  <SelectItem key={provider.id} value={provider.id}>
                    {provider.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="model">Model Identifier *</Label>
            <div className="flex space-x-2">
              <Input
                id="model"
                value={formData.model}
                onChange={(e) => handleInputChange('model', e.target.value)}
                placeholder="gpt-4"
                className="font-mono flex-1"
                disabled={isLoading}
                required
              />
              {availableModels.length > 0 && (
                <Select 
                  value={formData.model} 
                  onValueChange={(value) => handleInputChange('model', value)}
                  disabled={isLoading}
                >
                  <SelectTrigger className="w-32">
                    <SelectValue placeholder="Common" />
                  </SelectTrigger>
                  <SelectContent>
                    {availableModels.map((model) => (
                      <SelectItem key={model} value={model}>
                        {model}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="endpoint">API Endpoint *</Label>
            <Input
              id="endpoint"
              value={formData.endpoint}
              onChange={(e) => handleInputChange('endpoint', e.target.value)}
              placeholder="https://api.openai.com/v1"
              className="font-mono"
              disabled={isLoading}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="apiKey">API Key</Label>
            <Input
              id="apiKey"
              type="password"
              value={formData.apiKey}
              onChange={(e) => handleInputChange('apiKey', e.target.value)}
              placeholder="sk-..."
              className="font-mono"
              disabled={isLoading}
            />
            <p className="text-sm text-muted-foreground">
              Leave empty to use organization-level API key
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Model Parameters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Zap className="h-5 w-5" />
            <span>Model Parameters</span>
          </CardTitle>
          <CardDescription>
            Configure generation parameters and limits
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="maxTokens">Max Tokens</Label>
              <Input
                id="maxTokens"
                type="number"
                value={formData.maxTokens}
                onChange={(e) => handleInputChange('maxTokens', parseInt(e.target.value) || 0)}
                min={1}
                max={32768}
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="contextWindow">Context Window</Label>
              <Input
                id="contextWindow"
                type="number"
                value={formData.contextWindow}
                onChange={(e) => handleInputChange('contextWindow', parseInt(e.target.value) || 0)}
                min={1}
                max={1000000}
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="temperature">Temperature</Label>
              <Input
                id="temperature"
                type="number"
                step="0.1"
                value={formData.temperature}
                onChange={(e) => handleInputChange('temperature', parseFloat(e.target.value) || 0)}
                min={0}
                max={2}
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="topP">Top P</Label>
              <Input
                id="topP"
                type="number"
                step="0.1"
                value={formData.topP}
                onChange={(e) => handleInputChange('topP', parseFloat(e.target.value) || 0)}
                min={0}
                max={1}
                disabled={isLoading}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Cost Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <DollarSign className="h-5 w-5" />
            <span>Cost Configuration</span>
          </CardTitle>
          <CardDescription>
            Configure pricing for usage tracking
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="costPerInputToken">Cost per Input Token (USD)</Label>
              <Input
                id="costPerInputToken"
                type="number"
                step="0.000001"
                value={formData.costPerInputToken}
                onChange={(e) => handleInputChange('costPerInputToken', parseFloat(e.target.value) || 0)}
                min={0}
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="costPerOutputToken">Cost per Output Token (USD)</Label>
              <Input
                id="costPerOutputToken"
                type="number"
                step="0.000001"
                value={formData.costPerOutputToken}
                onChange={(e) => handleInputChange('costPerOutputToken', parseFloat(e.target.value) || 0)}
                min={0}
                disabled={isLoading}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Settings</CardTitle>
          <CardDescription>
            Additional model configuration
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Enable Model</Label>
              <p className="text-sm text-muted-foreground">
                Allow this model to be used by agents
              </p>
            </div>
            <Switch
              checked={formData.enabled}
              onCheckedChange={(checked) => handleInputChange('enabled', checked)}
              disabled={isLoading}
            />
          </div>

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Require Approval</Label>
              <p className="text-sm text-muted-foreground">
                Require admin approval before using this model
              </p>
            </div>
            <Switch
              checked={formData.requireApproval}
              onCheckedChange={(checked) => handleInputChange('requireApproval', checked)}
              disabled={isLoading}
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
      <div className="flex justify-end space-x-4">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isLoading}
        >
          Cancel
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading ? (isEdit ? 'Updating...' : 'Creating...') : (isEdit ? 'Update Model' : 'Create Model')}
        </Button>
      </div>
    </form>
  )
}