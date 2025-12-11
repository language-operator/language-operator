import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useOrganizationStore } from '@/store/organization-store'
import type { OrganizationMember, OrganizationInvite } from '@/store/organization-store'

interface AddMemberData {
  email: string
  role: 'admin' | 'editor' | 'viewer'
}

interface UpdateMemberData {
  role: 'owner' | 'admin' | 'editor' | 'viewer'
}

interface CreateInviteData {
  email: string
  role: 'admin' | 'editor' | 'viewer'
}

export function useOrganizationMembers(organizationId: string) {
  const { updateMembers } = useOrganizationStore()

  return useQuery({
    queryKey: ['organization-members', organizationId],
    queryFn: async () => {
      const response = await fetch(`/api/organizations/${organizationId}/members`)
      if (!response.ok) {
        throw new Error('Failed to fetch organization members')
      }
      const data = await response.json()
      updateMembers(organizationId, data.members)
      return data.members as OrganizationMember[]
    },
    enabled: !!organizationId,
    staleTime: 2 * 60 * 1000, // 2 minutes
  })
}

export function useAddOrganizationMember(organizationId: string) {
  const queryClient = useQueryClient()
  const { addMember } = useOrganizationStore()

  return useMutation({
    mutationFn: async (data: AddMemberData) => {
      const response = await fetch(`/api/organizations/${organizationId}/members`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to add member')
      }

      return response.json()
    },
    onSuccess: (data) => {
      addMember(organizationId, data.member)
      queryClient.invalidateQueries({ queryKey: ['organization-members', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useUpdateOrganizationMember(organizationId: string, memberId: string) {
  const queryClient = useQueryClient()
  const { updateMember } = useOrganizationStore()

  return useMutation({
    mutationFn: async (data: UpdateMemberData) => {
      const response = await fetch(`/api/organizations/${organizationId}/members/${memberId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to update member')
      }

      return response.json()
    },
    onSuccess: (data) => {
      updateMember(organizationId, memberId, data.member)
      queryClient.invalidateQueries({ queryKey: ['organization-members', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useRemoveOrganizationMember(organizationId: string) {
  const queryClient = useQueryClient()
  const { removeMember } = useOrganizationStore()

  return useMutation({
    mutationFn: async (memberId: string) => {
      const response = await fetch(`/api/organizations/${organizationId}/members/${memberId}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to remove member')
      }

      return response.json()
    },
    onSuccess: (_, memberId) => {
      removeMember(organizationId, memberId)
      queryClient.invalidateQueries({ queryKey: ['organization-members', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useOrganizationInvites(organizationId: string) {
  const { updateInvites } = useOrganizationStore()

  return useQuery({
    queryKey: ['organization-invites', organizationId],
    queryFn: async () => {
      const response = await fetch(`/api/organizations/${organizationId}/invites`)
      if (!response.ok) {
        throw new Error('Failed to fetch organization invites')
      }
      const data = await response.json()
      updateInvites(organizationId, data.invites)
      return data.invites as OrganizationInvite[]
    },
    enabled: !!organizationId,
    staleTime: 1 * 60 * 1000, // 1 minute
  })
}

export function useCreateOrganizationInvite(organizationId: string) {
  const queryClient = useQueryClient()
  const { addInvite } = useOrganizationStore()

  return useMutation({
    mutationFn: async (data: CreateInviteData) => {
      const response = await fetch(`/api/organizations/${organizationId}/invites`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to create invite')
      }

      return response.json()
    },
    onSuccess: (data) => {
      addInvite(organizationId, data.invite)
      queryClient.invalidateQueries({ queryKey: ['organization-invites', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useDeleteOrganizationInvite(organizationId: string) {
  const queryClient = useQueryClient()
  const { removeInvite } = useOrganizationStore()

  return useMutation({
    mutationFn: async (inviteId: string) => {
      const response = await fetch(`/api/organizations/${organizationId}/invites/${inviteId}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to delete invite')
      }

      return response.json()
    },
    onSuccess: (_, inviteId) => {
      removeInvite(organizationId, inviteId)
      queryClient.invalidateQueries({ queryKey: ['organization-invites', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useAcceptInvite() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (token: string) => {
      const response = await fetch('/api/invites/accept', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ token })
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to accept invite')
      }

      return response.json()
    },
    onSuccess: () => {
      // Invalidate organizations query to refresh the list
      queryClient.invalidateQueries({ queryKey: ['organizations'] })
    }
  })
}