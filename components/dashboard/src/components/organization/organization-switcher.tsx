'use client'

import { useState } from 'react'
import { Check, ChevronsUpDown, Plus, Building2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { useOrganizations, useActiveOrganization } from '@/hooks/use-organizations'
import { useOrganizationStore } from '@/store/organization-store'
import { useRouter } from 'next/navigation'

interface OrganizationSwitcherProps {
  className?: string
  onCreateNew?: () => void
}

export function OrganizationSwitcher({ className, onCreateNew }: OrganizationSwitcherProps) {
  const [open, setOpen] = useState(false)
  const router = useRouter()
  
  const { data: organizations = [] } = useOrganizations()
  const { organization: activeOrganization, userRole } = useActiveOrganization()
  const { setActiveOrganization } = useOrganizationStore()

  const handleOrganizationSwitch = (organizationId: string) => {
    setActiveOrganization(organizationId)
    setOpen(false)
    // Optionally redirect to dashboard
    router.push('/')
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

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn("w-64 justify-between", className)}
        >
          <div className="flex items-center gap-2 overflow-hidden">
            <Building2 className="h-4 w-4 flex-shrink-0" />
            {activeOrganization ? (
              <div className="flex flex-col items-start overflow-hidden">
                <span className="font-medium truncate">
                  {activeOrganization.name}
                </span>
                <span className="text-xs text-muted-foreground">
                  {activeOrganization.namespace}
                </span>
              </div>
            ) : (
              <span className="text-muted-foreground">Select organization...</span>
            )}
          </div>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-64" align="start">
        <DropdownMenuLabel>Organizations</DropdownMenuLabel>
        <DropdownMenuSeparator />
        
        {organizations.length === 0 ? (
          <DropdownMenuItem disabled>
            No organizations found
          </DropdownMenuItem>
        ) : (
          organizations.map((org) => {
            const userMembership = org.members?.find(member => 
              // This would need to match against current user email/id
              // For now, just finding the first member to show role
              member.role
            )
            
            const isActive = activeOrganization?.id === org.id
            
            return (
              <DropdownMenuItem
                key={org.id}
                className="flex items-center justify-between cursor-pointer"
                onSelect={() => handleOrganizationSwitch(org.id)}
              >
                <div className="flex items-center gap-2 overflow-hidden flex-1">
                  <div className="flex flex-col items-start overflow-hidden flex-1">
                    <div className="flex items-center gap-2 w-full">
                      <span className="font-medium truncate">{org.name}</span>
                      {userMembership && (
                        <Badge 
                          variant="secondary" 
                          className={cn("text-xs px-1.5 py-0", getRoleBadgeColor(userMembership.role))}
                        >
                          {userMembership.role}
                        </Badge>
                      )}
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {org.namespace}
                    </span>
                  </div>
                  {isActive && (
                    <Check className="h-4 w-4 flex-shrink-0" />
                  )}
                </div>
              </DropdownMenuItem>
            )
          })
        )}
        
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="cursor-pointer"
          onSelect={() => {
            setOpen(false)
            router.push('/settings/organizations')
          }}
        >
          <Building2 className="mr-2 h-4 w-4" />
          Manage Organizations
        </DropdownMenuItem>
        {onCreateNew && (
          <DropdownMenuItem
            className="cursor-pointer"
            onSelect={() => {
              setOpen(false)
              onCreateNew()
            }}
          >
            <Plus className="mr-2 h-4 w-4" />
            Create Organization
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}