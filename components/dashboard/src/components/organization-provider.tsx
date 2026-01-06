/**
 * Organization Provider for URL-based organization context
 * 
 * Provides organization data and utilities to child components
 * through React context. Replaces localStorage-based approach.
 */

'use client'

import React, { createContext, useContext, ReactNode } from 'react'

interface Organization {
  id: string
  name: string
  slug?: string
  namespace: string
  plan: string
  createdAt: Date
  updatedAt: Date
}

interface OrganizationContextType {
  organization: Organization
  userRole: string
  orgId: string
  // Helper functions for organization-aware navigation
  getOrgUrl: (path: string) => string
  isOwner: boolean
  isAdmin: boolean
  canManageMembers: boolean
  canManageBilling: boolean
}

const OrganizationContext = createContext<OrganizationContextType | null>(null)

interface OrganizationProviderProps {
  children: ReactNode
  organization: Organization
  userRole: string
  orgId: string
}

export function OrganizationProvider({ 
  children, 
  organization, 
  userRole, 
  orgId 
}: OrganizationProviderProps) {
  // Helper function to create organization-prefixed URLs
  const getOrgUrl = (path: string) => {
    const cleanPath = path.startsWith('/') ? path : `/${path}`
    return `/${orgId}${cleanPath}`
  }

  // Permission helpers based on user role
  const isOwner = userRole === 'owner'
  const isAdmin = userRole === 'admin' || isOwner
  const canManageMembers = isAdmin
  const canManageBilling = isOwner

  const value: OrganizationContextType = {
    organization,
    userRole,
    orgId,
    getOrgUrl,
    isOwner,
    isAdmin,
    canManageMembers,
    canManageBilling
  }

  return (
    <OrganizationContext.Provider value={value}>
      {children}
    </OrganizationContext.Provider>
  )
}

export function useOrganization() {
  const context = useContext(OrganizationContext)
  
  if (!context) {
    throw new Error('useOrganization must be used within an OrganizationProvider')
  }
  
  return context
}

// Hook to get organization ID from URL parameters (for client components)
export function useOrganizationFromUrl() {
  const context = useOrganization()
  return {
    orgId: context.orgId,
    organization: context.organization,
    userRole: context.userRole
  }
}