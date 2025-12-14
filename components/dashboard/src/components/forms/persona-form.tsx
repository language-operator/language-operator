'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Users, Brain, MessageSquare, AlertCircle, Target, BookOpen } from 'lucide-react'


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

      {/* Parameters */}


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