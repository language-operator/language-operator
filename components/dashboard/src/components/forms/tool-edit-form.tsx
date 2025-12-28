'use client'

import { useState, useRef, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Settings, AlertCircle, Server, Code } from 'lucide-react'

export interface ToolEditFormData {
  image: string
  port: number
  deploymentMode: 'service' | 'sidecar'
  envVars: Array<{key: string, value: string}>
  resources: {
    cpu: string
    memory: string
    cpuLimit: string
    memoryLimit: string
  }
}

interface ToolEditFormProps {
  initialData?: Partial<ToolEditFormData>
  isLoading?: boolean
  error?: string
  onSubmit: (data: ToolEditFormData) => Promise<void>
  onCancel: () => void
}

export function ToolEditForm({ 
  initialData, 
  isLoading = false, 
  error, 
  onSubmit, 
  onCancel
}: ToolEditFormProps) {
  const [formData, setFormData] = useState<ToolEditFormData>(() => ({
    image: '',
    port: 3000,
    deploymentMode: 'service',
    envVars: [],
    resources: {
      cpu: '100m',
      memory: '128Mi',
      cpuLimit: '500m',
      memoryLimit: '512Mi'
    },
    ...initialData
  }))

  const [validationError, setValidationError] = useState('')
  const [hasInitialized, setHasInitialized] = useState(false)

  // Update form when initialData changes - only once when component loads
  useEffect(() => {
    if (initialData && !hasInitialized) {
      setFormData({
        image: '',
        port: 3000,
        deploymentMode: 'service',
        envVars: [],
        resources: {
          cpu: '100m',
          memory: '128Mi',
          cpuLimit: '500m',
          memoryLimit: '512Mi'
        },
        ...initialData
      })
      setHasInitialized(true)
    }
  }, [initialData, hasInitialized])

  const handleInputChange = (field: keyof ToolEditFormData | string, value: string | number) => {
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
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
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
    if (!formData.image.trim()) {
      setValidationError('Container image is required')
      return false
    }

    if (formData.port && (formData.port < 1 || formData.port > 65535)) {
      setValidationError('Port must be between 1 and 65535')
      return false
    }

    // Validate resource values (basic pattern check)
    const resourcePattern = /^\d+(\.\d+)?(m|Mi|Gi|Ki|[MGK]i?)$/
    if (formData.resources.cpu && !resourcePattern.test(formData.resources.cpu)) {
      setValidationError('CPU request must be a valid resource quantity (e.g., 100m, 1, 500m)')
      return false
    }

    if (formData.resources.memory && !resourcePattern.test(formData.resources.memory)) {
      setValidationError('Memory request must be a valid resource quantity (e.g., 128Mi, 1Gi)')
      return false
    }

    if (formData.resources.cpuLimit && !resourcePattern.test(formData.resources.cpuLimit)) {
      setValidationError('CPU limit must be a valid resource quantity (e.g., 500m, 2)')
      return false
    }

    if (formData.resources.memoryLimit && !resourcePattern.test(formData.resources.memoryLimit)) {
      setValidationError('Memory limit must be a valid resource quantity (e.g., 512Mi, 2Gi)')
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

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Settings className="h-5 w-5" />
            <span>Configuration</span>
          </CardTitle>
          <CardDescription>
            Update container image, ports, and health check settings
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="image">Container Image *</Label>
            <Input
              id="image"
              value={formData.image}
              onChange={(e) => handleInputChange('image', e.target.value)}
              placeholder="ghcr.io/org/tool:latest"
              className="font-mono"
              required
              disabled={isLoading}
            />
            <p className="text-sm text-muted-foreground">
              Container image to run for this tool
            </p>
          </div>

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
              <p className="text-sm text-muted-foreground">
                Port the service listens on
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="deploymentMode">Deployment Mode</Label>
              <Select 
                value={formData.deploymentMode} 
                onValueChange={(value) => handleInputChange('deploymentMode', value as 'service' | 'sidecar')}
                disabled={isLoading}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select deployment mode" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="service">Service</SelectItem>
                  <SelectItem value="sidecar">Sidecar</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-sm text-muted-foreground">
                How the tool should be deployed
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Environment Variables */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Code className="h-5 w-5" />
            <span>Environment Variables</span>
          </CardTitle>
          <CardDescription>
            Configure environment variables for the tool container
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <div className="flex justify-between items-center">
              <Label>Variables</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addEnvVar}
                disabled={isLoading}
              >
                Add Variable
              </Button>
            </div>
            
            {formData.envVars.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No environment variables configured
              </p>
            ) : (
              <div className="space-y-2">
                {formData.envVars.map((envVar, index) => (
                  <div key={index} className="grid grid-cols-5 gap-2">
                    <div className="col-span-2">
                      <Input
                        value={envVar.key}
                        onChange={(e) => updateEnvVar(index, 'key', e.target.value)}
                        placeholder="Variable name"
                        className="font-mono"
                        disabled={isLoading}
                      />
                    </div>
                    <div className="col-span-2">
                      <Input
                        value={envVar.value}
                        onChange={(e) => updateEnvVar(index, 'value', e.target.value)}
                        placeholder="Variable value"
                        className="font-mono"
                        disabled={isLoading}
                      />
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeEnvVar(index)}
                      disabled={isLoading}
                      className="text-red-500"
                    >
                      Remove
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Resource Limits */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Server className="h-5 w-5" />
            <span>Resource Limits</span>
          </CardTitle>
          <CardDescription>
            Configure CPU and memory requests and limits
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
              <p className="text-sm text-muted-foreground">
                Minimum CPU guaranteed (e.g., 100m, 0.5, 1)
              </p>
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
              <p className="text-sm text-muted-foreground">
                Maximum CPU allowed (e.g., 500m, 2)
              </p>
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
              <p className="text-sm text-muted-foreground">
                Minimum memory guaranteed (e.g., 128Mi, 1Gi)
              </p>
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
              <p className="text-sm text-muted-foreground">
                Maximum memory allowed (e.g., 512Mi, 2Gi)
              </p>
            </div>
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
          {isLoading ? 'Updating...' : 'Update Tool'}
        </Button>
      </div>
    </form>
  )
}