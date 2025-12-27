'use client'

import { useState } from 'react'
import { Check, ChevronDown, Plus, Building2 } from 'lucide-react'
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
        return 'bg-stone-100 text-stone-800 hover:bg-stone-100 dark:bg-stone-800 dark:text-stone-300'
      default:
        return 'bg-stone-100 text-stone-800 hover:bg-stone-100 dark:bg-stone-800 dark:text-stone-300'
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          role="combobox"
          aria-expanded={open}
          className={cn("min-w-[32rem] max-w-none justify-between hover:bg-transparent dark:hover:bg-transparent", className)}
        >
          <div className="flex items-center gap-2">
            <Building2 className="h-4 w-4 flex-shrink-0" />
            {activeOrganization ? (
              <span className="font-medium text-sm whitespace-nowrap">
                {activeOrganization.name}
              </span>
            ) : (
              <span className="text-stone-600 dark:text-stone-400 text-sm">Select organization...</span>
            )}
            <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </div>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="min-w-[32rem]" align="start">
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
                  <span className="font-medium truncate">{org.name}</span>
                  {userMembership && (
                    <Badge 
                      variant="secondary" 
                      className={cn("text-xs px-1.5 py-0", getRoleBadgeColor(userMembership.role))}
                    >
                      {userMembership.role}
                    </Badge>
                  )}
                  {isActive && (
                    <Check className="h-4 w-4 flex-shrink-0 ml-auto" />
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