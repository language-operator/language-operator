import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from 'next-auth/react'
import { useOrganizationStore } from '@/store/organization-store'
import type { Organization } from '@/store/organization-store'

interface CreateOrganizationData {
  name: string
  slug: string
  namespace: string
  plan: 'free' | 'pro' | 'enterprise'
}

interface UpdateOrganizationData {
  name?: string
  slug?: string
  namespace?: string
  plan?: 'free' | 'pro' | 'enterprise'
}

export function useOrganizations() {
  const { data: session } = useSession()
  const queryClient = useQueryClient()
  const { setOrganizations, setLoading, setError, initializeActiveOrganization } = useOrganizationStore()

  return useQuery({
    queryKey: ['organizations'],
    queryFn: async () => {
      setLoading(true)
      try {
        const response = await fetch('/api/organizations')
        if (!response.ok) {
          throw new Error('Failed to fetch organizations')
        }
        const data = await response.json()
        setOrganizations(data.organizations)
        
        // Initialize active organization after setting organizations
        initializeActiveOrganization()
        
        setError(null)
        return data.organizations as Organization[]
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error'
        setError(errorMessage)
        throw error
      } finally {
        setLoading(false)
      }
    },
    enabled: !!session?.user,
    staleTime: 5 * 60 * 1000, // 5 minutes
    refetchOnWindowFocus: false
  })
}

export function useCreateOrganization() {
  const queryClient = useQueryClient()
  const { addOrganization } = useOrganizationStore()

  return useMutation({
    mutationFn: async (data: CreateOrganizationData) => {
      const response = await fetch('/api/organizations', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      })

      if (!response.ok) {
        const errorData = await response.json()
        
        // Handle specific error cases
        if (response.status === 409) {
          throw new Error('Organization with this slug or namespace already exists')
        } else if (response.status === 400) {
          if (errorData.details) {
            // Show validation errors
            const validationErrors = errorData.details.map((detail: any) => detail.message).join(', ')
            throw new Error(`Invalid input: ${validationErrors}`)
          } else {
            throw new Error(errorData.error || 'Invalid input')
          }
        } else {
          throw new Error(errorData.error || 'Failed to create organization')
        }
      }

      return response.json()
    },
    onSuccess: (data) => {
      addOrganization(data.organization)
      queryClient.invalidateQueries({ queryKey: ['organizations'] })
    }
  })
}

export function useUpdateOrganization(organizationId: string) {
  const queryClient = useQueryClient()
  const { updateOrganization } = useOrganizationStore()

  return useMutation({
    mutationFn: async (data: UpdateOrganizationData) => {
      const response = await fetch(`/api/organizations/${organizationId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      })

      if (!response.ok) {
        const errorData = await response.json()
        
        // Handle specific error cases
        if (response.status === 409) {
          throw new Error('Organization with this slug or namespace already exists')
        } else if (response.status === 400) {
          if (errorData.details) {
            // Show validation errors
            const validationErrors = errorData.details.map((detail: any) => detail.message).join(', ')
            throw new Error(`Invalid input: ${validationErrors}`)
          } else {
            throw new Error(errorData.error || 'Invalid input')
          }
        } else if (response.status === 403) {
          throw new Error('Forbidden: You do not have permission to edit this organization')
        } else {
          throw new Error(errorData.error || 'Failed to update organization')
        }
      }

      return response.json()
    },
    onSuccess: (data) => {
      updateOrganization(organizationId, data.organization)
      queryClient.invalidateQueries({ queryKey: ['organizations'] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useDeleteOrganization() {
  const queryClient = useQueryClient()
  const { removeOrganization } = useOrganizationStore()

  return useMutation({
    mutationFn: async (organizationId: string) => {
      const response = await fetch(`/api/organizations/${organizationId}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to delete organization')
      }

      return response.json()
    },
    onSuccess: (_, organizationId) => {
      removeOrganization(organizationId)
      queryClient.invalidateQueries({ queryKey: ['organizations'] })
    }
  })
}

export function useOrganization(organizationId: string) {
  return useQuery({
    queryKey: ['organization', organizationId],
    queryFn: async () => {
      const response = await fetch(`/api/organizations/${organizationId}`)
      if (!response.ok) {
        throw new Error('Failed to fetch organization')
      }
      return response.json()
    },
    enabled: !!organizationId,
    staleTime: 2 * 60 * 1000, // 2 minutes
  })
}

export function useActiveOrganization() {
  const { activeOrganization, activeOrganizationId } = useOrganizationStore()
  
  const { data: organizationData } = useOrganization(activeOrganizationId || '')
  
  return {
    organization: organizationData?.organization || activeOrganization,
    userRole: organizationData?.userRole,
    isLoading: !activeOrganizationId ? false : !organizationData && !activeOrganization
  }
}

// Organization members hooks
export function useOrganizationMembers(organizationId: string) {
  return useQuery({
    queryKey: ['organization-members', organizationId],
    queryFn: async () => {
      const response = await fetch(`/api/organizations/${organizationId}/members`)
      if (!response.ok) {
        throw new Error('Failed to fetch organization members')
      }
      return response.json()
    },
    enabled: !!organizationId,
    staleTime: 2 * 60 * 1000, // 2 minutes
  })
}

export function useUpdateMember(organizationId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ memberId, role }: { memberId: string; role: string }) => {
      const response = await fetch(`/api/organizations/${organizationId}/members/${memberId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ role })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update member')
      }

      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-members', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useRemoveMember(organizationId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (memberId: string) => {
      const response = await fetch(`/api/organizations/${organizationId}/members/${memberId}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to remove member')
      }

      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-members', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

// Organization invitations hooks
export function useOrganizationInvites(organizationId: string) {
  return useQuery({
    queryKey: ['organization-invites', organizationId],
    queryFn: async () => {
      const response = await fetch(`/api/organizations/${organizationId}/invites`)
      if (!response.ok) {
        throw new Error('Failed to fetch organization invites')
      }
      return response.json()
    },
    enabled: !!organizationId,
    staleTime: 1 * 60 * 1000, // 1 minute
  })
}

export function useCreateInvite(organizationId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ email, role }: { email: string; role: string }) => {
      const response = await fetch(`/api/organizations/${organizationId}/invites`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ email, role })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to send invitation')
      }

      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-invites', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}

export function useDeleteInvite(organizationId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (inviteId: string) => {
      const response = await fetch(`/api/organizations/${organizationId}/invites/${inviteId}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to delete invitation')
      }

      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-invites', organizationId] })
      queryClient.invalidateQueries({ queryKey: ['organization', organizationId] })
    }
  })
}