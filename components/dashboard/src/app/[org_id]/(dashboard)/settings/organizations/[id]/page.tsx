'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog'
import { ResourceHeader } from '@/components/ui/resource-header'
import { useOrganization, useActiveOrganization } from '@/hooks/use-organizations'
import { useOrganizationStore } from '@/store/organization-store'
import { toast } from 'sonner'
import { Building2, Trash2, Save } from 'lucide-react'

// Organization edit form schema
const organizationFormSchema = z.object({
  name: z.string().min(1, 'Organization name is required').max(100, 'Organization name is too long'),
})

export default function OrganizationSettingsPage() {
  const params = useParams()
  const router = useRouter()
  const organizationId = params.id as string
  
  const { data: organizationData, isLoading, refetch } = useOrganization(organizationId)
  const organization = organizationData?.organization
  const userRole = organizationData?.userRole
  
  const { organization: activeOrganization } = useActiveOrganization()
  const { setActiveOrganization } = useOrganizationStore()
  const [isDeleting, setIsDeleting] = useState(false)
  const [isSaving, setIsSaving] = useState(false)

  // Organization edit form
  const form = useForm<z.infer<typeof organizationFormSchema>>({
    resolver: zodResolver(organizationFormSchema),
    defaultValues: {
      name: '',
    }
  })

  // Load organization data into form
  useEffect(() => {
    if (organization) {
      form.reset({
        name: organization.name,
      })
    }
  }, [organization, form])

  // Handle organization name update
  const onSubmit = async (values: z.infer<typeof organizationFormSchema>) => {
    if (!organization) return

    setIsSaving(true)
    try {
      const response = await fetch(`/api/organizations/${organization.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: values.name })
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to update organization')
      }

      const data = await response.json()
      toast.success('Organization updated successfully')
      
      // Refetch organization data to update the UI
      refetch()
      
      // Update active organization name if this is the active one
      if (activeOrganization?.id === organization.id) {
        setActiveOrganization({ ...activeOrganization, name: values.name })
      }
    } catch (error) {
      console.error('Error updating organization:', error)
      toast.error(error instanceof Error ? error.message : 'Failed to update organization')
    } finally {
      setIsSaving(false)
    }
  }

  const handleDeleteOrganization = async () => {
    if (!organization) return

    setIsDeleting(true)
    try {
      const response = await fetch(`/api/organizations/${organization.id}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to delete organization')
      }

      toast.success('Organization deleted successfully')
      
      // If this was the active organization, clear it
      if (activeOrganization?.id === organization.id) {
        setActiveOrganization(null)
      }
      
      // Redirect to organizations list
      router.push('/settings/organizations')
    } catch (error) {
      console.error('Error deleting organization:', error)
      toast.error(error instanceof Error ? error.message : 'Failed to delete organization')
    } finally {
      setIsDeleting(false)
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-6 bg-stone-200 rounded w-1/4 animate-pulse dark:bg-stone-700"></div>
        <div className="space-y-4">
          <div className="h-32 bg-stone-200 rounded animate-pulse dark:bg-stone-700"></div>
        </div>
      </div>
    )
  }

  if (!organization) {
    return <div>Organization not found</div>
  }

  const isOwner = userRole === 'owner'
  const canEditOrganization = userRole === 'owner' || userRole === 'admin'

  return (
    <div className="space-y-6">
      <div className="grid gap-6">
        {/* Organization Edit Form */}
        <Card>
          <CardHeader>
            <CardTitle>Organization Information</CardTitle>
            <CardDescription>
              Basic information about your organization
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Organization Name</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          disabled={!canEditOrganization || isSaving}
                          placeholder="Enter organization name"
                          className="max-w-md"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                
                {canEditOrganization && (
                  <div className="flex justify-start">
                    <Button
                      type="submit"
                      disabled={isSaving}
                      className="flex items-center gap-2"
                    >
                      <Save className="h-4 w-4" />
                      {isSaving ? 'Saving...' : 'Save Changes'}
                    </Button>
                  </div>
                )}
              </form>
            </Form>
          </CardContent>
        </Card>

        {/* Danger Zone */}
        {isOwner && (
          <Card className="border-destructive">
            <CardHeader>
              <CardTitle className="text-destructive">Danger Zone</CardTitle>
              <CardDescription>
                Irreversible and destructive actions
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-medium">Delete Organization</h4>
                  <p className="text-xs text-stone-600 dark:text-stone-400 mb-3">
                    Permanently delete this organization and all its resources. This action cannot be undone.
                  </p>
                  
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="destructive" size="sm">
                        <Trash2 className="h-4 w-4 mr-2" />
                        Delete Organization
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete Organization</AlertDialogTitle>
                        <AlertDialogDescription>
                          Are you sure you want to delete <strong>{organization.name}</strong>? 
                          This will permanently delete the organization and all associated resources including:
                          <ul className="list-disc list-inside mt-2 space-y-1">
                            <li>All clusters and agents</li>
                            <li>All models and tools</li>
                            <li>All personas and workflows</li>
                            <li>The Kubernetes namespace and its contents</li>
                            <li>All member access and invitations</li>
                          </ul>
                          <p className="mt-2 font-medium text-destructive">
                            This action cannot be undone.
                          </p>
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel disabled={isDeleting}>
                          Cancel
                        </AlertDialogCancel>
                        <AlertDialogAction
                          onClick={handleDeleteOrganization}
                          disabled={isDeleting}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                          {isDeleting ? 'Deleting...' : 'Delete Organization'}
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}