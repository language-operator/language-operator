'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Server, Globe, Shield, AlertCircle } from 'lucide-react'

export interface ClusterFormData {
  name: string
  domain: string
  enableTLS: boolean
  gatewayName?: string
  ingressClassName?: string
}

interface ClusterFormProps {
  initialData?: Partial<ClusterFormData>
  isLoading?: boolean
  error?: string
  onSubmit: (data: ClusterFormData) => Promise<void>
  onCancel: () => void
  isEdit?: boolean
}

export function ClusterForm({ 
  initialData, 
  isLoading = false, 
  error, 
  onSubmit, 
  onCancel,
  isEdit = false 
}: ClusterFormProps) {
  const [formData, setFormData] = useState<ClusterFormData>({
    name: '',
    domain: '',
    enableTLS: true,
    gatewayName: '',
    ingressClassName: '',
    ...initialData
  })

  const [validationError, setValidationError] = useState('')

  // Update form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      setFormData(prev => ({ ...prev, ...initialData }))
    }
  }, [initialData])

  const handleInputChange = (field: keyof ClusterFormData, value: string | boolean) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }))
    
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('')
    }
  }

  const validateForm = () => {
    if (!formData.name.trim()) {
      setValidationError('Cluster name is required')
      return false
    }
    
    // Validate cluster name (DNS-compatible)
    const nameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/
    if (!nameRegex.test(formData.name)) {
      setValidationError('Cluster name must be lowercase alphanumeric with hyphens')
      return false
    }
    
    if (formData.name.length > 63) {
      setValidationError('Cluster name must be 63 characters or less')
      return false
    }

    // Validate domain if provided
    if (formData.domain && formData.domain.trim()) {
      const domainRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$/
      if (!domainRegex.test(formData.domain)) {
        setValidationError('Invalid domain format')
        return false
      }
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
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Server className="h-5 w-5" />
            <span>Basic Information</span>
          </CardTitle>
          <CardDescription>
            Configure the basic settings for your language cluster
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Cluster Name *</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="my-cluster"
              className="font-mono"
              disabled={isEdit || isLoading} // Don't allow name changes in edit mode
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

        </CardContent>
      </Card>

      {/* Network Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Globe className="h-5 w-5" />
            <span>Network Configuration</span>
          </CardTitle>
          <CardDescription>
            Configure domain and network settings
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="domain">Domain (Optional)</Label>
            <Input
              id="domain"
              value={formData.domain}
              onChange={(e) => handleInputChange('domain', e.target.value)}
              placeholder="cluster.example.com"
              className="font-mono"
              disabled={isLoading}
            />
            <p className="text-sm text-muted-foreground">
              Custom domain for accessing agents in this cluster
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="gatewayName">Gateway Name (Optional)</Label>
            <Input
              id="gatewayName"
              value={formData.gatewayName}
              onChange={(e) => handleInputChange('gatewayName', e.target.value)}
              placeholder="my-gateway"
              className="font-mono"
              disabled={isLoading}
            />
            <p className="text-sm text-muted-foreground">
              Name of the Gateway API gateway to use
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="ingressClassName">Ingress Class Name (Optional)</Label>
            <Input
              id="ingressClassName"
              value={formData.ingressClassName}
              onChange={(e) => handleInputChange('ingressClassName', e.target.value)}
              placeholder="nginx"
              className="font-mono"
              disabled={isLoading}
            />
            <p className="text-sm text-muted-foreground">
              Fallback ingress class when Gateway API is not available
            </p>
          </div>

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Enable TLS</Label>
              <p className="text-sm text-muted-foreground">
                Enable TLS/SSL encryption for secure connections
              </p>
            </div>
            <Switch
              checked={formData.enableTLS}
              onCheckedChange={(checked) => handleInputChange('enableTLS', checked)}
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
          {isLoading ? (isEdit ? 'Updating...' : 'Creating...') : (isEdit ? 'Update Cluster' : 'Create Cluster')}
        </Button>
      </div>
    </form>
  )
}