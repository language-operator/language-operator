'use client'

import { useState, useEffect } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  Brain, Zap, DollarSign, AlertCircle, Server, RefreshCw,
  Settings, Shield, Activity, Timer, Globe, Network,
  Database, BarChart3, Plus, Minus
} from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'

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

// Types for enterprise features
export type CachingBackend = 'memory' | 'redis' | 'memcached'
export type LoadBalancingStrategy = 'round-robin' | 'least-connections' | 'random' | 'weighted' | 'latency-based'
export type LogLevel = 'debug' | 'info' | 'warn' | 'error'

export interface ModelFormData {
  // Basic Information (existing fields)
  name: string
  provider: string
  model: string
  endpoint: string
  apiKey: string
  description: string
  
  // Existing configuration fields
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
  
  // Enterprise Features (optional - only used in edit mode)
  
  // Advanced Configuration
  stopSequences?: string[]
  additionalParameters?: Record<string, string>
  
  // Caching Configuration
  cachingEnabled?: boolean
  cachingBackend?: CachingBackend
  cachingTtl?: string // e.g., "5m", "1h"
  cachingMaxSize?: number // MB
  
  // Cost Tracking
  currency?: string
  
  // Load Balancing
  loadBalancingEnabled?: boolean
  loadBalancingStrategy?: LoadBalancingStrategy
  loadBalancingEndpoints?: Array<{
    url: string
    weight: number
    priority: number
    region?: string
  }>
  healthCheckEnabled?: boolean
  healthCheckInterval?: string // e.g., "30s"
  healthCheckTimeout?: string // e.g., "5s"
  healthCheckHealthyThreshold?: number
  healthCheckUnhealthyThreshold?: number
  
  // High Availability (Fallbacks)
  fallbacks?: Array<{
    modelRef: string
    conditions: string[]
  }>
  
  // Observability
  logLevel?: LogLevel
  logRequests?: boolean
  logResponses?: boolean
  metricsEnabled?: boolean
  tracingEnabled?: boolean
  
  // Rate Limiting
  requestsPerMinute?: number
  tokensPerMinute?: number
  concurrentRequests?: number
  
  // Multi-Region
  regions?: Array<{
    name: string
    endpoint?: string
    enabled: boolean
    priority?: number
  }>
  
  // Retry Policy
  retryMaxAttempts?: number
  retryInitialBackoff?: string // e.g., "1s"
  retryMaxBackoff?: string // e.g., "30s" 
  retryBackoffMultiplier?: number
  retryableStatusCodes?: number[]
  
  // Security (Network Egress Rules)
  egress?: Array<{
    description: string
    to: {
      dns?: string[]
      cidr?: string
    }
    ports?: number[]
  }>
  
  // General Settings
  timeout?: string // e.g., "5m"
}

// Helper functions for managing form arrays (used in edit mode only)
const addStopSequence = (sequences: string[], newSequence: string) => 
  newSequence.trim() ? [...sequences, newSequence.trim()] : sequences
const removeStopSequence = (sequences: string[], index: number) => sequences.filter((_, i) => i !== index)

const addEgressRule = (egress: any[], newRule: any) => [...egress, newRule]
const removeEgressRule = (egress: any[], index: number) => egress.filter((_, i) => i !== index)

const addEndpoint = (endpoints: any[], newEndpoint: any) => [...endpoints, newEndpoint]
const removeEndpoint = (endpoints: any[], index: number) => endpoints.filter((_, i) => i !== index)

interface ModelFormProps {
  initialData?: Partial<ModelFormData>
  isLoading?: boolean
  error?: string
  onSubmit: (data: ModelFormData) => Promise<void>
  onCancel: () => void
  isEdit?: boolean
  clusterName?: string  // Add clusterName for cluster-scoped operations
}

export function ModelForm({ 
  initialData, 
  isLoading = false, 
  error, 
  onSubmit, 
  onCancel,
  isEdit = false,
  clusterName
}: ModelFormProps) {
  const [step, setStep] = useState(1)
  const [currentTab, setCurrentTab] = useState('basic') // Only used in edit mode
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
    
    // Enterprise feature defaults (only used in edit mode)
    stopSequences: [],
    additionalParameters: {},
    cachingEnabled: false,
    cachingBackend: 'memory',
    cachingTtl: '5m',
    cachingMaxSize: 100,
    currency: 'USD',
    loadBalancingEnabled: false,
    loadBalancingStrategy: 'round-robin',
    loadBalancingEndpoints: [],
    healthCheckEnabled: true,
    healthCheckInterval: '30s',
    healthCheckTimeout: '5s',
    healthCheckHealthyThreshold: 2,
    healthCheckUnhealthyThreshold: 3,
    fallbacks: [],
    logLevel: 'info',
    logRequests: true,
    logResponses: false,
    metricsEnabled: true,
    tracingEnabled: false,
    regions: [],
    retryMaxAttempts: 3,
    retryInitialBackoff: '1s',
    retryMaxBackoff: '30s',
    retryBackoffMultiplier: 2,
    retryableStatusCodes: [429, 500, 502, 503, 504],
    egress: [],
    timeout: '5m',
    
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
      // Use cluster-scoped endpoint if clusterName is provided, otherwise use legacy endpoint
      const apiEndpoint = clusterName 
        ? `/api/clusters/${clusterName}/models/discover`
        : `/api/models/discover`
      
      const response = await fetchWithOrganization(apiEndpoint, {
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
            {isLoading ? (isEdit ? 'Updating...' : 'Creating...') : (isEdit ? 'Update Model' : 'Create Model')}
          </Button>
        </div>
      </div>
    </form>
  )

  // Tab rendering functions for edit mode
  function renderBasicTab() {
    return (
      <TabsContent value="basic" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Brain className="h-5 w-5" />
              Basic Information
            </CardTitle>
            <CardDescription>
              Configure the essential model information
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="name">Model Name</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                  className="font-mono"
                  disabled={true} // Name cannot be changed in edit mode
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="provider">Provider</Label>
                <Select 
                  value={formData.provider} 
                  onValueChange={(value) => setFormData(prev => ({ ...prev, provider: value }))}
                >
                  <SelectTrigger>
                    <SelectValue />
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
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="model">Model Identifier</Label>
                <Input
                  id="model"
                  value={formData.model}
                  onChange={(e) => setFormData(prev => ({ ...prev, model: e.target.value }))}
                  className="font-mono"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="endpoint">API Endpoint</Label>
                <Input
                  id="endpoint"
                  value={formData.endpoint}
                  onChange={(e) => setFormData(prev => ({ ...prev, endpoint: e.target.value }))}
                  className="font-mono"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                rows={3}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="timeout">Request Timeout</Label>
                <Input
                  id="timeout"
                  value={formData.timeout || '5m'}
                  onChange={(e) => setFormData(prev => ({ ...prev, timeout: e.target.value }))}
                  placeholder="5m"
                />
                <p className="text-sm text-muted-foreground">
                  Duration format: 30s, 5m, 1h
                </p>
              </div>
              <div className="flex items-center space-x-2 pt-6">
                <Switch
                  id="enabled"
                  checked={formData.enabled}
                  onCheckedChange={(checked) => setFormData(prev => ({ ...prev, enabled: checked }))}
                />
                <Label htmlFor="enabled">Model Enabled</Label>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    )
  }

  function renderConfigTab() {
    return (
      <TabsContent value="config" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Model Configuration
            </CardTitle>
            <CardDescription>
              Advanced model parameters and behavior settings
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
                  onChange={(e) => setFormData(prev => ({ ...prev, maxTokens: parseInt(e.target.value) || 0 }))}
                  min="1"
                  max="128000"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="temperature">Temperature</Label>
                <Input
                  id="temperature"
                  type="number"
                  step="0.1"
                  value={formData.temperature}
                  onChange={(e) => setFormData(prev => ({ ...prev, temperature: parseFloat(e.target.value) || 0 }))}
                  min="0"
                  max="2"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="topP">Top P</Label>
                <Input
                  id="topP"
                  type="number"
                  step="0.1"
                  value={formData.topP}
                  onChange={(e) => setFormData(prev => ({ ...prev, topP: parseFloat(e.target.value) || 0 }))}
                  min="0"
                  max="1"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="contextWindow">Context Window</Label>
                <Input
                  id="contextWindow"
                  type="number"
                  value={formData.contextWindow}
                  onChange={(e) => setFormData(prev => ({ ...prev, contextWindow: parseInt(e.target.value) || 0 }))}
                />
              </div>
            </div>
            <Separator />
            <div className="space-y-2">
              <Label>Stop Sequences</Label>
              <div className="space-y-2">
                {(formData.stopSequences || []).map((sequence, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      value={sequence}
                      onChange={(e) => {
                        const newSequences = [...(formData.stopSequences || [])]
                        newSequences[index] = e.target.value
                        setFormData(prev => ({ ...prev, stopSequences: newSequences }))
                      }}
                      placeholder="Stop sequence"
                      className="font-mono"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        const newSequences = removeStopSequence(formData.stopSequences || [], index)
                        setFormData(prev => ({ ...prev, stopSequences: newSequences }))
                      }}
                    >
                      <Minus className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    const newSequences = addStopSequence(formData.stopSequences || [], "")
                    setFormData(prev => ({ ...prev, stopSequences: newSequences }))
                  }}
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Add Stop Sequence
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    )
  }

  function renderCachingTab() {
    return (
      <TabsContent value="caching" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              Response Caching
            </CardTitle>
            <CardDescription>
              Configure response caching to improve performance and reduce costs
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center space-x-2">
              <Switch
                id="caching-enabled"
                checked={formData.cachingEnabled || false}
                onCheckedChange={(checked) => setFormData(prev => ({ ...prev, cachingEnabled: checked }))}
              />
              <Label htmlFor="caching-enabled">Enable Response Caching</Label>
            </div>

            {formData.cachingEnabled && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="caching-backend">Backend</Label>
                  <Select 
                    value={formData.cachingBackend || 'memory'} 
                    onValueChange={(value) => setFormData(prev => ({ ...prev, cachingBackend: value as CachingBackend }))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="memory">Memory</SelectItem>
                      <SelectItem value="redis">Redis</SelectItem>
                      <SelectItem value="memcached">Memcached</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="caching-ttl">TTL (Time to Live)</Label>
                    <Input
                      id="caching-ttl"
                      value={formData.cachingTtl || '5m'}
                      onChange={(e) => setFormData(prev => ({ ...prev, cachingTtl: e.target.value }))}
                      placeholder="5m"
                    />
                    <p className="text-sm text-muted-foreground">
                      Duration format: 30s, 5m, 1h
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="caching-maxSize">Max Size (MB)</Label>
                    <Input
                      id="caching-maxSize"
                      type="number"
                      value={formData.cachingMaxSize || 100}
                      onChange={(e) => setFormData(prev => ({ ...prev, cachingMaxSize: parseInt(e.target.value) || 0 }))}
                      min="1"
                    />
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </TabsContent>
    )
  }

  function renderLoadBalancingTab() {
    return (
      <TabsContent value="loadbalancing" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" />
              Load Balancing
            </CardTitle>
            <CardDescription>
              Configure multiple endpoints and load balancing strategies
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="text-center py-8 text-muted-foreground">
              Load balancing configuration will be implemented in the next iteration.
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    )
  }

  function renderSecurityTab() {
    return (
      <TabsContent value="security" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Network Security
            </CardTitle>
            <CardDescription>
              Configure network egress rules and security policies
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="text-center py-8 text-muted-foreground">
              Network security configuration will be implemented in the next iteration.
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    )
  }

  function renderObservabilityTab() {
    return (
      <TabsContent value="observability" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Observability
            </CardTitle>
            <CardDescription>
              Configure logging, metrics, and tracing for monitoring
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-4">
              <h4 className="text-sm font-medium">Logging</h4>
              <div className="space-y-2">
                <Label htmlFor="log-level">Log Level</Label>
                <Select 
                  value={formData.logLevel || 'info'} 
                  onValueChange={(value) => setFormData(prev => ({ ...prev, logLevel: value as LogLevel }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="debug">Debug</SelectItem>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warn">Warn</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-3">
                <div className="flex items-center space-x-2">
                  <Switch
                    id="log-requests"
                    checked={formData.logRequests !== false}
                    onCheckedChange={(checked) => setFormData(prev => ({ ...prev, logRequests: checked }))}
                  />
                  <Label htmlFor="log-requests">Log Requests</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <Switch
                    id="log-responses"
                    checked={formData.logResponses || false}
                    onCheckedChange={(checked) => setFormData(prev => ({ ...prev, logResponses: checked }))}
                  />
                  <Label htmlFor="log-responses">Log Responses</Label>
                </div>
              </div>
            </div>

            <Separator />

            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <Switch
                  id="metrics-enabled"
                  checked={formData.metricsEnabled !== false}
                  onCheckedChange={(checked) => setFormData(prev => ({ ...prev, metricsEnabled: checked }))}
                />
                <Label htmlFor="metrics-enabled">Enable Metrics Collection</Label>
              </div>
            </div>

            <Separator />

            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <Switch
                  id="tracing-enabled"
                  checked={formData.tracingEnabled || false}
                  onCheckedChange={(checked) => setFormData(prev => ({ ...prev, tracingEnabled: checked }))}
                />
                <Label htmlFor="tracing-enabled">Enable Distributed Tracing</Label>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    )
  }

  function renderAdvancedTab() {
    return (
      <TabsContent value="advanced" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Timer className="h-5 w-5" />
              Rate Limiting
            </CardTitle>
            <CardDescription>
              Configure rate limits to prevent abuse
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label htmlFor="requestsPerMinute">Requests per Minute</Label>
                <Input
                  id="requestsPerMinute"
                  type="number"
                  value={formData.requestsPerMinute || ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, requestsPerMinute: e.target.value ? parseInt(e.target.value) : undefined }))}
                  placeholder="60"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tokensPerMinute">Tokens per Minute</Label>
                <Input
                  id="tokensPerMinute"
                  type="number"
                  value={formData.tokensPerMinute || ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, tokensPerMinute: e.target.value ? parseInt(e.target.value) : undefined }))}
                  placeholder="10000"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="concurrentRequests">Concurrent Requests</Label>
                <Input
                  id="concurrentRequests"
                  type="number"
                  value={formData.concurrentRequests || ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, concurrentRequests: e.target.value ? parseInt(e.target.value) : undefined }))}
                  placeholder="10"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <RefreshCw className="h-5 w-5" />
              Retry Policy
            </CardTitle>
            <CardDescription>
              Configure retry behavior for failed requests
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="maxAttempts">Max Attempts</Label>
                <Input
                  id="maxAttempts"
                  type="number"
                  value={formData.retryMaxAttempts || 3}
                  onChange={(e) => setFormData(prev => ({ ...prev, retryMaxAttempts: parseInt(e.target.value) || 0 }))}
                  min="0"
                  max="10"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="backoffMultiplier">Backoff Multiplier</Label>
                <Input
                  id="backoffMultiplier"
                  type="number"
                  step="0.1"
                  value={formData.retryBackoffMultiplier || 2}
                  onChange={(e) => setFormData(prev => ({ ...prev, retryBackoffMultiplier: parseFloat(e.target.value) || 0 }))}
                  min="1"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="initialBackoff">Initial Backoff</Label>
                <Input
                  id="initialBackoff"
                  value={formData.retryInitialBackoff || '1s'}
                  onChange={(e) => setFormData(prev => ({ ...prev, retryInitialBackoff: e.target.value }))}
                  placeholder="1s"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="maxBackoff">Max Backoff</Label>
                <Input
                  id="maxBackoff"
                  value={formData.retryMaxBackoff || '30s'}
                  onChange={(e) => setFormData(prev => ({ ...prev, retryMaxBackoff: e.target.value }))}
                  placeholder="30s"
                />
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    )
  }

  // Enterprise edit form with tabbed interface
  function renderEditForm() {
    return (
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Header */}
        <div>
          <h3 className="text-lg font-medium">Edit Model</h3>
          <p className="text-sm text-muted-foreground">
            Update model configuration and enterprise features
          </p>
        </div>

        {/* Error Display */}
        {displayError && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{displayError}</AlertDescription>
          </Alert>
        )}

        {/* Tab-based Form */}
        <Tabs value={currentTab} onValueChange={setCurrentTab} className="w-full">
          <TabsList className="grid w-full grid-cols-7">
            <TabsTrigger value="basic" className="flex items-center gap-2">
              <Brain className="h-4 w-4" />
              Basic
            </TabsTrigger>
            <TabsTrigger value="config" className="flex items-center gap-2">
              <Settings className="h-4 w-4" />
              Config
            </TabsTrigger>
            <TabsTrigger value="caching" className="flex items-center gap-2">
              <Database className="h-4 w-4" />
              Cache
            </TabsTrigger>
            <TabsTrigger value="loadbalancing" className="flex items-center gap-2">
              <Server className="h-4 w-4" />
              Balance
            </TabsTrigger>
            <TabsTrigger value="security" className="flex items-center gap-2">
              <Shield className="h-4 w-4" />
              Security
            </TabsTrigger>
            <TabsTrigger value="observability" className="flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Observe
            </TabsTrigger>
            <TabsTrigger value="advanced" className="flex items-center gap-2">
              <Globe className="h-4 w-4" />
              Advanced
            </TabsTrigger>
          </TabsList>

          {renderBasicTab()}
          {renderConfigTab()}
          {renderCachingTab()}
          {renderLoadBalancingTab()}
          {renderSecurityTab()}
          {renderObservabilityTab()}
          {renderAdvancedTab()}
        </Tabs>

        {/* Form Actions */}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" disabled={isLoading}>
            {isLoading ? 'Updating...' : 'Update Model'}
          </Button>
        </div>
      </form>
    )
  }

  // Conditional rendering: step-based for creation, tabbed for edit
  if (isEdit) {
    return renderEditForm()
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