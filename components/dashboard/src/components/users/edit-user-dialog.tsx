'use client'

import { useState, useEffect } from 'react'
import { Check, X, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface User {
  id: string
  name: string
  email: string
  image?: string | null
}

interface OrganizationMembership {
  organizationId: string
  organizationName: string
  role: 'owner' | 'admin' | 'editor' | 'viewer'
}

interface EditUserDialogProps {
  user: User | null
  memberships: OrganizationMembership[]
  availableOrganizations: { id: string; name: string }[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (userId: string, userData: Partial<User>, memberships: OrganizationMembership[], updateMemberships?: boolean) => void
}

const roleColors = {
  owner: 'bg-purple-100 text-purple-800 hover:bg-purple-100',
  admin: 'bg-blue-100 text-blue-800 hover:bg-blue-100',
  editor: 'bg-green-100 text-green-800 hover:bg-green-100',
  viewer: 'bg-gray-100 text-gray-800 hover:bg-gray-100',
}

const roleOptions = [
  { value: 'owner', label: 'Owner' },
  { value: 'admin', label: 'Admin' },
  { value: 'editor', label: 'Editor' },
  { value: 'viewer', label: 'Viewer' },
]

export function EditUserDialog({
  user,
  memberships: initialMemberships,
  availableOrganizations,
  open,
  onOpenChange,
  onSave,
}: EditUserDialogProps) {
  const [memberships, setMemberships] = useState<OrganizationMembership[]>(initialMemberships)
  const [userData, setUserData] = useState<Partial<User>>({})
  const [isLoading, setIsLoading] = useState(false)
  const [membershipsModified, setMembershipsModified] = useState(false)

  // Reset state when user changes
  useEffect(() => {
    if (user) {
      setMemberships(initialMemberships)
      setUserData({
        name: user.name,
        email: user.email
      })
      setMembershipsModified(false)
    }
  }, [user?.id, initialMemberships])

  if (!user) return null

  const handleRoleChange = (organizationId: string, newRole: string) => {
    setMemberships(prev => 
      prev.map(membership => 
        membership.organizationId === organizationId
          ? { ...membership, role: newRole as OrganizationMembership['role'] }
          : membership
      )
    )
    setMembershipsModified(true)
  }

  const handleRemoveMembership = (organizationId: string) => {
    setMemberships(prev => 
      prev.filter(membership => membership.organizationId !== organizationId)
    )
    setMembershipsModified(true)
  }

  const handleAddMembership = (organizationId: string) => {
    const org = availableOrganizations.find(o => o.id === organizationId)
    if (org) {
      setMemberships(prev => [
        ...prev,
        {
          organizationId: org.id,
          organizationName: org.name,
          role: 'viewer'
        }
      ])
      setMembershipsModified(true)
    }
  }

  const handleSave = async () => {
    setIsLoading(true)
    try {
      await onSave(user.id, userData, memberships)
      onOpenChange(false)
    } catch (error) {
      console.error('Error saving user:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const availableToAdd = availableOrganizations.filter(
    org => !memberships.some(membership => membership.organizationId === org.id)
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Edit User</DialogTitle>
          <DialogDescription>
            Manage organization memberships and roles for this user
          </DialogDescription>
        </DialogHeader>

        {/* User Basic Information */}
        <Card>
          <CardHeader>
            <CardTitle>User Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-4 mb-4">
              <Avatar className="h-16 w-16">
                <AvatarImage src={user.image || undefined} />
                <AvatarFallback>
                  {(userData.name || user.name).split(' ').map(n => n[0]).join('').toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <div className="text-sm text-gray-500">
                User ID: {user.id}
              </div>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="name">Full Name</Label>
                <Input
                  id="name"
                  value={userData.name || ''}
                  onChange={(e) => setUserData(prev => ({ ...prev, name: e.target.value }))}
                  placeholder="Enter full name"
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="email">Email Address</Label>
                <Input
                  id="email"
                  type="email"
                  value={userData.email || ''}
                  onChange={(e) => setUserData(prev => ({ ...prev, email: e.target.value }))}
                  placeholder="Enter email address"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Organization Memberships */}
        <Card>
          <CardHeader>
            <CardTitle>Organization Memberships</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {memberships.length === 0 ? (
              <p className="text-gray-500 text-center py-4">
                This user is not a member of any organizations
              </p>
            ) : (
              memberships.map((membership) => (
                <div
                  key={membership.organizationId}
                  className="flex items-center justify-between p-3 border rounded-lg"
                >
                  <div className="flex items-center gap-3">
                    <span className="font-medium">{membership.organizationName}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Select
                      value={membership.role}
                      onValueChange={(value) => handleRoleChange(membership.organizationId, value)}
                    >
                      <SelectTrigger className="w-32">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {roleOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleRemoveMembership(membership.organizationId)}
                      className="text-red-600 hover:text-red-700"
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))
            )}

            {/* Add Organization */}
            {availableToAdd.length > 0 && (
              <div className="border-t pt-4">
                <p className="text-sm font-medium mb-2">Add to Organization:</p>
                <div className="flex flex-wrap gap-2">
                  {availableToAdd.map((org) => (
                    <Button
                      key={org.id}
                      variant="outline"
                      size="sm"
                      onClick={() => handleAddMembership(org.id)}
                      className="flex items-center gap-1"
                    >
                      <Plus className="h-3 w-3" />
                      {org.name}
                    </Button>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSave}
            disabled={isLoading}
          >
            {isLoading ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}