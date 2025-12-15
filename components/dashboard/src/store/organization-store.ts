import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface Organization {
  id: string
  name: string
  slug: string
  namespace: string
  plan: string
  createdAt: string
  updatedAt: string
  members?: OrganizationMember[]
  invites?: OrganizationInvite[]
  _count?: {
    members: number
    invites: number
  }
}

export interface OrganizationMember {
  id: string
  organizationId: string
  userId: string
  role: 'owner' | 'admin' | 'editor' | 'viewer'
  createdAt: string
  user: {
    id: string
    name: string | null
    email: string
    image: string | null
    createdAt?: string
  }
}

export interface OrganizationInvite {
  id: string
  organizationId: string
  email: string
  role: 'admin' | 'editor' | 'viewer'
  token: string
  expiresAt: string
  createdAt: string
}

interface OrganizationStore {
  // State
  organizations: Organization[]
  activeOrganizationId: string | null
  isLoading: boolean
  error: string | null

  // Computed
  activeOrganization: Organization | null
  userRole: string | null

  // Actions
  setOrganizations: (organizations: Organization[]) => void
  setActiveOrganization: (organizationId: string | null) => void
  addOrganization: (organization: Organization) => void
  updateOrganization: (organizationId: string, updates: Partial<Organization>) => void
  removeOrganization: (organizationId: string) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  reset: () => void
  initializeActiveOrganization: () => void

  // Member management
  updateMembers: (organizationId: string, members: OrganizationMember[]) => void
  addMember: (organizationId: string, member: OrganizationMember) => void
  updateMember: (organizationId: string, memberId: string, updates: Partial<OrganizationMember>) => void
  removeMember: (organizationId: string, memberId: string) => void

  // Invite management
  updateInvites: (organizationId: string, invites: OrganizationInvite[]) => void
  addInvite: (organizationId: string, invite: OrganizationInvite) => void
  removeInvite: (organizationId: string, inviteId: string) => void
}

export const useOrganizationStore = create<OrganizationStore>()(
  persist(
    (set, get) => ({
      // Initial state
      organizations: [],
      activeOrganizationId: null,
      isLoading: false,
      error: null,

      // Computed properties
      get activeOrganization() {
        const { organizations, activeOrganizationId } = get()
        return organizations.find(org => org.id === activeOrganizationId) || null
      },

      get userRole() {
        const { activeOrganization } = get()
        // This would need to be populated based on current user session
        // For now, returning null - will be set when we fetch organization data
        return activeOrganization?.members?.find(member => 
          // We'll need to match against current user ID
          member.role
        )?.role || null
      },

      // Actions
      setOrganizations: (organizations) => set({ organizations }),

      setActiveOrganization: (organizationId) => {
        set({ activeOrganizationId: organizationId })
      },

      addOrganization: (organization) => set((state) => ({
        organizations: [...state.organizations, organization]
      })),

      updateOrganization: (organizationId, updates) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId ? { ...org, ...updates } : org
        )
      })),

      removeOrganization: (organizationId) => set((state) => ({
        organizations: state.organizations.filter(org => org.id !== organizationId),
        activeOrganizationId: state.activeOrganizationId === organizationId 
          ? null 
          : state.activeOrganizationId
      })),

      setLoading: (loading) => set({ isLoading: loading }),

      setError: (error) => set({ error }),

      reset: () => set({
        organizations: [],
        activeOrganizationId: null,
        isLoading: false,
        error: null
      }),

      // Member management
      updateMembers: (organizationId, members) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId ? { ...org, members } : org
        )
      })),

      addMember: (organizationId, member) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId 
            ? { ...org, members: [...(org.members || []), member] }
            : org
        )
      })),

      updateMember: (organizationId, memberId, updates) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId 
            ? {
                ...org, 
                members: (org.members || []).map(member =>
                  member.id === memberId ? { ...member, ...updates } : member
                )
              }
            : org
        )
      })),

      removeMember: (organizationId, memberId) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId 
            ? {
                ...org, 
                members: (org.members || []).filter(member => member.id !== memberId)
              }
            : org
        )
      })),

      // Invite management
      updateInvites: (organizationId, invites) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId ? { ...org, invites } : org
        )
      })),

      addInvite: (organizationId, invite) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId 
            ? { ...org, invites: [...(org.invites || []), invite] }
            : org
        )
      })),

      removeInvite: (organizationId, inviteId) => set((state) => ({
        organizations: state.organizations.map(org =>
          org.id === organizationId 
            ? {
                ...org, 
                invites: (org.invites || []).filter(invite => invite.id !== inviteId)
              }
            : org
        )
      })),

      // Initialize active organization from persisted data on app startup
      initializeActiveOrganization: () => {
        const { activeOrganizationId, organizations } = get()
        
        // If we have a stored active organization ID but no active organization in memory,
        // or if the active organization doesn't exist in the current organizations list,
        // try to set it from the organizations list
        if (activeOrganizationId && organizations.length > 0) {
          const foundOrg = organizations.find(org => org.id === activeOrganizationId)
          if (foundOrg) {
            // Organization found in list, we're good
            return
          }
        }
        
        // If no active organization is set, or the stored one doesn't exist anymore,
        // default to the first available organization
        if (organizations.length > 0 && !activeOrganizationId) {
          set({ activeOrganizationId: organizations[0].id })
        }
      }
    }),
    {
      name: 'organization-store',
      partialize: (state) => ({
        activeOrganizationId: state.activeOrganizationId,
        organizations: state.organizations
      })
    }
  )
)