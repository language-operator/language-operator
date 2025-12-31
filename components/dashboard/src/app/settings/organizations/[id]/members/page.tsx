'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import { Plus, MoreHorizontal, Mail, UserCheck, UserX, Crown, Shield, Edit, Eye, Trash2, Copy, Link } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { 
  useOrganization,
  useOrganizationMembers, 
  useOrganizationInvites,
  useUpdateMember, 
  useRemoveMember,
  useDeleteInvite,
  useActiveOrganization 
} from '@/hooks/use-organizations'
import { toast } from 'sonner'
import { InviteMemberDialog } from '@/components/organization/invite-member-dialog'
import { config } from '@/lib/config'

interface Member {
  id: string
  role: string
  createdAt: string
  user: {
    id: string
    name: string | null
    email: string
    image: string | null
    createdAt: string
  }
}

interface Invite {
  id: string
  email: string
  role: string
  token: string
  expiresAt: string
  createdAt: string
}

export default function OrganizationMembersPage() {
  const params = useParams()
  const organizationId = params.id as string
  
  const [showInviteDialog, setShowInviteDialog] = useState(false)
  const [memberToRemove, setMemberToRemove] = useState<Member | null>(null)
  const [inviteToDelete, setInviteToDelete] = useState<Invite | null>(null)

  const { data: organizationData } = useOrganization(organizationId)
  const { data: membersData, isLoading: membersLoading } = useOrganizationMembers(organizationId)
  const { data: invitesData, isLoading: invitesLoading } = useOrganizationInvites(organizationId)
  const { userRole } = useActiveOrganization()

  const updateMember = useUpdateMember(organizationId)
  const removeMember = useRemoveMember(organizationId)
  const deleteInvite = useDeleteInvite(organizationId)

  const organization = organizationData?.organization
  const members = membersData?.members || []
  const invites = invitesData?.invites || []

  const canManageMembers = userRole === 'owner' || userRole === 'admin'
  const isOwner = userRole === 'owner'

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'owner': return <Crown className="w-4 h-4" />
      case 'admin': return <Shield className="w-4 h-4" />
      case 'editor': return <Edit className="w-4 h-4" />
      case 'viewer': return <Eye className="w-4 h-4" />
      default: return <Eye className="w-4 h-4" />
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
        return 'bg-gray-100 text-gray-800 hover:bg-gray-100'
      default:
        return 'bg-gray-100 text-gray-800 hover:bg-gray-100'
    }
  }

  const getRoleDescription = (role: string) => {
    switch (role) {
      case 'owner': return 'Full access including billing and organization deletion'
      case 'admin': return 'Manage members and all resources'
      case 'editor': return 'Create and edit resources'
      case 'viewer': return 'View-only access'
      default: return ''
    }
  }

  const handleRoleChange = async (member: Member, newRole: string) => {
    try {
      await updateMember.mutateAsync({ memberId: member.id, role: newRole })
      toast.success(`${member.user.name || member.user.email}'s role updated to ${newRole}`)
    } catch (error: any) {
      toast.error(error.message || 'Failed to update member role')
    }
  }

  const handleRemoveMember = async (member: Member) => {
    try {
      await removeMember.mutateAsync(member.id)
      toast.success(`${member.user.name || member.user.email} has been removed from the organization`)
      setMemberToRemove(null)
    } catch (error: any) {
      toast.error(error.message || 'Failed to remove member')
    }
  }

  const handleDeleteInvite = async (invite: Invite) => {
    try {
      await deleteInvite.mutateAsync(invite.id)
      toast.success(`Invitation for ${invite.email} has been cancelled`)
      setInviteToDelete(null)
    } catch (error: any) {
      toast.error(error.message || 'Failed to delete invitation')
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

  if (membersLoading || invitesLoading) {
    return (
      <div className="space-y-6">
        <div className="space-y-4">
          {[...Array(3)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="p-4">
                <div className="flex items-center space-x-4">
                  <div className="w-10 h-10 bg-gray-200 rounded-full"></div>
                  <div className="flex-1">
                    <div className="h-4 bg-gray-200 rounded w-1/4 mb-2"></div>
                    <div className="h-3 bg-gray-200 rounded w-1/3"></div>
                  </div>
                  <div className="h-6 bg-gray-200 rounded w-16"></div>
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
      {/* Members List */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Team Members ({members.length})</CardTitle>
              <CardDescription>
                People who are currently part of this organization
              </CardDescription>
            </div>
            {canManageMembers && (
              <Button onClick={() => setShowInviteDialog(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Invite Member
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="divide-y">
            {members.map((member: Member) => (
              <div key={member.id} className="p-4 flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <Avatar className="h-10 w-10">
                    <AvatarImage src={member.user.image || ''} alt={member.user.name || ''} />
                    <AvatarFallback>
                      {member.user.name 
                        ? member.user.name.split(' ').map((n: string) => n[0]).join('').toUpperCase()
                        : member.user.email[0].toUpperCase()
                      }
                    </AvatarFallback>
                  </Avatar>
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">
                        {member.user.name || member.user.email}
                      </p>
                      <Badge 
                        variant="secondary" 
                        className={`${getRoleBadgeColor(member.role)} flex items-center gap-1`}
                      >
                        {getRoleIcon(member.role)}
                        {member.role}
                      </Badge>
                    </div>
                    <p className="text-xs text-gray-500">
                      {member.user.email}
                    </p>
                    <p className="text-xs text-gray-400">
                      {getRoleDescription(member.role)}
                    </p>
                  </div>
                </div>

                {canManageMembers && (
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      {/* Role Change Options */}
                      {isOwner && member.role !== 'owner' && (
                        <DropdownMenuItem onClick={() => handleRoleChange(member, 'owner')}>
                          <Crown className="mr-2 h-4 w-4" />
                          Make Owner
                        </DropdownMenuItem>
                      )}
                      {member.role !== 'admin' && (
                        <DropdownMenuItem onClick={() => handleRoleChange(member, 'admin')}>
                          <Shield className="mr-2 h-4 w-4" />
                          Make Admin
                        </DropdownMenuItem>
                      )}
                      {member.role !== 'editor' && (
                        <DropdownMenuItem onClick={() => handleRoleChange(member, 'editor')}>
                          <Edit className="mr-2 h-4 w-4" />
                          Make Editor
                        </DropdownMenuItem>
                      )}
                      {member.role !== 'viewer' && (
                        <DropdownMenuItem onClick={() => handleRoleChange(member, 'viewer')}>
                          <Eye className="mr-2 h-4 w-4" />
                          Make Viewer
                        </DropdownMenuItem>
                      )}
                      
                      {/* Remove Member */}
                      {(isOwner || (userRole === 'admin' && member.role !== 'owner')) && (
                        <>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem 
                            className="text-red-600"
                            onClick={() => setMemberToRemove(member)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Remove from organization
                          </DropdownMenuItem>
                        </>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                )}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Pending Invitations */}
      {invites.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Pending Invitations ({invites.length})</CardTitle>
            <CardDescription>
              Invitations that have been sent but not yet accepted
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y">
              {invites.map((invite: Invite) => (
                <div key={invite.id} className="p-4 flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <div className="h-10 w-10 bg-gray-100 rounded-full flex items-center justify-center">
                      <Mail className="h-5 w-5 text-gray-400" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium">{invite.email}</p>
                        <Badge 
                          variant="secondary" 
                          className={`${getRoleBadgeColor(invite.role)} flex items-center gap-1`}
                        >
                          {getRoleIcon(invite.role)}
                          {invite.role}
                        </Badge>
                        <Badge variant="outline" className="text-yellow-600 bg-yellow-50">
                          Pending
                        </Badge>
                      </div>
                      <p className="text-xs text-gray-500">
                        Invited {new Date(invite.createdAt).toLocaleDateString()}
                      </p>
                      <p className="text-xs text-gray-400">
                        Expires {new Date(invite.expiresAt).toLocaleDateString()}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    {canManageMembers && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleCopyInviteLink(invite)}
                        className="h-8 px-3"
                      >
                        <Copy className="mr-1 h-3 w-3" />
                        Copy Link
                      </Button>
                    )}
                    {canManageMembers && (
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
                            <Link className="mr-2 h-4 w-4" />
                            Copy invitation link
                          </DropdownMenuItem>
                          <DropdownMenuItem 
                            className="text-red-600"
                            onClick={() => setInviteToDelete(invite)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Cancel invitation
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Empty State */}
      {members.length === 0 && invites.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center p-12 text-center">
            <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mb-4">
              <UserCheck className="w-6 h-6 text-gray-400" />
            </div>
            <h3 className="text-lg font-semibold mb-2">No team members</h3>
            <p className="text-gray-600 mb-4">
              Get started by inviting your first team member
            </p>
            {canManageMembers && (
              <Button onClick={() => setShowInviteDialog(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Invite Member
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Invite Member Dialog */}
      <InviteMemberDialog
        open={showInviteDialog}
        onOpenChange={setShowInviteDialog}
        organizationId={organizationId}
      />

      {/* Remove Member Confirmation */}
      <AlertDialog open={!!memberToRemove} onOpenChange={() => setMemberToRemove(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove team member</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove <strong>{memberToRemove?.user.name || memberToRemove?.user.email}</strong> from this organization?
              This will immediately revoke their access to all resources.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => memberToRemove && handleRemoveMember(memberToRemove)}
              className="bg-red-600 hover:bg-red-700"
            >
              Remove member
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete Invite Confirmation */}
      <AlertDialog open={!!inviteToDelete} onOpenChange={() => setInviteToDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cancel invitation</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to cancel the invitation for <strong>{inviteToDelete?.email}</strong>?
              They will no longer be able to join the organization using this invitation.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep invitation</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => inviteToDelete && handleDeleteInvite(inviteToDelete)}
              className="bg-red-600 hover:bg-red-700"
            >
              Cancel invitation
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}