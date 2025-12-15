'use client'

import { useState, useEffect } from 'react'
import { Users, MoreHorizontal, UserPlus, Trash2, Edit, ChevronDown } from 'lucide-react'
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
import { AddUserDialog } from '@/components/users/add-user-dialog'

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


export default function UsersPage() {
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [showAddUser, setShowAddUser] = useState(false)
  const [users, setUsers] = useState<User[]>([])
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [currentUser, setCurrentUser] = useState<User | null>(null)
  const [currentOrganization, setCurrentOrganization] = useState<Organization | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch users and organizations on component mount
  useEffect(() => {
    const fetchData = async () => {
      try {
        setIsLoading(true)
        
        // Fetch users
        const usersResponse = await fetch('/api/users')
        if (!usersResponse.ok) {
          throw new Error('Failed to fetch users')
        }
        const usersData = await usersResponse.json()
        setUsers(usersData)

        // Fetch organizations
        const orgsResponse = await fetch('/api/organizations')
        if (!orgsResponse.ok) {
          throw new Error('Failed to fetch organizations')
        }
        const orgsData = await orgsResponse.json()
        setOrganizations(orgsData.organizations || [])
        
        // Set current organization (assuming first org for now)
        if (orgsData.organizations && orgsData.organizations.length > 0) {
          setCurrentOrganization(orgsData.organizations[0])
        }

        // Find current user from the users list (assuming it's based on session)
        // For now, we'll identify the current user as the one with 'owner' role
        const currentUserInList = usersData.find((user: any) => 
          user.memberships.some((m: any) => m.role === 'owner')
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
  }, [])

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'owner':
        return 'bg-purple-100 text-purple-800 hover:bg-purple-100'
      case 'admin':
        return 'bg-blue-100 text-blue-800 hover:bg-blue-100'
      case 'editor':
        return 'bg-green-100 text-green-800 hover:bg-green-100'
      case 'viewer':
        return 'bg-gray-100 text-gray-800 hover:bg-gray-100'
      default:
        return 'bg-gray-100 text-gray-800 hover:bg-gray-100'
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
      
      const response = await fetch(`/api/users/${userId}`, {
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

  const handleAddUser = async (userData: { name: string; email: string; password: string }) => {
    try {
      const response = await fetch('/api/users', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(userData),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create user')
      }

      const result = await response.json()
      
      // Add the new user to the local state
      setUsers(prevUsers => [...prevUsers, result.user])
      
      setShowAddUser(false)
    } catch (error) {
      console.error('Error creating user:', error)
      setError(error instanceof Error ? error.message : 'Failed to create user')
      throw error // Re-throw to let the modal handle the error state
    }
  }

  const handleChangeRole = async (userId: string, newRole: string) => {
    try {
      // Find the user to update their membership for the current organization
      const user = users.find(u => u.id === userId)
      if (!user || !currentOrganization) {
        throw new Error('User or organization not found')
      }

      // Update the membership with the new role
      const updatedMemberships = user.memberships.map(membership => 
        membership.organizationId === currentOrganization.id
          ? { ...membership, role: newRole as any }
          : membership
      )

      const response = await fetch(`/api/users/${userId}`, {
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


  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Users</h1>
            <p className="text-gray-600">Manage user access and organization memberships</p>
          </div>
        </div>
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-4"></div>
          <div className="space-y-3">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-200 rounded"></div>
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
            <h1 className="text-2xl font-bold">Users</h1>
            <p className="text-gray-600">Manage user access and organization memberships</p>
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
          <p className="text-gray-600">Manage user access and organization memberships</p>
        </div>
        <Button onClick={() => setShowAddUser(true)}>
          <UserPlus className="mr-2 h-4 w-4" />
          Add User
        </Button>
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
                  <TableHead>Last Seen</TableHead>
                  <TableHead className="w-[70px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8 text-gray-500">
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
                            <div className="text-sm text-gray-500">{user.email}</div>
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
                                    className={primaryMembership.role === 'owner' ? 'text-gray-400' : ''}
                                  >
                                    Owner
                                  </DropdownMenuItem>
                                  <DropdownMenuItem 
                                    onClick={() => handleChangeRole(user.id, 'admin')}
                                    disabled={primaryMembership.role === 'admin'}
                                    className={primaryMembership.role === 'admin' ? 'text-gray-400' : ''}
                                  >
                                    Admin
                                  </DropdownMenuItem>
                                  <DropdownMenuItem 
                                    onClick={() => handleChangeRole(user.id, 'editor')}
                                    disabled={primaryMembership.role === 'editor'}
                                    className={primaryMembership.role === 'editor' ? 'text-gray-400' : ''}
                                  >
                                    Editor
                                  </DropdownMenuItem>
                                  <DropdownMenuItem 
                                    onClick={() => handleChangeRole(user.id, 'viewer')}
                                    disabled={primaryMembership.role === 'viewer'}
                                    className={primaryMembership.role === 'viewer' ? 'text-gray-400' : ''}
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
                        <TableCell className="text-sm text-gray-500">
                          {formatLastSeen(user.lastSeen)}
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

      {/* Add User Dialog */}
      <AddUserDialog
        open={showAddUser}
        onOpenChange={setShowAddUser}
        onSave={handleAddUser}
      />


      {/* Edit User Dialog */}
      <EditUserDialog
        user={editingUser}
        memberships={editingUser ? editingUser.memberships : []}
        availableOrganizations={organizations}
        open={!!editingUser}
        onOpenChange={(open) => !open && setEditingUser(null)}
        onSave={handleSaveUser}
      />
    </div>
  )
}