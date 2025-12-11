'use client'

import { useState } from 'react'
import { Plus, Settings, Users, Mail, MoreHorizontal, Trash2, Edit } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useOrganizations, useActiveOrganization } from '@/hooks/use-organizations'
import { useOrganizationStore } from '@/store/organization-store'
import { OrganizationSwitcher } from '@/components/organization/organization-switcher'

export default function OrganizationsPage() {
  const [showCreateForm, setShowCreateForm] = useState(false)
  
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
        return 'bg-gray-100 text-gray-800 hover:bg-gray-100'
      default:
        return 'bg-gray-100 text-gray-800 hover:bg-gray-100'
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Organizations</h1>
            <p className="text-gray-600">Manage your organizations and switch between them</p>
          </div>
        </div>
        <div className="grid gap-4">
          {[...Array(3)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="p-6">
                <div className="h-4 bg-gray-200 rounded w-1/4 mb-2"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2 mb-4"></div>
                <div className="flex gap-2">
                  <div className="h-6 bg-gray-200 rounded w-16"></div>
                  <div className="h-6 bg-gray-200 rounded w-20"></div>
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
          <p className="text-gray-600">Manage your organizations and switch between them</p>
        </div>
        <Button onClick={() => setShowCreateForm(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Organization
        </Button>
      </div>

      {/* Active Organization Switcher */}
      <Card>
        <CardHeader>
          <CardTitle>Current Organization</CardTitle>
          <CardDescription>
            Switch between organizations to manage different projects and teams
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <OrganizationSwitcher 
              className="w-96"
              onCreateNew={() => setShowCreateForm(true)}
            />
            {activeOrganization && (
              <div className="text-sm text-gray-600">
                All Kubernetes resources will be scoped to the <strong>{activeOrganization.namespace}</strong> namespace
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Organizations List */}
      <div className="grid gap-4">
        {organizations.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center p-12 text-center">
              <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mb-4">
                <Settings className="w-6 h-6 text-gray-400" />
              </div>
              <h3 className="text-lg font-semibold mb-2">No organizations</h3>
              <p className="text-gray-600 mb-4">
                Create your first organization to start managing Language Operator resources
              </p>
              <Button onClick={() => setShowCreateForm(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create Organization
              </Button>
            </CardContent>
          </Card>
        ) : (
          organizations.map((org) => {
            const userMembership = org.members?.find(member => 
              // This would need to match against current user
              member.role
            )
            
            const isActive = activeOrganization?.id === org.id
            
            return (
              <Card key={org.id} className={isActive ? 'ring-2 ring-blue-500' : ''}>
                <CardContent className="p-6">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="text-lg font-semibold">{org.name}</h3>
                        {isActive && (
                          <Badge variant="default" className="bg-blue-100 text-blue-800">
                            Active
                          </Badge>
                        )}
                        {userMembership && (
                          <Badge 
                            variant="secondary" 
                            className={getRoleBadgeColor(userMembership.role)}
                          >
                            {userMembership.role}
                          </Badge>
                        )}
                        <Badge variant="outline" className="capitalize">
                          {org.plan}
                        </Badge>
                      </div>
                      
                      <div className="space-y-1 text-sm text-gray-600">
                        <div>
                          <span className="font-medium">Namespace:</span> {org.namespace}
                        </div>
                        <div>
                          <span className="font-medium">Slug:</span> {org.slug}
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                        <div className="flex items-center gap-1">
                          <Users className="w-4 h-4" />
                          {org._count?.members || 0} members
                        </div>
                        {(org._count?.invites || 0) > 0 && (
                          <div className="flex items-center gap-1">
                            <Mail className="w-4 h-4" />
                            {org._count?.invites} pending invites
                          </div>
                        )}
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-2">
                      {!isActive && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setActiveOrganization(org.id)}
                        >
                          Switch to
                        </Button>
                      )}
                      
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem>
                            <Edit className="mr-2 h-4 w-4" />
                            Edit Organization
                          </DropdownMenuItem>
                          <DropdownMenuItem>
                            <Users className="mr-2 h-4 w-4" />
                            Manage Members
                          </DropdownMenuItem>
                          <DropdownMenuItem>
                            <Mail className="mr-2 h-4 w-4" />
                            Manage Invites
                          </DropdownMenuItem>
                          {userMembership?.role === 'owner' && (
                            <>
                              <DropdownMenuItem className="text-red-600">
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete Organization
                              </DropdownMenuItem>
                            </>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })
        )}
      </div>
      
      {showCreateForm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold mb-4">Create Organization</h2>
            <p className="text-gray-600 mb-4">
              Organization creation form will be implemented in the next step.
            </p>
            <div className="flex gap-2 justify-end">
              <Button variant="outline" onClick={() => setShowCreateForm(false)}>
                Cancel
              </Button>
              <Button onClick={() => setShowCreateForm(false)}>
                Create
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}