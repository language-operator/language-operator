'use client'

import { useState, useEffect } from 'react'
import { Users, MoreHorizontal, UserPlus, Trash2, Edit, ChevronDown, Copy, Mail } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { EditUserDialog } from '@/components/users/edit-user-dialog'
import { InviteMemberDialog } from '@/components/organization/invite-member-dialog'
import { toast } from 'sonner'
import { config } from '@/lib/config'
import { useOrganization } from '@/components/organization-provider'
import { fetchWithOrganization } from '@/lib/api-client'

interface User {
  id: string
  name: string
  email: string
  image?: string | null
  status: 'active' | 'inactive' | 'suspended'
  lastSeen: string | Date
  createdAt: Date
  updatedAt: Date
  memberships: {
    organizationId: string
    organizationName: string
    role: 'owner' | 'admin' | 'editor' | 'viewer'
  }[]
}

interface Organization {
  id: string
  name: string
}

interface Invite {
  id: string
  email: string
  role: string
  token: string
  expiresAt: string
  createdAt: string
}


export default function UsersPage() {
  const { organization, orgId } = useOrganization()
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [showInviteMember, setShowInviteMember] = useState(false)
  const [users, setUsers] = useState<User[]>([])
  const [invites, setInvites] = useState<Invite[]>([])
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [currentUser, setCurrentUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch users and organizations on component mount
  useEffect(() => {
    const fetchData = async () => {
      if (!organization || !orgId) return
      
      try {
        setIsLoading(true)
        
        // Fetch users for this organization
        const usersResponse = await fetchWithOrganization('/api/users')
        if (!usersResponse.ok) {
          throw new Error('Failed to fetch users')
        }
        const usersData = await usersResponse.json()
        setUsers(usersData)

        // Fetch organizations (for edit dialogs)
        const orgsResponse = await fetchWithOrganization('/api/organizations')
        if (!orgsResponse.ok) {
          throw new Error('Failed to fetch organizations')
        }
        const orgsData = await orgsResponse.json()
        setOrganizations(orgsData.organizations || [])
        
        // Fetch invitations for the current organization
        const invitesResponse = await fetchWithOrganization(`/api/organizations/${orgId}/invites`)
        if (invitesResponse.ok) {
          const invitesData = await invitesResponse.json()
          setInvites(invitesData.invites || [])
        }

        // Find current user from the users list
        const currentUserInList = usersData.find((user: any) => 
          user.memberships.some((m: any) => m.organizationId === orgId)
        )
        if (currentUserInList) {
          setCurrentUser(currentUserInList)
        }

      } catch (err) {
        setError(err instanceof Error ? err.message : 'An error occurred')
      } finally {
        setIsLoading(false)
      }
    }

    fetchData()
  }, [organization, orgId])

  const refreshInvites = async () => {
    if (orgId) {
      try {
        const invitesResponse = await fetchWithOrganization(`/api/organizations/${orgId}/invites`)
        if (invitesResponse.ok) {
          const invitesData = await invitesResponse.json()
          setInvites(invitesData.invites || [])
        }
      } catch (error) {
        console.error('Failed to refresh invites:', error)
      }
    }
  }

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'owner':
        return 'bg-purple-100 text-purple-800 hover:bg-purple-100'
      case 'admin':
        return 'bg-blue-100 text-blue-800 hover:bg-blue-100'
      case 'editor':
        return 'bg-green-100 text-green-800 hover:bg-green-100'
      case 'viewer':
        return 'bg-stone-100 text-stone-800 hover:bg-stone-100'
      default:
        return 'bg-stone-100 text-stone-800 hover:bg-stone-100'
    }
  }

  const formatLastSeen = (lastSeen: string | Date) => {
    const date = new Date(lastSeen)
    const now = new Date()
    const diffInMs = now.getTime() - date.getTime()
    const diffInMinutes = Math.floor(diffInMs / (1000 * 60))
    
    if (diffInMinutes < 1) return 'Just now'
    if (diffInMinutes < 60) return `${diffInMinutes} minute${diffInMinutes > 1 ? 's' : ''} ago`
    
    const diffInHours = Math.floor(diffInMinutes / 60)
    if (diffInHours < 24) return `${diffInHours} hour${diffInHours > 1 ? 's' : ''} ago`
    
    const diffInDays = Math.floor(diffInHours / 24)
    return `${diffInDays} day${diffInDays > 1 ? 's' : ''} ago`
  }

  const handleEditUser = (user: User) => {
    setEditingUser(user)
  }

  const handleSaveUser = async (userId: string, userData: any, memberships: any[], updateMemberships: boolean = true) => {
    try {
      const requestBody: any = {
        name: userData.name,
        email: userData.email,
      }
      
      // Only include membership data if explicitly updating memberships
      if (updateMemberships) {
        requestBody.memberships = memberships
        requestBody.updateMemberships = true
      }
      
      const response = await fetchWithOrganization(`/api/users/${userId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update user')
      }

      const updatedUser = await response.json()
      
      // Update the user in the local state
      setUsers(prevUsers => 
        prevUsers.map(user => 
          user.id === userId ? updatedUser : user
        )
      )

      setEditingUser(null)
    } catch (error) {
      console.error('Error saving user:', error)
      setError(error instanceof Error ? error.message : 'Failed to save user')
    }
  }


  const handleChangeRole = async (userId: string, newRole: string) => {
    try {
      // Find the user to update their membership for the current organization
      const user = users.find(u => u.id === userId)
      if (!user || !orgId) {
        throw new Error('User or organization not found')
      }

      // Update the membership with the new role
      const updatedMemberships = user.memberships.map(membership => 
        membership.organizationId === orgId
          ? { ...membership, role: newRole as any }
          : membership
      )

      const response = await fetchWithOrganization(`/api/users/${userId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          memberships: updatedMemberships,
          updateMemberships: true
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update user role')
      }

      const updatedUser = await response.json()
      
      // Update the user in the local state
      setUsers(prevUsers => 
        prevUsers.map(user => 
          user.id === userId ? updatedUser : user
        )
      )

    } catch (error) {
      console.error('Error changing role:', error)
      setError(error instanceof Error ? error.message : 'Failed to change user role')
      throw error
    }
  }

  const handleCopyInviteLink = async (invite: Invite) => {
    try {
      const invitationUrl = `${config.dashboardUrl}/invites/${invite.token}`
      await navigator.clipboard.writeText(invitationUrl)
      toast.success('Invitation link copied to clipboard')
    } catch (error) {
      console.error('Failed to copy to clipboard:', error)
      toast.error('Failed to copy link to clipboard')
    }
  }

  const handleCancelInvitation = async (invite: Invite) => {
    if (!orgId) return

    try {
      const response = await fetchWithOrganization(`/api/organizations/${orgId}/invites/${invite.id}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to cancel invitation')
      }

      // Remove the invitation from local state
      setInvites(prevInvites => prevInvites.filter(i => i.id !== invite.id))
      toast.success(`Invitation to ${invite.email} has been cancelled`)
    } catch (error) {
      console.error('Error cancelling invitation:', error)
      toast.error(error instanceof Error ? error.message : 'Failed to cancel invitation')
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Users</h1>
            <p className="text-stone-600 dark:text-stone-400">Manage user access and organization memberships</p>
          </div>
        </div>
        <div className="animate-pulse">
          <div className="h-8 bg-stone-200 rounded w-1/4 mb-4"></div>
          <div className="space-y-3">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-16 bg-stone-200 rounded"></div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold font-mono">Users</h1>
            <p className="text-stone-600 dark:text-stone-400 mt-2">Manage user access and organization memberships</p>
          </div>
        </div>
        <Card>
          <CardContent className="p-6">
            <div className="text-center">
              <p className="text-red-600">Error loading users: {error}</p>
              <Button 
                onClick={() => window.location.reload()} 
                className="mt-4"
              >
                Try Again
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Users</h1>
          <p className="text-stone-600 dark:text-stone-400">Manage user access and organization memberships</p>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => setShowInviteMember(true)}>
            <UserPlus className="mr-2 h-4 w-4" />
            Invite Member
          </Button>
        </div>
      </div>


      {/* Users Tab */}
      <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Users className="h-5 w-5" />
              Active Users
            </CardTitle>
            <CardDescription>
              Users with access to this organization
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>User</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Created At</TableHead>
                  <TableHead className="w-[70px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8 text-stone-500 dark:text-stone-400">
                      No users found
                    </TableCell>
                  </TableRow>
                ) : (
                  users.map((user) => {
                    // Get primary organization for display (first one, or most senior role)
                    const primaryMembership = user.memberships.find(m => m.role === 'owner') || user.memberships[0]
                    const isCurrentUser = currentUser && currentUser.id === user.id
                    const isOwner = primaryMembership?.role === 'owner'
                    const canChangeRole = !isCurrentUser || !isOwner // Current user who is owner cannot change their own role
                    const canDelete = !isCurrentUser || !isOwner // Current user who is owner cannot delete themselves
                    
                    return (
                      <TableRow key={user.id}>
                        <TableCell className="flex items-center gap-3">
                          <Avatar className="h-8 w-8">
                            <AvatarImage src={user.image || undefined} />
                            <AvatarFallback>
                              {user.name.split(' ').map(n => n[0]).join('').toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <div>
                            <div className="font-medium">{user.name}</div>
                            <div className="text-sm text-stone-500 dark:text-stone-400">{user.email}</div>
                          </div>
                        </TableCell>
                        <TableCell>
                          {primaryMembership ? (
                            canChangeRole ? (
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <Badge 
                                    variant="secondary" 
                                    className={`${getRoleBadgeColor(primaryMembership.role)} cursor-pointer hover:opacity-80 flex items-center gap-1`}
                                  >
                                    {primaryMembership.role.charAt(0).toUpperCase() + primaryMembership.role.slice(1)}
                                    <ChevronDown className="h-3 w-3" />
                                  </Badge>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="start">
                                  <DropdownMenuItem 
                                    onClick={() => handleChangeRole(user.id, 'owner')}
                                    disabled={primaryMembership.role === 'owner'}
                                    className={primaryMembership.role === 'owner' ? 'text-stone-400' : ''}
                                  >
                                    Owner
                                  </DropdownMenuItem>
                                  <DropdownMenuItem 
                                    onClick={() => handleChangeRole(user.id, 'admin')}
                                    disabled={primaryMembership.role === 'admin'}
                                    className={primaryMembership.role === 'admin' ? 'text-stone-400' : ''}
                                  >
                                    Admin
                                  </DropdownMenuItem>
                                  <DropdownMenuItem 
                                    onClick={() => handleChangeRole(user.id, 'editor')}
                                    disabled={primaryMembership.role === 'editor'}
                                    className={primaryMembership.role === 'editor' ? 'text-stone-400' : ''}
                                  >
                                    Editor
                                  </DropdownMenuItem>
                                  <DropdownMenuItem 
                                    onClick={() => handleChangeRole(user.id, 'viewer')}
                                    disabled={primaryMembership.role === 'viewer'}
                                    className={primaryMembership.role === 'viewer' ? 'text-stone-400' : ''}
                                  >
                                    Viewer
                                  </DropdownMenuItem>
                                </DropdownMenuContent>
                              </DropdownMenu>
                            ) : (
                              <Badge 
                                variant="secondary" 
                                className={getRoleBadgeColor(primaryMembership.role)}
                              >
                                {primaryMembership.role.charAt(0).toUpperCase() + primaryMembership.role.slice(1)}
                              </Badge>
                            )
                          ) : '-'}
                        </TableCell>
                        <TableCell className="text-sm text-stone-500 dark:text-stone-400">
                          {new Date(user.createdAt).toLocaleDateString()}
                        </TableCell>
                        <TableCell>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem onClick={() => handleEditUser(user)}>
                                <Edit className="mr-2 h-4 w-4" />
                                Edit User
                              </DropdownMenuItem>
                              {canDelete ? (
                                <DropdownMenuItem className="text-red-600">
                                  <Trash2 className="mr-2 h-4 w-4" />
                                  Remove User
                                </DropdownMenuItem>
                              ) : (
                                <DropdownMenuItem disabled>
                                  <Trash2 className="mr-2 h-4 w-4" />
                                  Remove User
                                </DropdownMenuItem>
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

      {/* Pending Invitations */}
      {invites.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Mail className="h-5 w-5" />
              Pending Invitations ({invites.length})
            </CardTitle>
            <CardDescription>
              Invitations that have been sent but not yet accepted
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Invited</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead className="w-[70px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {invites.map((invite) => (
                  <TableRow key={invite.id}>
                    <TableCell className="flex items-center gap-3">
                      <Avatar className="h-8 w-8">
                        <AvatarFallback>
                          <Mail className="h-4 w-4" />
                        </AvatarFallback>
                      </Avatar>
                      <div>
                        <div className="font-medium">{invite.email}</div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge 
                        variant="secondary" 
                        className={getRoleBadgeColor(invite.role)}
                      >
                        {invite.role.charAt(0).toUpperCase() + invite.role.slice(1)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-stone-500 dark:text-stone-400">
                      {new Date(invite.createdAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-sm text-stone-500 dark:text-stone-400">
                      {new Date(invite.expiresAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem 
                            onClick={() => handleCopyInviteLink(invite)}
                          >
                            <Copy className="mr-2 h-4 w-4" />
                            Copy invitation link
                          </DropdownMenuItem>
                          <DropdownMenuItem 
                            className="text-red-600"
                            onClick={() => handleCancelInvitation(invite)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Cancel invitation
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Edit User Dialog */}
      <EditUserDialog
        user={editingUser}
        memberships={editingUser ? editingUser.memberships : []}
        availableOrganizations={organizations}
        open={!!editingUser}
        onOpenChange={(open) => !open && setEditingUser(null)}
        onSave={handleSaveUser}
      />

      {/* Invite Member Dialog */}
      {orgId && (
        <InviteMemberDialog
          open={showInviteMember}
          onOpenChange={setShowInviteMember}
          organizationId={orgId}
          onSuccess={refreshInvites}
        />
      )}
    </div>
  )
}