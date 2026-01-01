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
  onSuccess?: () => void
}

export function InviteMemberDialog({
  open,
  onOpenChange,
  organizationId,
  onSuccess
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
      
      // Call onSuccess callback to refresh parent data
      onSuccess?.()
      
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
      <DialogContent className="max-w-2xl">
        {!invitationUrl ? (
          <>
            <DialogHeader>
              <DialogTitle>Invite Team Member</DialogTitle>
              <DialogDescription>
                Generate a shareable invitation link.
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
                          <SelectItem value="admin">Admin</SelectItem>
                          <SelectItem value="editor">Editor</SelectItem>
                          <SelectItem value="viewer">Viewer</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                      {form.watch('role') && (
                        <p className="text-xs text-stone-600 dark:text-stone-400">
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
                The link expires in 7 days and can only be used once.
              </DialogDescription>
            </DialogHeader>
            
            <div className="space-y-4">
              <div className="space-y-3">
                <p className="text-xs font-medium text-stone-700 dark:text-stone-300">Invitation Link</p>
                <div className="relative">
                  <Input
                    value={invitationUrl}
                    disabled
                    className="font-mono text-sm pr-10"
                  />
                  <button
                    onClick={handleCopyInvitationUrl}
                    className="absolute right-3 top-1/2 -translate-y-1/2 p-1 hover:bg-stone-100 dark:hover:bg-stone-700 rounded transition-colors"
                  >
                    <Copy className="h-4 w-4 text-stone-500 dark:text-stone-400" />
                  </button>
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-4">
                <Button
                  variant="outline"
                  onClick={handleCopyInvitationUrl}
                >
                  <Copy className="mr-2 h-4 w-4" />
                  Copy Link
                </Button>
                <Button onClick={handleDone}>
                  Close
                </Button>
              </div>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}