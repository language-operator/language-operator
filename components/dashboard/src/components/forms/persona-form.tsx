'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Users, Brain, MessageSquare, AlertCircle, Target, BookOpen, Database, Shield, Settings, Workflow, FileText, Share2 } from 'lucide-react'


const PERSONALITY_TRAITS = [
  { id: 'professional', name: 'Professional', description: 'Formal, business-oriented tone' },
  { id: 'friendly', name: 'Friendly', description: 'Warm, approachable, and conversational' },
  { id: 'concise', name: 'Concise', description: 'Brief, to-the-point responses' },
  { id: 'detailed', name: 'Detailed', description: 'Thorough, comprehensive explanations' },
  { id: 'creative', name: 'Creative', description: 'Imaginative and innovative thinking' },
  { id: 'analytical', name: 'Analytical', description: 'Logical, data-driven approach' },
  { id: 'empathetic', name: 'Empathetic', description: 'Understanding and supportive' },
  { id: 'authoritative', name: 'Authoritative', description: 'Confident, expert knowledge' }
]

const SAMPLE_PERSONAS = {
  assistant: {
    systemPrompt: 'You are a helpful assistant that provides clear, accurate, and useful information to help users accomplish their goals.',
    traits: ['friendly', 'professional'],
    examples: [
      { input: 'How do I reset my password?', output: 'I\'d be happy to help you reset your password. Here are the steps...' },
      { input: 'What\'s the weather like?', output: 'I\'d be glad to help with weather information. Could you please specify your location?' }
    ]
  },
  developer: {
    systemPrompt: 'You are an experienced software developer who provides practical coding solutions, best practices, and technical guidance.',
    traits: ['analytical', 'detailed', 'professional'],
    examples: [
      { input: 'How do I handle errors in React?', output: 'Here are the main approaches for error handling in React applications...' },
      { input: 'What\'s the best database for my project?', output: 'The choice depends on your specific requirements. Let me break down the options...' }
    ]
  },
  analyst: {
    systemPrompt: 'You are a data analyst who helps interpret data, identify patterns, and provide actionable insights.',
    traits: ['analytical', 'detailed', 'professional'],
    examples: [
      { input: 'What does this trend mean?', output: 'Looking at this data trend, I can see several key patterns that suggest...' },
      { input: 'How should I visualize this data?', output: 'Based on your data type and intended audience, I recommend...' }
    ]
  }
}

// Knowledge source types from CRD
export type KnowledgeSourceType = 'url' | 'document' | 'database' | 'api' | 'vector-store'

// Tool usage strategy from CRD  
export type ToolUsageStrategy = 'conservative' | 'balanced' | 'aggressive' | 'minimal'

// Response format types from CRD
export type ResponseFormatType = 'text' | 'markdown' | 'json' | 'structured' | 'list' | 'table'

export interface PersonaFormData {
  name: string
  displayName: string // Required by CRD
  description: string // Required by CRD
  systemPrompt: string // Required by CRD
  traits: string[] // UI-only, maps to optional tone field
  tone: string // Optional by CRD
  language: string // Optional by CRD  
  version: string // Optional by CRD
  capabilities: string[] // Optional by CRD
  limitations: string[] // Optional by CRD
  instructions: string[] // Optional by CRD
  examples: Array<{input: string, output: string, context?: string, tags?: string[]}>
  
  // Missing CRD features
  knowledgeSources: Array<{
    name: string
    type: KnowledgeSourceType
    enabled: boolean
    url?: string
    query?: string
    priority: number
    secretRef?: {
      name: string
      namespace?: string
      key: string
    }
  }>
  
  constraints: {
    maxResponseTokens?: number
    maxToolCalls?: number
    maxKnowledgeQueries?: number
    responseTimeout?: string
    requireDocumentation: boolean
    blockedTopics: string[]
    allowedDomains: string[]
  }
  
  rules: Array<{
    name: string
    condition: string
    action: string
    description?: string
    enabled: boolean
    priority: number
  }>
  
  responseFormat: {
    type: ResponseFormatType
    includeConfidence: boolean
    includeSources: boolean
    maxLength?: number
    template?: string
    schema?: string
  }
  
  toolPreferences: {
    strategy: ToolUsageStrategy
    alwaysConfirm: boolean
    explainToolUse: boolean
    preferredTools: string[]
    avoidTools: string[]
  }
  
  parentPersona?: {
    name: string
    namespace?: string
  }
}

interface PersonaFormProps {
  initialData?: Partial<PersonaFormData>
  isLoading?: boolean
  error?: string
  onSubmit: (data: PersonaFormData) => Promise<void>
  onCancel: () => void
  isEdit?: boolean
}

export function PersonaForm({ 
  initialData, 
  isLoading = false, 
  error, 
  onSubmit, 
  onCancel,
  isEdit = false 
}: PersonaFormProps) {
  const [formData, setFormData] = useState<PersonaFormData>({
    name: '',
    displayName: '',
    description: '',
    systemPrompt: '',
    traits: [],
    tone: '',
    language: '',
    version: '',
    capabilities: [],
    limitations: [],
    instructions: [],
    examples: [
      { input: '', output: '', context: '', tags: [] },
      { input: '', output: '', context: '', tags: [] }
    ],
    // New CRD features with defaults
    knowledgeSources: [],
    constraints: {
      requireDocumentation: false,
      blockedTopics: [],
      allowedDomains: []
    },
    rules: [],
    responseFormat: {
      type: 'text',
      includeConfidence: false,
      includeSources: false
    },
    toolPreferences: {
      strategy: 'balanced',
      alwaysConfirm: false,
      explainToolUse: true,
      preferredTools: [],
      avoidTools: []
    },
    ...initialData
  })

  const [validationError, setValidationError] = useState('')

  // Update form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      setFormData(prev => ({ ...prev, ...initialData }))
    }
  }, [initialData])

  // Helper function to validate and sanitize numeric inputs
  const validateNumericInput = (value: string, min: number, max: number): number | undefined => {
    if (!value.trim()) return undefined
    
    const num = parseInt(value, 10)
    
    // Check for invalid number or extreme values
    if (isNaN(num) || num < min || num > max) {
      return undefined
    }
    
    return num
  }

  const handleInputChange = (field: keyof PersonaFormData, value: any) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }))
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
    }
  }

  // Specialized handler for constraints with validation
  const handleConstraintNumericChange = (field: 'maxResponseTokens' | 'maxToolCalls' | 'maxKnowledgeQueries', value: string) => {
    // Store the raw value for immediate feedback
    let numericValue: number | undefined = undefined
    
    if (value.trim()) {
      const parsed = parseInt(value, 10)
      if (!isNaN(parsed)) {
        numericValue = parsed
      }
    }
    
    setFormData(prev => ({
      ...prev,
      constraints: {
        ...prev.constraints,
        [field]: numericValue
      }
    }))
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
    }
  }

  // Specialized handler for response format max length
  const handleResponseFormatNumericChange = (field: 'maxLength', value: string) => {
    let numericValue: number | undefined = undefined
    
    if (value.trim()) {
      const parsed = parseInt(value, 10)
      if (!isNaN(parsed)) {
        numericValue = parsed
      }
    }
    
    setFormData(prev => ({
      ...prev,
      responseFormat: {
        ...prev.responseFormat,
        [field]: numericValue
      }
    }))
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
    }
  }


  const updateExample = (index: number, field: 'input' | 'output', value: string) => {
    setFormData(prev => ({
      ...prev,
      examples: prev.examples.map((ex, i) => 
        i === index ? { ...ex, [field]: value } : ex
      )
    }))
  }

  const addExample = () => {
    setFormData(prev => ({
      ...prev,
      examples: [...prev.examples, { input: '', output: '', context: '', tags: [] }]
    }))
  }

  const removeExample = (index: number) => {
    if (formData.examples.length > 1) {
      setFormData(prev => ({
        ...prev,
        examples: prev.examples.filter((_, i) => i !== index)
      }))
    }
  }

  const updateExampleContext = (index: number, context: string) => {
    setFormData(prev => ({
      ...prev,
      examples: prev.examples.map((ex, i) => 
        i === index ? { ...ex, context } : ex
      )
    }))
  }

  const updateExampleTags = (index: number, tags: string[]) => {
    setFormData(prev => ({
      ...prev,
      examples: prev.examples.map((ex, i) => 
        i === index ? { ...ex, tags } : ex
      )
    }))
  }

  const addCapability = () => {
    setFormData(prev => ({
      ...prev,
      capabilities: [...prev.capabilities, '']
    }))
  }

  const removeCapability = (index: number) => {
    setFormData(prev => ({
      ...prev,
      capabilities: prev.capabilities.filter((_, i) => i !== index)
    }))
  }

  const updateCapability = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      capabilities: prev.capabilities.map((cap, i) => i === index ? value : cap)
    }))
  }

  const addLimitation = () => {
    setFormData(prev => ({
      ...prev,
      limitations: [...prev.limitations, '']
    }))
  }

  const removeLimitation = (index: number) => {
    setFormData(prev => ({
      ...prev,
      limitations: prev.limitations.filter((_, i) => i !== index)
    }))
  }

  const updateLimitation = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      limitations: prev.limitations.map((lim, i) => i === index ? value : lim)
    }))
  }

  const addInstruction = () => {
    setFormData(prev => ({
      ...prev,
      instructions: [...prev.instructions, '']
    }))
  }

  const removeInstruction = (index: number) => {
    setFormData(prev => ({
      ...prev,
      instructions: prev.instructions.filter((_, i) => i !== index)
    }))
  }

  const updateInstruction = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      instructions: prev.instructions.map((inst, i) => i === index ? value : inst)
    }))
  }

  // Knowledge Sources helpers
  const addKnowledgeSource = () => {
    setFormData(prev => ({
      ...prev,
      knowledgeSources: [...prev.knowledgeSources, {
        name: '',
        type: 'url',
        enabled: true,
        priority: 100
      }]
    }))
  }

  const removeKnowledgeSource = (index: number) => {
    setFormData(prev => ({
      ...prev,
      knowledgeSources: prev.knowledgeSources.filter((_, i) => i !== index)
    }))
  }

  const updateKnowledgeSource = (index: number, field: string, value: any) => {
    setFormData(prev => ({
      ...prev,
      knowledgeSources: prev.knowledgeSources.map((ks, i) => 
        i === index ? { ...ks, [field]: value } : ks
      )
    }))
  }

  // Rules helpers
  const addRule = () => {
    setFormData(prev => ({
      ...prev,
      rules: [...prev.rules, {
        name: '',
        condition: '',
        action: '',
        description: '',
        enabled: true,
        priority: 100
      }]
    }))
  }

  const removeRule = (index: number) => {
    setFormData(prev => ({
      ...prev,
      rules: prev.rules.filter((_, i) => i !== index)
    }))
  }

  const updateRule = (index: number, field: string, value: any) => {
    setFormData(prev => ({
      ...prev,
      rules: prev.rules.map((rule, i) => 
        i === index ? { ...rule, [field]: value } : rule
      )
    }))
  }

  // Constraints helpers  
  const addBlockedTopic = () => {
    setFormData(prev => ({
      ...prev,
      constraints: {
        ...prev.constraints,
        blockedTopics: [...prev.constraints.blockedTopics, '']
      }
    }))
  }

  const removeBlockedTopic = (index: number) => {
    setFormData(prev => ({
      ...prev,
      constraints: {
        ...prev.constraints,
        blockedTopics: prev.constraints.blockedTopics.filter((_, i) => i !== index)
      }
    }))
  }

  const updateBlockedTopic = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      constraints: {
        ...prev.constraints,
        blockedTopics: prev.constraints.blockedTopics.map((topic, i) => 
          i === index ? value : topic
        )
      }
    }))
  }

  const addAllowedDomain = () => {
    setFormData(prev => ({
      ...prev,
      constraints: {
        ...prev.constraints,
        allowedDomains: [...prev.constraints.allowedDomains, '']
      }
    }))
  }

  const removeAllowedDomain = (index: number) => {
    setFormData(prev => ({
      ...prev,
      constraints: {
        ...prev.constraints,
        allowedDomains: prev.constraints.allowedDomains.filter((_, i) => i !== index)
      }
    }))
  }

  const updateAllowedDomain = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      constraints: {
        ...prev.constraints,
        allowedDomains: prev.constraints.allowedDomains.map((domain, i) => 
          i === index ? value : domain
        )
      }
    }))
  }

  // Tool Preferences helpers
  const addPreferredTool = () => {
    setFormData(prev => ({
      ...prev,
      toolPreferences: {
        ...prev.toolPreferences,
        preferredTools: [...prev.toolPreferences.preferredTools, '']
      }
    }))
  }

  const removePreferredTool = (index: number) => {
    setFormData(prev => ({
      ...prev,
      toolPreferences: {
        ...prev.toolPreferences,
        preferredTools: prev.toolPreferences.preferredTools.filter((_, i) => i !== index)
      }
    }))
  }

  const updatePreferredTool = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      toolPreferences: {
        ...prev.toolPreferences,
        preferredTools: prev.toolPreferences.preferredTools.map((tool, i) => 
          i === index ? value : tool
        )
      }
    }))
  }

  const addAvoidTool = () => {
    setFormData(prev => ({
      ...prev,
      toolPreferences: {
        ...prev.toolPreferences,
        avoidTools: [...prev.toolPreferences.avoidTools, '']
      }
    }))
  }

  const removeAvoidTool = (index: number) => {
    setFormData(prev => ({
      ...prev,
      toolPreferences: {
        ...prev.toolPreferences,
        avoidTools: prev.toolPreferences.avoidTools.filter((_, i) => i !== index)
      }
    }))
  }

  const updateAvoidTool = (index: number, value: string) => {
    setFormData(prev => ({
      ...prev,
      toolPreferences: {
        ...prev.toolPreferences,
        avoidTools: prev.toolPreferences.avoidTools.map((tool, i) => 
          i === index ? value : tool
        )
      }
    }))
  }

  const validateForm = () => {
    if (!formData.name.trim()) {
      setValidationError('Persona name is required')
      return false
    }
    
    // Validate persona name (DNS-compatible)
    const nameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/
    if (!nameRegex.test(formData.name)) {
      setValidationError('Persona name must be lowercase alphanumeric with hyphens')
      return false
    }
    
    if (formData.name.length > 63) {
      setValidationError('Persona name must be 63 characters or less')
      return false
    }

    // CRD required field validation
    if (!formData.displayName.trim()) {
      setValidationError('Display name is required')
      return false
    }

    if (!formData.description.trim()) {
      setValidationError('Description is required')
      return false
    }

    if (!formData.systemPrompt.trim()) {
      setValidationError('System prompt is required')
      return false
    }

    if (formData.systemPrompt.length < 20) {
      setValidationError('System prompt must be at least 20 characters')
      return false
    }

    // Numeric validation for constraints
    if (formData.constraints.maxResponseTokens !== undefined) {
      if (formData.constraints.maxResponseTokens < 1 || formData.constraints.maxResponseTokens > 100000) {
        setValidationError('Max Response Tokens must be between 1 and 100,000')
        return false
      }
    }

    if (formData.constraints.maxToolCalls !== undefined) {
      if (formData.constraints.maxToolCalls < 1 || formData.constraints.maxToolCalls > 50) {
        setValidationError('Max Tool Calls must be between 1 and 50')
        return false
      }
    }

    if (formData.constraints.maxKnowledgeQueries !== undefined) {
      if (formData.constraints.maxKnowledgeQueries < 1 || formData.constraints.maxKnowledgeQueries > 20) {
        setValidationError('Max Knowledge Queries must be between 1 and 20')
        return false
      }
    }

    // Validate response timeout format
    if (formData.constraints.responseTimeout && formData.constraints.responseTimeout.trim()) {
      const timeoutRegex = /^\d+[smh]$/
      if (!timeoutRegex.test(formData.constraints.responseTimeout)) {
        setValidationError('Response Timeout must be in format like "30s", "5m", or "1h"')
        return false
      }
    }

    // Validate response format max length
    if (formData.responseFormat.maxLength !== undefined) {
      if (formData.responseFormat.maxLength < 1 || formData.responseFormat.maxLength > 50000) {
        setValidationError('Response Format Max Length must be between 1 and 50,000')
        return false
      }
    }

    // Validate knowledge source priorities
    for (let i = 0; i < formData.knowledgeSources.length; i++) {
      const source = formData.knowledgeSources[i]
      if (source.priority < 1 || source.priority > 1000) {
        setValidationError(`Knowledge Source ${i + 1} priority must be between 1 and 1,000`)
        return false
      }
    }

    // Validate rule priorities
    for (let i = 0; i < formData.rules.length; i++) {
      const rule = formData.rules[i]
      if (rule.priority < 1 || rule.priority > 1000) {
        setValidationError(`Rule ${i + 1} priority must be between 1 and 1,000`)
        return false
      }
    }

    // UI-only validations (not CRD requirements)
    // All remaining validations are optional

    // Note: Personality traits (tone) are optional per CRD spec

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

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Users className="h-5 w-5" />
            <span>Basic Information</span>
          </CardTitle>
          <CardDescription>
            Configure the basic settings for your persona
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Persona Name *</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="helpful-assistant"
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
            <Label htmlFor="displayName">Display Name *</Label>
            <Input
              id="displayName"
              value={formData.displayName}
              onChange={(e) => handleInputChange('displayName', e.target.value)}
              placeholder="Helpful Assistant"
              disabled={isLoading}
              required
            />
            <p className="text-sm text-muted-foreground">
              Human-readable name for this persona
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description *</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              placeholder="A brief description of this persona's purpose..."
              rows={3}
              disabled={isLoading}
              required
            />
            <p className="text-sm text-muted-foreground">
              Brief description of this persona's role and capabilities
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Personality & Behavior */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Brain className="h-5 w-5" />
            <span>Personality & Behavior</span>
          </CardTitle>
          <CardDescription>
            Define the persona's personality and communication style
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="systemPrompt">System Prompt *</Label>
            <Textarea
              id="systemPrompt"
              value={formData.systemPrompt}
              onChange={(e) => handleInputChange('systemPrompt', e.target.value)}
              placeholder="You are a helpful assistant that..."
              rows={4}
              disabled={isLoading}
              required
            />
            <p className="text-sm text-muted-foreground">
              The core instructions that define how this persona should behave
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="tone">Tone</Label>
            <Select value={formData.tone} onValueChange={(value) => handleInputChange('tone', value)}>
              <SelectTrigger>
                <SelectValue placeholder="Select a tone (optional)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="professional">Professional</SelectItem>
                <SelectItem value="friendly">Friendly</SelectItem>
                <SelectItem value="concise">Concise</SelectItem>
                <SelectItem value="detailed">Detailed</SelectItem>
                <SelectItem value="creative">Creative</SelectItem>
                <SelectItem value="analytical">Analytical</SelectItem>
                <SelectItem value="empathetic">Empathetic</SelectItem>
                <SelectItem value="authoritative">Authoritative</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-sm text-muted-foreground">
              The overall tone and communication style for this persona
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="language">Language</Label>
              <Input
                id="language"
                value={formData.language}
                onChange={(e) => handleInputChange('language', e.target.value)}
                placeholder="English"
                disabled={isLoading}
              />
              <p className="text-sm text-muted-foreground">
                Primary language for responses
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="version">Version</Label>
              <Input
                id="version"
                value={formData.version}
                onChange={(e) => handleInputChange('version', e.target.value)}
                placeholder="1.0.0"
                disabled={isLoading}
              />
              <p className="text-sm text-muted-foreground">
                Version identifier for this persona
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Capabilities and Limitations */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Target className="h-5 w-5" />
            <span>Capabilities & Limitations</span>
          </CardTitle>
          <CardDescription>
            Define what this persona can and cannot do
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-3">
            <Label>Capabilities</Label>
            {formData.capabilities.map((capability, index) => (
              <div key={index} className="flex items-center space-x-2">
                <Input
                  value={capability}
                  onChange={(e) => updateCapability(index, e.target.value)}
                  placeholder="Describe a capability..."
                  disabled={isLoading}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeCapability(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
            ))}
            <Button 
              type="button" 
              variant="outline" 
              onClick={addCapability}
              disabled={isLoading}
            >
              Add Capability
            </Button>
            <p className="text-sm text-muted-foreground">
              List the specific capabilities this persona possesses
            </p>
          </div>

          <div className="space-y-3">
            <Label>Limitations</Label>
            {formData.limitations.map((limitation, index) => (
              <div key={index} className="flex items-center space-x-2">
                <Input
                  value={limitation}
                  onChange={(e) => updateLimitation(index, e.target.value)}
                  placeholder="Describe a limitation..."
                  disabled={isLoading}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeLimitation(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
            ))}
            <Button 
              type="button" 
              variant="outline" 
              onClick={addLimitation}
              disabled={isLoading}
            >
              Add Limitation
            </Button>
            <p className="text-sm text-muted-foreground">
              List the specific limitations of this persona
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Instructions */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <BookOpen className="h-5 w-5" />
            <span>Instructions</span>
          </CardTitle>
          <CardDescription>
            Additional specific instructions for this persona
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {formData.instructions.map((instruction, index) => (
            <div key={index} className="flex items-center space-x-2">
              <Textarea
                value={instruction}
                onChange={(e) => updateInstruction(index, e.target.value)}
                placeholder="Add an instruction..."
                rows={2}
                disabled={isLoading}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => removeInstruction(index)}
                disabled={isLoading}
              >
                Remove
              </Button>
            </div>
          ))}
          <Button 
            type="button" 
            variant="outline" 
            onClick={addInstruction}
            disabled={isLoading}
          >
            Add Instruction
          </Button>
          <p className="text-sm text-muted-foreground">
            Detailed instructions that supplement the system prompt
          </p>
        </CardContent>
      </Card>

      {/* Response Examples */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <MessageSquare className="h-5 w-5" />
            <span>Response Examples</span>
          </CardTitle>
          <CardDescription>
            Provide examples of how this persona should respond
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {formData.examples.map((example, index) => (
            <div key={index} className="space-y-3 p-4 border rounded-lg">
              <div className="flex items-center justify-between">
                <Label>Example {index + 1}</Label>
                {formData.examples.length > 1 && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => removeExample(index)}
                    disabled={isLoading}
                  >
                    Remove
                  </Button>
                )}
              </div>
              
              <div className="space-y-2">
                <Label htmlFor={`context-${index}`}>Context (optional)</Label>
                <Input
                  id={`context-${index}`}
                  value={example.context || ''}
                  onChange={(e) => updateExampleContext(index, e.target.value)}
                  placeholder="Context for this example..."
                  disabled={isLoading}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor={`input-${index}`}>User Input</Label>
                <Input
                  id={`input-${index}`}
                  value={example.input}
                  onChange={(e) => updateExample(index, 'input', e.target.value)}
                  placeholder="What the user might say..."
                  disabled={isLoading}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor={`output-${index}`}>Expected Response</Label>
                <Textarea
                  id={`output-${index}`}
                  value={example.output}
                  onChange={(e) => updateExample(index, 'output', e.target.value)}
                  placeholder="How the persona should respond..."
                  rows={3}
                  disabled={isLoading}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor={`tags-${index}`}>Tags (optional)</Label>
                <Input
                  id={`tags-${index}`}
                  value={example.tags?.join(', ') || ''}
                  onChange={(e) => updateExampleTags(index, e.target.value.split(',').map(tag => tag.trim()).filter(tag => tag))}
                  placeholder="tag1, tag2, tag3"
                  disabled={isLoading}
                />
                <p className="text-xs text-muted-foreground">
                  Comma-separated tags for categorizing this example
                </p>
              </div>
            </div>
          ))}
          <Button 
            type="button" 
            variant="outline" 
            onClick={addExample}
            disabled={isLoading}
          >
            Add Example
          </Button>
        </CardContent>
      </Card>

      {/* Knowledge Sources */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Database className="h-5 w-5" />
            <span>Knowledge Sources</span>
          </CardTitle>
          <CardDescription>
            External knowledge bases this persona can access
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {formData.knowledgeSources.map((source, index) => (
            <div key={index} className="space-y-3 p-4 border rounded-lg">
              <div className="flex items-center justify-between">
                <Label>Knowledge Source {index + 1}</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeKnowledgeSource(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Name *</Label>
                  <Input
                    value={source.name}
                    onChange={(e) => updateKnowledgeSource(index, 'name', e.target.value)}
                    placeholder="Knowledge source name..."
                    disabled={isLoading}
                  />
                </div>
                
                <div className="space-y-2">
                  <Label>Type *</Label>
                  <Select 
                    value={source.type} 
                    onValueChange={(value) => updateKnowledgeSource(index, 'type', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="url">URL</SelectItem>
                      <SelectItem value="document">Document</SelectItem>
                      <SelectItem value="database">Database</SelectItem>
                      <SelectItem value="api">API</SelectItem>
                      <SelectItem value="vector-store">Vector Store</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              
              <div className="space-y-2">
                <Label>URL</Label>
                <Input
                  value={source.url || ''}
                  onChange={(e) => updateKnowledgeSource(index, 'url', e.target.value)}
                  placeholder="https://example.com/api"
                  disabled={isLoading}
                />
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Priority</Label>
                  <Input
                    type="number"
                    min="1"
                    max="1000"
                    value={source.priority}
                    onChange={(e) => {
                      const validatedValue = validateNumericInput(e.target.value, 1, 1000) || 100
                      updateKnowledgeSource(index, 'priority', validatedValue)
                    }}
                    placeholder="100 (1-1000)"
                    disabled={isLoading}
                  />
                  <p className="text-xs text-muted-foreground">
                    Priority order (1-1000)
                  </p>
                </div>
                
                <div className="space-y-2">
                  <Label className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      checked={source.enabled}
                      onChange={(e) => updateKnowledgeSource(index, 'enabled', e.target.checked)}
                      disabled={isLoading}
                    />
                    <span>Enabled</span>
                  </Label>
                </div>
              </div>
            </div>
          ))}
          <Button 
            type="button" 
            variant="outline" 
            onClick={addKnowledgeSource}
            disabled={isLoading}
          >
            Add Knowledge Source
          </Button>
        </CardContent>
      </Card>

      {/* Constraints */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Shield className="h-5 w-5" />
            <span>Safety & Constraints</span>
          </CardTitle>
          <CardDescription>
            Operational limits and safety controls
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Max Response Tokens</Label>
              <Input
                type="number"
                min="1"
                max="100000"
                value={formData.constraints.maxResponseTokens || ''}
                onChange={(e) => handleConstraintNumericChange('maxResponseTokens', e.target.value)}
                placeholder="2000 (max: 100,000)"
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Maximum tokens in response (1-100,000)
              </p>
            </div>
            
            <div className="space-y-2">
              <Label>Max Tool Calls</Label>
              <Input
                type="number"
                min="1"
                max="50"
                value={formData.constraints.maxToolCalls || ''}
                onChange={(e) => handleConstraintNumericChange('maxToolCalls', e.target.value)}
                placeholder="5 (max: 50)"
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Maximum tool calls per request (1-50)
              </p>
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Response Timeout</Label>
              <Input
                value={formData.constraints.responseTimeout || ''}
                onChange={(e) => handleInputChange('constraints', {
                  ...formData.constraints,
                  responseTimeout: e.target.value
                })}
                placeholder="30s, 5m, or 1h"
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Format: 30s (seconds), 5m (minutes), 1h (hours)
              </p>
            </div>
            
            <div className="space-y-2">
              <Label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={formData.constraints.requireDocumentation}
                  onChange={(e) => handleInputChange('constraints', {
                    ...formData.constraints,
                    requireDocumentation: e.target.checked
                  })}
                  disabled={isLoading}
                />
                <span>Require Documentation</span>
              </Label>
            </div>
          </div>
          
          <div className="space-y-3">
            <Label>Blocked Topics</Label>
            {formData.constraints.blockedTopics.map((topic, index) => (
              <div key={index} className="flex items-center space-x-2">
                <Input
                  value={topic}
                  onChange={(e) => updateBlockedTopic(index, e.target.value)}
                  placeholder="Topic to block..."
                  disabled={isLoading}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeBlockedTopic(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
            ))}
            <Button 
              type="button" 
              variant="outline" 
              onClick={addBlockedTopic}
              disabled={isLoading}
            >
              Add Blocked Topic
            </Button>
          </div>
          
          <div className="space-y-3">
            <Label>Allowed Domains</Label>
            {formData.constraints.allowedDomains.map((domain, index) => (
              <div key={index} className="flex items-center space-x-2">
                <Input
                  value={domain}
                  onChange={(e) => updateAllowedDomain(index, e.target.value)}
                  placeholder="example.com"
                  disabled={isLoading}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeAllowedDomain(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
            ))}
            <Button 
              type="button" 
              variant="outline" 
              onClick={addAllowedDomain}
              disabled={isLoading}
            >
              Add Allowed Domain
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Rules */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Workflow className="h-5 w-5" />
            <span>Behavioral Rules</span>
          </CardTitle>
          <CardDescription>
            Conditional behaviors and logic rules
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {formData.rules.map((rule, index) => (
            <div key={index} className="space-y-3 p-4 border rounded-lg">
              <div className="flex items-center justify-between">
                <Label>Rule {index + 1}</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeRule(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Name *</Label>
                  <Input
                    value={rule.name}
                    onChange={(e) => updateRule(index, 'name', e.target.value)}
                    placeholder="Rule name..."
                    disabled={isLoading}
                  />
                </div>
                
                <div className="space-y-2">
                  <Label>Priority</Label>
                  <Input
                    type="number"
                    min="1"
                    max="1000"
                    value={rule.priority}
                    onChange={(e) => {
                      const validatedValue = validateNumericInput(e.target.value, 1, 1000) || 100
                      updateRule(index, 'priority', validatedValue)
                    }}
                    placeholder="100 (1-1000)"
                    disabled={isLoading}
                  />
                  <p className="text-xs text-muted-foreground">
                    Rule priority (1-1000)
                  </p>
                </div>
              </div>
              
              <div className="space-y-2">
                <Label>Condition *</Label>
                <Textarea
                  value={rule.condition}
                  onChange={(e) => updateRule(index, 'condition', e.target.value)}
                  placeholder="when user asks about..."
                  rows={2}
                  disabled={isLoading}
                />
              </div>
              
              <div className="space-y-2">
                <Label>Action *</Label>
                <Textarea
                  value={rule.action}
                  onChange={(e) => updateRule(index, 'action', e.target.value)}
                  placeholder="respond with..."
                  rows={2}
                  disabled={isLoading}
                />
              </div>
              
              <div className="space-y-2">
                <Label>Description</Label>
                <Input
                  value={rule.description || ''}
                  onChange={(e) => updateRule(index, 'description', e.target.value)}
                  placeholder="What this rule does..."
                  disabled={isLoading}
                />
              </div>
              
              <div className="space-y-2">
                <Label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={rule.enabled}
                    onChange={(e) => updateRule(index, 'enabled', e.target.checked)}
                    disabled={isLoading}
                  />
                  <span>Enabled</span>
                </Label>
              </div>
            </div>
          ))}
          <Button 
            type="button" 
            variant="outline" 
            onClick={addRule}
            disabled={isLoading}
          >
            Add Rule
          </Button>
        </CardContent>
      </Card>

      {/* Response Format */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <FileText className="h-5 w-5" />
            <span>Response Format</span>
          </CardTitle>
          <CardDescription>
            Configure how responses should be structured
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Response Type</Label>
              <Select 
                value={formData.responseFormat.type} 
                onValueChange={(value) => handleInputChange('responseFormat', {
                  ...formData.responseFormat,
                  type: value as ResponseFormatType
                })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Text</SelectItem>
                  <SelectItem value="markdown">Markdown</SelectItem>
                  <SelectItem value="json">JSON</SelectItem>
                  <SelectItem value="structured">Structured</SelectItem>
                  <SelectItem value="list">List</SelectItem>
                  <SelectItem value="table">Table</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div className="space-y-2">
              <Label>Max Length</Label>
              <Input
                type="number"
                min="1"
                max="50000"
                value={formData.responseFormat.maxLength || ''}
                onChange={(e) => handleResponseFormatNumericChange('maxLength', e.target.value)}
                placeholder="1000 (max: 50,000)"
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Maximum response length in characters (1-50,000)
              </p>
            </div>
          </div>
          
          <div className="space-y-3">
            <Label className="flex items-center space-x-2">
              <input
                type="checkbox"
                checked={formData.responseFormat.includeConfidence}
                onChange={(e) => handleInputChange('responseFormat', {
                  ...formData.responseFormat,
                  includeConfidence: e.target.checked
                })}
                disabled={isLoading}
              />
              <span>Include Confidence Scores</span>
            </Label>
            
            <Label className="flex items-center space-x-2">
              <input
                type="checkbox"
                checked={formData.responseFormat.includeSources}
                onChange={(e) => handleInputChange('responseFormat', {
                  ...formData.responseFormat,
                  includeSources: e.target.checked
                })}
                disabled={isLoading}
              />
              <span>Include Sources</span>
            </Label>
          </div>
          
          <div className="space-y-2">
            <Label>Response Template</Label>
            <Textarea
              value={formData.responseFormat.template || ''}
              onChange={(e) => handleInputChange('responseFormat', {
                ...formData.responseFormat,
                template: e.target.value
              })}
              placeholder="Custom response template..."
              rows={3}
              disabled={isLoading}
            />
          </div>
        </CardContent>
      </Card>

      {/* Tool Preferences */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Settings className="h-5 w-5" />
            <span>Tool Preferences</span>
          </CardTitle>
          <CardDescription>
            Configure how this persona uses tools
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Usage Strategy</Label>
              <Select 
                value={formData.toolPreferences.strategy} 
                onValueChange={(value) => handleInputChange('toolPreferences', {
                  ...formData.toolPreferences,
                  strategy: value as ToolUsageStrategy
                })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="conservative">Conservative</SelectItem>
                  <SelectItem value="balanced">Balanced</SelectItem>
                  <SelectItem value="aggressive">Aggressive</SelectItem>
                  <SelectItem value="minimal">Minimal</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          
          <div className="space-y-3">
            <Label className="flex items-center space-x-2">
              <input
                type="checkbox"
                checked={formData.toolPreferences.alwaysConfirm}
                onChange={(e) => handleInputChange('toolPreferences', {
                  ...formData.toolPreferences,
                  alwaysConfirm: e.target.checked
                })}
                disabled={isLoading}
              />
              <span>Always Confirm Tool Use</span>
            </Label>
            
            <Label className="flex items-center space-x-2">
              <input
                type="checkbox"
                checked={formData.toolPreferences.explainToolUse}
                onChange={(e) => handleInputChange('toolPreferences', {
                  ...formData.toolPreferences,
                  explainToolUse: e.target.checked
                })}
                disabled={isLoading}
              />
              <span>Explain Tool Usage</span>
            </Label>
          </div>
          
          <div className="space-y-3">
            <Label>Preferred Tools</Label>
            {formData.toolPreferences.preferredTools.map((tool, index) => (
              <div key={index} className="flex items-center space-x-2">
                <Input
                  value={tool}
                  onChange={(e) => updatePreferredTool(index, e.target.value)}
                  placeholder="Tool name..."
                  disabled={isLoading}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removePreferredTool(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
            ))}
            <Button 
              type="button" 
              variant="outline" 
              onClick={addPreferredTool}
              disabled={isLoading}
            >
              Add Preferred Tool
            </Button>
          </div>
          
          <div className="space-y-3">
            <Label>Tools to Avoid</Label>
            {formData.toolPreferences.avoidTools.map((tool, index) => (
              <div key={index} className="flex items-center space-x-2">
                <Input
                  value={tool}
                  onChange={(e) => updateAvoidTool(index, e.target.value)}
                  placeholder="Tool name..."
                  disabled={isLoading}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeAvoidTool(index)}
                  disabled={isLoading}
                >
                  Remove
                </Button>
              </div>
            ))}
            <Button 
              type="button" 
              variant="outline" 
              onClick={addAvoidTool}
              disabled={isLoading}
            >
              Add Tool to Avoid
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Parent Persona */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Share2 className="h-5 w-5" />
            <span>Persona Inheritance</span>
          </CardTitle>
          <CardDescription>
            Inherit behaviors from another persona
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Parent Persona Name</Label>
              <Input
                value={formData.parentPersona?.name || ''}
                onChange={(e) => handleInputChange('parentPersona', 
                  e.target.value ? {
                    ...formData.parentPersona,
                    name: e.target.value
                  } : undefined
                )}
                placeholder="parent-persona-name"
                disabled={isLoading}
              />
            </div>
            
            <div className="space-y-2">
              <Label>Namespace (optional)</Label>
              <Input
                value={formData.parentPersona?.namespace || ''}
                onChange={(e) => handleInputChange('parentPersona', 
                  formData.parentPersona ? {
                    ...formData.parentPersona,
                    namespace: e.target.value
                  } : undefined
                )}
                placeholder="Leave empty for same namespace"
                disabled={isLoading}
              />
            </div>
          </div>
          <p className="text-sm text-muted-foreground">
            This persona will inherit capabilities and behaviors from the specified parent persona
          </p>
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
          {isLoading ? (isEdit ? 'Updating...' : 'Creating...') : (isEdit ? 'Update Persona' : 'Create Persona')}
        </Button>
      </div>
    </form>
  )
}