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
import { Copy, CheckCircle } from 'lucide-react'

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
  const [invitationUrl, setInvitationUrl] = useState<string | null>(null)
  const [copiedEmail, setCopiedEmail] = useState<string | null>(null)
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
      const result = await createInvite.mutateAsync(data)
      
      // Show the invitation URL
      if (result.invite?.invitationUrl) {
        setInvitationUrl(result.invite.invitationUrl)
        setCopiedEmail(data.email)
        toast.success(`Invitation created for ${data.email}`)
      } else {
        // Fallback to old behavior if URL not available
        onOpenChange(false)
        form.reset()
        toast.success(`Invitation sent to ${data.email}`)
      }
      
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
      setInvitationUrl(null)
      setCopiedEmail(null)
    }
  }

  const handleCopyInvitationUrl = async () => {
    if (!invitationUrl) return
    
    try {
      await navigator.clipboard.writeText(invitationUrl)
      toast.success('Invitation link copied to clipboard')
    } catch (error) {
      console.error('Failed to copy to clipboard:', error)
      toast.error('Failed to copy link to clipboard')
    }
  }

  const handleDone = () => {
    onOpenChange(false)
    form.reset()
    setInvitationUrl(null)
    setCopiedEmail(null)
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
        {!invitationUrl ? (
          <>
            <DialogHeader>
              <DialogTitle>Invite Team Member</DialogTitle>
              <DialogDescription>
                Send an invitation to add someone to your organization.
                They'll receive a shareable link to join.
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
                    {isLoading ? 'Creating...' : 'Create Invitation'}
                  </Button>
                </div>
              </form>
            </Form>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-green-600" />
                Invitation Created
              </DialogTitle>
              <DialogDescription>
                Share this link with {copiedEmail} to invite them to join your organization.
                The link expires in 7 days and can only be used once.
              </DialogDescription>
            </DialogHeader>
            
            <div className="space-y-4">
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium text-gray-700 mb-1">Invitation Link</p>
                    <p className="text-sm text-gray-600 font-mono truncate">
                      {invitationUrl}
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopyInvitationUrl}
                    className="shrink-0"
                  >
                    <Copy className="mr-1 h-3 w-3" />
                    Copy
                  </Button>
                </div>
              </div>
              
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                <p className="text-xs text-blue-800">
                  💡 <strong>Next steps:</strong> Share this link with the person you're inviting. 
                  They can click it to accept the invitation and join your organization.
                </p>
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <Button
                  variant="outline"
                  onClick={handleCopyInvitationUrl}
                >
                  <Copy className="mr-2 h-4 w-4" />
                  Copy Link
                </Button>
                <Button onClick={handleDone}>
                  Done
                </Button>
              </div>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}