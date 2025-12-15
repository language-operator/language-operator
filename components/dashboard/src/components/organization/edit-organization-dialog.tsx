'use client'

import { useState, useEffect } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { OrganizationForm, OrganizationFormData } from './organization-form'
import { useUpdateOrganization } from '@/hooks/use-organizations'
import { toast } from 'sonner'
import type { Organization } from '@/store/organization-store'

interface EditOrganizationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  organization: Organization | null
}

export function EditOrganizationDialog({
  open,
  onOpenChange,
  organization
}: EditOrganizationDialogProps) {
  const [isLoading, setIsLoading] = useState(false)
  const updateOrganization = useUpdateOrganization(organization?.id || '')

  const handleSubmit = async (data: OrganizationFormData) => {
    if (!organization) return
    
    setIsLoading(true)
    
    try {
      updateOrganization.mutate({ 
        name: data.name
      }, {
        onSuccess: () => {
          // Close the dialog
          onOpenChange(false)
          
          // Show success toast
          toast.success('Organization updated successfully!')
          
          setIsLoading(false)
        },
        onError: (error: any) => {
          console.error('Error updating organization:', error)
          
          // Handle different error cases
          if (error.message.includes('already exists')) {
            toast.error('Organization with this slug or namespace already exists')
          } else if (error.message.includes('Invalid input') || error.message.includes('validation')) {
            toast.error(`Validation error: ${error.message}`)
          } else if (error.message.includes('Forbidden')) {
            toast.error('You do not have permission to edit this organization')
          } else {
            toast.error(error.message || 'Failed to update organization. Please try again.')
          }
          
          setIsLoading(false)
        }
      })
    } catch (error) {
      console.error('Unexpected error:', error)
      toast.error('An unexpected error occurred. Please try again.')
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    if (!isLoading) {
      onOpenChange(false)
    }
  }

  // Don't render if no organization is provided
  if (!organization) {
    return null
  }

  const initialData: Partial<OrganizationFormData> = {
    name: organization.name
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Organization</DialogTitle>
          <DialogDescription>
            Update your organization settings. Changes to the namespace will affect
            how your Kubernetes resources are organized.
          </DialogDescription>
        </DialogHeader>
        
        <OrganizationForm
          mode="edit"
          initialData={initialData}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          isLoading={isLoading}
        />
      </DialogContent>
    </Dialog>
  )
}