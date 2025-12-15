'use client'

import { useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { OrganizationForm, OrganizationFormData } from './organization-form'
import { useCreateOrganization } from '@/hooks/use-organizations'
import { useOrganizationStore } from '@/store/organization-store'
import { toast } from 'sonner'

interface CreateOrganizationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateOrganizationDialog({
  open,
  onOpenChange
}: CreateOrganizationDialogProps) {
  const [isLoading, setIsLoading] = useState(false)
  const { mutate: createOrganization } = useCreateOrganization()
  const { setActiveOrganization } = useOrganizationStore()

  const handleSubmit = async (data: OrganizationFormData) => {
    setIsLoading(true)
    
    try {
      // Transform form data to match API requirements
      const slug = data.name.toLowerCase().replace(/[^a-z0-9]/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '')
      const namespace = `org-${slug}`
      
      createOrganization({
        name: data.name,
        slug: slug,
        namespace: namespace,
        plan: 'free' as const
      }, {
        onSuccess: (response) => {
          // Close the dialog
          onOpenChange(false)
          
          // Switch to the newly created organization if requested
          if (data.switchToNew) {
            setActiveOrganization(response.organization.id)
          }
          
          // Show success toast
          toast.success('Organization created successfully!')
          
          setIsLoading(false)
        },
        onError: (error: any) => {
          console.error('Error creating organization:', error)
          
          // The error comes from our mutation function, so error.message contains the parsed error
          if (error.message.includes('already exists')) {
            toast.error('Organization with this slug or namespace already exists')
          } else if (error.message.includes('Invalid input') || error.message.includes('validation')) {
            toast.error(`Validation error: ${error.message}`)
          } else {
            toast.error(error.message || 'Failed to create organization. Please try again.')
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Create Organization</DialogTitle>
          <DialogDescription>
            What is the organization's name?
          </DialogDescription>
        </DialogHeader>
        
        <OrganizationForm
          mode="create"
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          isLoading={isLoading}
        />
      </DialogContent>
    </Dialog>
  )
}