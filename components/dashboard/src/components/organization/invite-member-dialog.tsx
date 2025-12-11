'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { useCreateInvite } from '@/hooks/use-organizations'
import { toast } from 'sonner'

const inviteMemberSchema = z.object({
  email: z.string().email('Please enter a valid email address'),
  role: z.enum(['admin', 'editor', 'viewer'])
})

type InviteMemberFormData = z.infer<typeof inviteMemberSchema>

interface InviteMemberDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  organizationId: string
}

export function InviteMemberDialog({
  open,
  onOpenChange,
  organizationId
}: InviteMemberDialogProps) {
  const [isLoading, setIsLoading] = useState(false)
  const createInvite = useCreateInvite(organizationId)

  const form = useForm<InviteMemberFormData>({
    resolver: zodResolver(inviteMemberSchema),
    defaultValues: {
      email: '',
      role: 'viewer'
    }
  })

  const handleSubmit = async (data: InviteMemberFormData) => {
    setIsLoading(true)
    
    try {
      await createInvite.mutateAsync(data)
      
      // Close the dialog and reset form
      onOpenChange(false)
      form.reset()
      
      // Show success toast
      toast.success(`Invitation sent to ${data.email}`)
      
      setIsLoading(false)
    } catch (error: any) {
      console.error('Error sending invitation:', error)
      
      // Handle specific error cases
      if (error.message.includes('already a member')) {
        toast.error('This user is already a member of the organization')
      } else if (error.message.includes('already exists')) {
        toast.error('A pending invitation already exists for this email')
      } else if (error.message.includes('not found')) {
        toast.error('User not found. They may need to create an account first.')
      } else {
        toast.error(error.message || 'Failed to send invitation. Please try again.')
      }
      
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    if (!isLoading) {
      onOpenChange(false)
      form.reset()
    }
  }

  const getRoleDescription = (role: string) => {
    switch (role) {
      case 'admin':
        return 'Can manage members and all resources'
      case 'editor':
        return 'Can create and edit resources'
      case 'viewer':
        return 'Read-only access to resources'
      default:
        return ''
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Invite Team Member</DialogTitle>
          <DialogDescription>
            Send an invitation to add someone to your organization.
            They'll receive an email with a link to join.
          </DialogDescription>
        </DialogHeader>
        
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email Address</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      placeholder="colleague@example.com"
                      disabled={isLoading}
                      type="email"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="role"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Role</FormLabel>
                  <Select
                    onValueChange={field.onChange}
                    defaultValue={field.value}
                    disabled={isLoading}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a role" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="admin">
                        <div>
                          <div className="font-medium">Admin</div>
                          <div className="text-xs text-gray-500">
                            Can manage members and all resources
                          </div>
                        </div>
                      </SelectItem>
                      <SelectItem value="editor">
                        <div>
                          <div className="font-medium">Editor</div>
                          <div className="text-xs text-gray-500">
                            Can create and edit resources
                          </div>
                        </div>
                      </SelectItem>
                      <SelectItem value="viewer">
                        <div>
                          <div className="font-medium">Viewer</div>
                          <div className="text-xs text-gray-500">
                            Read-only access to resources
                          </div>
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                  {form.watch('role') && (
                    <p className="text-xs text-gray-600">
                      {getRoleDescription(form.watch('role'))}
                    </p>
                  )}
                </FormItem>
              )}
            />

            <div className="flex justify-end gap-3 pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={handleCancel}
                disabled={isLoading}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Sending...' : 'Send Invitation'}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}