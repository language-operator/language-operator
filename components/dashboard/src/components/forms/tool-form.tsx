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
import { Wrench, Globe, Code, AlertCircle, Settings } from 'lucide-react'

const TOOL_TYPES = [
  { id: 'mcp', name: 'MCP (Model Context Protocol)', description: 'Standard MCP server with tools' },
  { id: 'api', name: 'REST API', description: 'HTTP-based API service' },
  { id: 'webhook', name: 'Webhook', description: 'Receive HTTP callbacks' },
  { id: 'database', name: 'Database', description: 'Database connection tool' },
  { id: 'filesystem', name: 'File System', description: 'File and directory operations' },
  { id: 'custom', name: 'Custom', description: 'Custom tool implementation' }
]

const MCP_TOOLS = [
  { id: 'workspace', name: 'Workspace Tool', image: 'ghcr.io/language-operator/workspace-tool:latest' },
  { id: 'browser', name: 'Browser Tool', image: 'ghcr.io/language-operator/browser-tool:latest' },
  { id: 'git', name: 'Git Tool', image: 'ghcr.io/language-operator/git-tool:latest' },
  { id: 'database', name: 'Database Tool', image: 'ghcr.io/language-operator/database-tool:latest' },
  { id: 'kubernetes', name: 'Kubernetes Tool', image: 'ghcr.io/language-operator/kubernetes-tool:latest' },
]

export interface ToolFormData {
  name: string
  type: string
  description: string
  image: string
  endpoint: string
  port: number
  healthCheckPath: string
  envVars: Array<{key: string, value: string}>
  resources: {
    cpu: string
    memory: string
    cpuLimit: string
    memoryLimit: string
  }
  enabled: boolean
  requireApproval: boolean
  timeout: number
  retries: number
}

interface ToolFormProps {
  initialData?: Partial<ToolFormData>
  isLoading?: boolean
  error?: string
  onSubmit: (data: ToolFormData) => Promise<void>
  onCancel: () => void
  isEdit?: boolean
}

export function ToolForm({ 
  initialData, 
  isLoading = false, 
  error, 
  onSubmit, 
  onCancel,
  isEdit = false 
}: ToolFormProps) {
  const [formData, setFormData] = useState<ToolFormData>({
    name: '',
    type: '',
    description: '',
    image: '',
    endpoint: '',
    port: 3000,
    healthCheckPath: '/health',
    envVars: [],
    resources: {
      cpu: '100m',
      memory: '128Mi',
      cpuLimit: '500m',
      memoryLimit: '512Mi'
    },
    enabled: true,
    requireApproval: false,
    timeout: 30,
    retries: 3,
    ...initialData
  })

  const [validationError, setValidationError] = useState('')

  // Update form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      setFormData(prev => ({ ...prev, ...initialData }))
    }
  }, [initialData])

  const handleInputChange = (field: keyof ToolFormData | string, value: any) => {
    if (field.startsWith('resources.')) {
      const resourceField = field.split('.')[1]
      setFormData(prev => ({
        ...prev,
        resources: {
          ...prev.resources,
          [resourceField]: value
        }
      }))
    } else {
      setFormData(prev => ({
        ...prev,
        [field]: value
      }))
    }

    // Auto-set port when type changes
    if (field === 'type' && value === 'mcp') {
      if (!formData.port || formData.port === 3000) {
        setFormData(prev => ({ ...prev, port: 3001 }))
      }
    }
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
    }
  }

  const handleMcpToolSelect = (toolId: string) => {
    const tool = MCP_TOOLS.find(t => t.id === toolId)
    if (tool) {
      setFormData(prev => ({
        ...prev,
        name: prev.name || toolId,
        image: tool.image,
        type: 'mcp'
      }))
    }
  }

  const addEnvVar = () => {
    setFormData(prev => ({
      ...prev,
      envVars: [...prev.envVars, { key: '', value: '' }]
    }))
  }

  const removeEnvVar = (index: number) => {
    setFormData(prev => ({
      ...prev,
      envVars: prev.envVars.filter((_, i) => i !== index)
    }))
  }

  const updateEnvVar = (index: number, field: 'key' | 'value', value: string) => {
    setFormData(prev => ({
      ...prev,
      envVars: prev.envVars.map((env, i) => 
        i === index ? { ...env, [field]: value } : env
      )
    }))
  }

  const validateForm = () => {
    if (!formData.name.trim()) {
      setValidationError('Tool name is required')
      return false
    }
    
    // Validate tool name (DNS-compatible)
    const nameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/
    if (!nameRegex.test(formData.name)) {
      setValidationError('Tool name must be lowercase alphanumeric with hyphens')
      return false
    }
    
    if (formData.name.length > 63) {
      setValidationError('Tool name must be 63 characters or less')
      return false
    }

    if (!formData.type) {
      setValidationError('Tool type is required')
      return false
    }

    if (formData.type === 'mcp' && !formData.image.trim()) {
      setValidationError('Container image is required for MCP tools')
      return false
    }

    if (formData.type === 'api' && !formData.endpoint.trim()) {
      setValidationError('API endpoint is required for API tools')
      return false
    }

    if (formData.port && (formData.port < 1 || formData.port > 65535)) {
      setValidationError('Port must be between 1 and 65535')
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

  const selectedType = TOOL_TYPES.find(t => t.id === formData.type)
  const displayError = error || validationError

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Wrench className="h-5 w-5" />
            <span>Basic Information</span>
          </CardTitle>
          <CardDescription>
            Configure the basic settings for your language tool
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Tool Name *</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="my-tool"
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
            <Label htmlFor="type">Tool Type *</Label>
            <Select 
              value={formData.type} 
              onValueChange={(value) => handleInputChange('type', value)}
              disabled={isLoading}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select tool type" />
              </SelectTrigger>
              <SelectContent>
                {TOOL_TYPES.map((type) => (
                  <SelectItem key={type.id} value={type.id}>
                    <div>
                      <div className="font-medium">{type.name}</div>
                      <div className="text-sm text-muted-foreground">{type.description}</div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {selectedType && (
              <p className="text-sm text-muted-foreground">
                {selectedType.description}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              placeholder="Description of this tool's capabilities..."
              rows={3}
              disabled={isLoading}
            />
          </div>
        </CardContent>
      </Card>

      {/* MCP Tool Selection */}
      {formData.type === 'mcp' && (
        <Card>
          <CardHeader>
            <CardTitle>Pre-built MCP Tools</CardTitle>
            <CardDescription>
              Select from available MCP tools or configure a custom one
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-2">
              {MCP_TOOLS.map((tool) => (
                <Button
                  key={tool.id}
                  type="button"
                  variant="outline"
                  className="justify-start h-auto p-4"
                  onClick={() => handleMcpToolSelect(tool.id)}
                  disabled={isLoading}
                >
                  <div className="text-left">
                    <div className="font-medium">{tool.name}</div>
                    <div className="text-sm text-muted-foreground">{tool.image}</div>
                  </div>
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Settings className="h-5 w-5" />
            <span>Configuration</span>
          </CardTitle>
          <CardDescription>
            Configure tool-specific settings
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {(formData.type === 'mcp' || formData.type === 'custom') && (
            <div className="space-y-2">
              <Label htmlFor="image">Container Image *</Label>
              <Input
                id="image"
                value={formData.image}
                onChange={(e) => handleInputChange('image', e.target.value)}
                placeholder="ghcr.io/org/tool:latest"
                className="font-mono"
                required={formData.type === 'mcp'}
                disabled={isLoading}
              />
            </div>
          )}

          {formData.type === 'api' && (
            <div className="space-y-2">
              <Label htmlFor="endpoint">API Endpoint *</Label>
              <Input
                id="endpoint"
                value={formData.endpoint}
                onChange={(e) => handleInputChange('endpoint', e.target.value)}
                placeholder="https://api.example.com"
                className="font-mono"
                required
                disabled={isLoading}
              />
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="port">Port</Label>
              <Input
                id="port"
                type="number"
                value={formData.port}
                onChange={(e) => handleInputChange('port', parseInt(e.target.value) || 0)}
                min={1}
                max={65535}
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="healthCheckPath">Health Check Path</Label>
              <Input
                id="healthCheckPath"
                value={formData.healthCheckPath}
                onChange={(e) => handleInputChange('healthCheckPath', e.target.value)}
                placeholder="/health"
                className="font-mono"
                disabled={isLoading}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Resource Limits */}
      <Card>
        <CardHeader>
          <CardTitle>Resource Limits</CardTitle>
          <CardDescription>
            Configure CPU and memory limits
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="cpu">CPU Request</Label>
              <Input
                id="cpu"
                value={formData.resources.cpu}
                onChange={(e) => handleInputChange('resources.cpu', e.target.value)}
                placeholder="100m"
                className="font-mono"
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="cpuLimit">CPU Limit</Label>
              <Input
                id="cpuLimit"
                value={formData.resources.cpuLimit}
                onChange={(e) => handleInputChange('resources.cpuLimit', e.target.value)}
                placeholder="500m"
                className="font-mono"
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="memory">Memory Request</Label>
              <Input
                id="memory"
                value={formData.resources.memory}
                onChange={(e) => handleInputChange('resources.memory', e.target.value)}
                placeholder="128Mi"
                className="font-mono"
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="memoryLimit">Memory Limit</Label>
              <Input
                id="memoryLimit"
                value={formData.resources.memoryLimit}
                onChange={(e) => handleInputChange('resources.memoryLimit', e.target.value)}
                placeholder="512Mi"
                className="font-mono"
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
            Additional tool configuration
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Enable Tool</Label>
              <p className="text-sm text-muted-foreground">
                Allow this tool to be used by agents
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
                Require admin approval before using this tool
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
          {isLoading ? (isEdit ? 'Updating...' : 'Creating...') : (isEdit ? 'Update Tool' : 'Create Tool')}
        </Button>
      </div>
    </form>
  )
}