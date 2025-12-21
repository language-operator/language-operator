'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog'
import { ResourceHeader } from '@/components/ui/resource-header'
import { useOrganization, useActiveOrganization } from '@/hooks/use-organizations'
import { useOrganizationStore } from '@/store/organization-store'
import { toast } from 'sonner'
import { Building2, Trash2 } from 'lucide-react'

export default function OrganizationSettingsPage() {
  const params = useParams()
  const router = useRouter()
  const organizationId = params.id as string
  
  const { data: organizationData, isLoading } = useOrganization(organizationId)
  const organization = organizationData?.organization
  const userRole = organizationData?.userRole
  
  const { organization: activeOrganization } = useActiveOrganization()
  const { setActiveOrganization } = useOrganizationStore()
  const [isDeleting, setIsDeleting] = useState(false)

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
        <div className="h-6 bg-gray-200 rounded w-1/4 animate-pulse"></div>
        <div className="space-y-4">
          <div className="h-32 bg-gray-200 rounded animate-pulse"></div>
        </div>
      </div>
    )
  }

  if (!organization) {
    return <div>Organization not found</div>
  }

  const isOwner = userRole === 'owner'

  return (
    <div className="space-y-6">
      <div className="grid gap-6">
        {/* Organization Info */}
        <Card>
          <CardHeader>
            <CardTitle>Organization Information</CardTitle>
            <CardDescription>
              Basic information about your organization
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm font-medium text-gray-700">Name</label>
                <p className="text-sm text-gray-900">{organization.name}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700">Plan</label>
                <p className="text-sm text-gray-900 capitalize">{organization.plan}</p>
              </div>
            </div>
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
                  <p className="text-xs text-gray-600 mb-3">
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