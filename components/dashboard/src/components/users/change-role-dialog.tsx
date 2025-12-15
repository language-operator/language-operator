'use client'

import { useState, useEffect } from 'react'
import { Check } from 'lucide-react'
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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

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

interface ChangeRoleDialogProps {
  user: User | null
  currentRole: string | null
  organizationName: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (userId: string, newRole: string) => void
}

const roleOptions = [
  { value: 'owner', label: 'Owner', description: 'Full access and organization management' },
  { value: 'admin', label: 'Admin', description: 'Administrative access to all features' },
  { value: 'editor', label: 'Editor', description: 'Can edit and manage content' },
  { value: 'viewer', label: 'Viewer', description: 'Read-only access' },
]

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

export function ChangeRoleDialog({
  user,
  currentRole,
  organizationName,
  open,
  onOpenChange,
  onSave,
}: ChangeRoleDialogProps) {
  const [selectedRole, setSelectedRole] = useState<string>(currentRole || 'viewer')
  const [isLoading, setIsLoading] = useState(false)

  // Reset when user changes
  useEffect(() => {
    if (currentRole) {
      setSelectedRole(currentRole)
    }
  }, [currentRole])

  if (!user || !currentRole || !organizationName) return null

  const handleSave = async () => {
    if (selectedRole === currentRole) {
      onOpenChange(false)
      return
    }

    setIsLoading(true)
    try {
      await onSave(user.id, selectedRole)
      onOpenChange(false)
    } catch (error) {
      console.error('Error changing role:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const hasChanged = selectedRole !== currentRole

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Change User Role</DialogTitle>
          <DialogDescription>
            Update {user.name}'s role in {organizationName}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* User Information */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">User Information</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-4">
                <Avatar className="h-12 w-12">
                  <AvatarImage src={user.image || undefined} />
                  <AvatarFallback>
                    {user.name.split(' ').map(n => n[0]).join('').toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <div className="font-medium">{user.name}</div>
                  <div className="text-sm text-gray-500">{user.email}</div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Current Role */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Current Role</CardTitle>
              <CardDescription>
                Current role in {organizationName}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Badge 
                variant="secondary" 
                className={getRoleBadgeColor(currentRole)}
              >
                {currentRole.charAt(0).toUpperCase() + currentRole.slice(1)}
              </Badge>
            </CardContent>
          </Card>

          {/* New Role Selection */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">New Role</CardTitle>
              <CardDescription>
                Select the new role for this user
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Select value={selectedRole} onValueChange={setSelectedRole}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {roleOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      <div className="flex flex-col">
                        <div className="font-medium">{option.label}</div>
                        <div className="text-sm text-gray-500">{option.description}</div>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </CardContent>
          </Card>
        </div>

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
            disabled={!hasChanged || isLoading}
          >
            {isLoading ? 'Updating...' : 'Update Role'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}