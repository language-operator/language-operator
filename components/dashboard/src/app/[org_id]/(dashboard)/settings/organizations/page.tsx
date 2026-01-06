'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Plus, Settings, Users, MoreHorizontal, Trash2, Edit } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useOrganizations, useActiveOrganization } from '@/hooks/use-organizations'
import { useOrganizationStore } from '@/store/organization-store'
import { CreateOrganizationDialog } from '@/components/organization/create-organization-dialog'
import { EditOrganizationDialog } from '@/components/organization/edit-organization-dialog'
import type { Organization } from '@/store/organization-store'
import { useOrganization } from '@/components/organization-provider'
import { fetchWithOrganization } from '@/lib/api-client'

export default function OrganizationsPage() {
  const router = useRouter()
  const { getOrgUrl } = useOrganization()
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [deletingOrganization, setDeletingOrganization] = useState<Organization | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  const { data: organizations = [], isLoading } = useOrganizations()
  const { organization: activeOrganization } = useActiveOrganization()
  const { setActiveOrganization } = useOrganizationStore()

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'owner':
        return 'bg-purple-100 text-purple-800 hover:bg-purple-100'
      case 'admin':
        return 'bg-blue-100 text-blue-800 hover:bg-blue-100'
      case 'editor':
        return 'bg-green-100 text-green-800 hover:bg-green-100'
      case 'viewer':
        return 'bg-stone-100 text-stone-800 hover:bg-stone-100 dark:bg-stone-700 dark:text-stone-300'
      default:
        return 'bg-stone-100 text-stone-800 hover:bg-stone-100 dark:bg-stone-700 dark:text-stone-300'
    }
  }

  const handleDeleteOrganization = async (organization: Organization) => {
    if (!organization) return

    setIsDeleting(true)
    try {
      const response = await fetchWithOrganization(`/api/organizations/${organization.id}`, {
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
      
      // Close the dialog and refresh the list (react-query will handle this)
      setDeletingOrganization(null)
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
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Organizations</h1>
            <p className="text-stone-600 dark:text-stone-400">Manage your organizations and switch between them</p>
          </div>
        </div>
        <div className="grid gap-4">
          {[...Array(3)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="p-6">
                <div className="h-4 bg-stone-200 w-1/4 mb-2 dark:bg-stone-700"></div>
                <div className="h-3 bg-stone-200 w-1/2 mb-4 dark:bg-stone-700"></div>
                <div className="flex gap-2">
                  <div className="h-6 bg-stone-200 w-16 dark:bg-stone-700"></div>
                  <div className="h-6 bg-stone-200 w-20 dark:bg-stone-700"></div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Organizations</h1>
          <p className="text-stone-600 dark:text-stone-400">Manage your organizations and switch between them</p>
        </div>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Organization
        </Button>
      </div>


      {/* Organizations Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            Active Organizations
          </CardTitle>
          <CardDescription>
            Manage your organizations and their settings
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Namespace</TableHead>
                <TableHead># Members</TableHead>
                <TableHead>Role</TableHead>
                <TableHead className="w-[70px]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {organizations.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-12">
                    <div className="flex flex-col items-center">
                      <div className="w-12 h-12 bg-stone-100 border border-stone-200 flex items-center justify-center mb-4 dark:bg-stone-800 dark:border-stone-700">
                        <Settings className="w-6 h-6 text-stone-400 dark:text-stone-500" />
                      </div>
                      <h3 className="text-lg font-semibold mb-2">No organizations</h3>
                      <p className="text-stone-600 dark:text-stone-400 mb-4">
                        Create your first organization to start managing Language Operator resources
                      </p>
                      <Button onClick={() => setShowCreateDialog(true)}>
                        <Plus className="mr-2 h-4 w-4" />
                        Create Organization
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                organizations.map((org) => {
                  const userMembership = org.members?.find(member => 
                    // This would need to match against current user
                    member.role
                  )
                  
                  const isActive = activeOrganization?.id === org.id
                  
                  return (
                    <TableRow key={org.id} className={isActive ? 'bg-stone-50 dark:bg-stone-800/50' : ''}>
                      <TableCell>
                        <div className="font-medium flex items-center gap-2">
                          <Link 
                            href={getOrgUrl(`/settings/organizations/${org.id}`)}
                            className="hover:underline"
                          >
                            {org.name}
                          </Link>
                          {isActive && (
                            <Badge variant="default" className="bg-blue-100 text-blue-800">
                              Selected
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {org.namespace || org.slug}
                      </TableCell>
                      <TableCell>
                        {org._count?.members || 0}
                      </TableCell>
                      <TableCell>
                        {userMembership && (
                          <Badge 
                            variant="secondary" 
                            className={getRoleBadgeColor(userMembership.role)}
                          >
                            {userMembership.role.charAt(0).toUpperCase() + userMembership.role.slice(1)}
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => setActiveOrganization(org.id)}>
                              <Users className="mr-2 h-4 w-4" />
                              Switch to Organization
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => router.push(getOrgUrl(`/settings/organizations/${org.id}`))}>
                              <Edit className="mr-2 h-4 w-4" />
                              Edit Organization
                            </DropdownMenuItem>
                            {userMembership?.role === 'owner' && (
                              <>
                                <DropdownMenuItem 
                                  className="text-red-600"
                                  onClick={() => setDeletingOrganization(org)}
                                >
                                  <Trash2 className="mr-2 h-4 w-4" />
                                  Delete Organization
                                </DropdownMenuItem>
                              </>
                            )}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      
      {/* Create Organization Dialog */}
      <CreateOrganizationDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
      />

      {/* Delete Organization Dialog */}
      <AlertDialog 
        open={!!deletingOrganization} 
        onOpenChange={(open) => !open && setDeletingOrganization(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Organization</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deletingOrganization?.name}</strong>? 
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
              onClick={() => deletingOrganization && handleDeleteOrganization(deletingOrganization)}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? 'Deleting...' : 'Delete Organization'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}