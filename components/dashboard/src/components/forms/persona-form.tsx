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
import { Users, Brain, MessageSquare, AlertCircle, Settings } from 'lucide-react'

const PERSONA_ROLES = [
  { id: 'assistant', name: 'General Assistant', description: 'Helpful, general-purpose assistant' },
  { id: 'analyst', name: 'Data Analyst', description: 'Data analysis and insights specialist' },
  { id: 'developer', name: 'Software Developer', description: 'Programming and development expert' },
  { id: 'writer', name: 'Content Writer', description: 'Creative and technical writing specialist' },
  { id: 'researcher', name: 'Researcher', description: 'Research and information gathering expert' },
  { id: 'teacher', name: 'Teacher', description: 'Educational and training specialist' },
  { id: 'consultant', name: 'Business Consultant', description: 'Strategic business advice expert' },
  { id: 'support', name: 'Customer Support', description: 'Customer service and support specialist' },
  { id: 'custom', name: 'Custom Role', description: 'Define your own specialized role' }
]

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

export interface PersonaFormData {
  name: string
  displayName: string // Required by CRD
  role: string // UI-only field, not mapped to CRD
  customRole: string
  description: string // Required by CRD
  systemPrompt: string // Required by CRD
  traits: string[] // UI-only, maps to optional tone field
  examples: Array<{input: string, output: string}>
  temperature: number
  maxTokens: number
  enabled: boolean
  requireApproval: boolean
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
    role: '',
    customRole: '',
    description: '',
    systemPrompt: '',
    traits: [],
    examples: [
      { input: '', output: '' },
      { input: '', output: '' }
    ],
    temperature: 0.7,
    maxTokens: 2048,
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

  const handleInputChange = (field: keyof PersonaFormData, value: any) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }))

    // Auto-populate fields when role changes
    if (field === 'role' && value in SAMPLE_PERSONAS) {
      const sample = SAMPLE_PERSONAS[value as keyof typeof SAMPLE_PERSONAS]
      setFormData(prev => ({
        ...prev,
        systemPrompt: prev.systemPrompt || sample.systemPrompt,
        traits: prev.traits.length === 0 ? sample.traits : prev.traits,
        examples: prev.examples.every(ex => !ex.input && !ex.output) ? sample.examples : prev.examples
      }))
    }
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
    }
  }

  const handleTraitToggle = (traitId: string) => {
    setFormData(prev => ({
      ...prev,
      traits: prev.traits.includes(traitId)
        ? prev.traits.filter(t => t !== traitId)
        : [...prev.traits, traitId]
    }))
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
      examples: [...prev.examples, { input: '', output: '' }]
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

    // UI-only validations (not CRD requirements)
    if (!formData.role) {
      setValidationError('Persona role is required')
      return false
    }

    if (formData.role === 'custom' && !formData.customRole.trim()) {
      setValidationError('Custom role description is required')
      return false
    }

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

  const selectedRole = PERSONA_ROLES.find(r => r.id === formData.role)
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
            <Label htmlFor="role">Role *</Label>
            <Select 
              value={formData.role} 
              onValueChange={(value) => handleInputChange('role', value)}
              disabled={isLoading}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select persona role" />
              </SelectTrigger>
              <SelectContent>
                {PERSONA_ROLES.map((role) => (
                  <SelectItem key={role.id} value={role.id}>
                    <div>
                      <div className="font-medium">{role.name}</div>
                      <div className="text-sm text-muted-foreground">{role.description}</div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {selectedRole && (
              <p className="text-sm text-muted-foreground">
                {selectedRole.description}
              </p>
            )}
          </div>

          {formData.role === 'custom' && (
            <div className="space-y-2">
              <Label htmlFor="customRole">Custom Role *</Label>
              <Input
                id="customRole"
                value={formData.customRole}
                onChange={(e) => handleInputChange('customRole', e.target.value)}
                placeholder="Specialized Expert"
                disabled={isLoading}
                required
              />
            </div>
          )}

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
            <Label>Personality Traits</Label>
            <div className="grid grid-cols-2 gap-2">
              {PERSONALITY_TRAITS.map((trait) => (
                <div
                  key={trait.id}
                  className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                    formData.traits.includes(trait.id)
                      ? 'border-primary bg-primary/5'
                      : 'border-gray-200 hover:border-gray-300'
                  } ${isLoading ? 'opacity-50 pointer-events-none' : ''}`}
                  onClick={() => !isLoading && handleTraitToggle(trait.id)}
                >
                  <div className="font-medium text-sm">{trait.name}</div>
                  <div className="text-xs text-muted-foreground">{trait.description}</div>
                </div>
              ))}
            </div>
            <p className="text-sm text-muted-foreground">
              Optional traits that describe this persona's communication style (defaults to professional tone)
            </p>
          </div>
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
            <div key={index} className="space-y-2 p-4 border rounded-lg">
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
                  rows={2}
                  disabled={isLoading}
                />
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

      {/* Parameters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Settings className="h-5 w-5" />
            <span>Generation Parameters</span>
          </CardTitle>
          <CardDescription>
            Configure how the persona generates responses
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
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
              <p className="text-xs text-muted-foreground">
                Lower = more focused, Higher = more creative
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="maxTokens">Max Tokens</Label>
              <Input
                id="maxTokens"
                type="number"
                value={formData.maxTokens}
                onChange={(e) => handleInputChange('maxTokens', parseInt(e.target.value) || 0)}
                min={1}
                max={8192}
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Maximum response length
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Settings</CardTitle>
          <CardDescription>
            Additional persona configuration
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Enable Persona</Label>
              <p className="text-sm text-muted-foreground">
                Allow this persona to be used by agents
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
                Require admin approval before using this persona
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
          {isLoading ? (isEdit ? 'Updating...' : 'Creating...') : (isEdit ? 'Update Persona' : 'Create Persona')}
        </Button>
      </div>
    </form>
  )
}