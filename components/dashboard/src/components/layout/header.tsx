'use client'

import { signOut, useSession } from 'next-auth/react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { LogOut, Settings, Building2 } from 'lucide-react'

export function Header() {
  const { data: session } = useSession()

  const getInitials = (name: string | null | undefined) => {
    if (!name) return '?'
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }

  return (
    <header className="flex h-16 items-center justify-between border-b bg-white px-6">
      <div className="flex items-center gap-4">
        {session?.activeOrganization && (
          <div className="flex items-center gap-2">
            <Building2 className="h-5 w-5 text-gray-500" />
            <div>
              <div className="text-sm font-medium">
                {session.activeOrganization.name}
              </div>
              <div className="text-xs text-gray-500">
                {session.activeOrganization.namespace}
              </div>
            </div>
          </div>
        )}
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger className="outline-none">
          <Avatar>
            <AvatarImage src={session?.user?.image || undefined} />
            <AvatarFallback>
              {getInitials(session?.user?.name)}
            </AvatarFallback>
          </Avatar>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuLabel>
            <div className="flex flex-col space-y-1">
              <p className="text-sm font-medium">{session?.user?.name}</p>
              <p className="text-xs text-gray-500">{session?.user?.email}</p>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem asChild>
            <a href="/settings/organizations" className="flex items-center cursor-pointer">
              <Building2 className="mr-2 h-4 w-4" />
              <span>Organizations</span>
            </a>
          </DropdownMenuItem>
          <DropdownMenuItem asChild>
            <a href="/settings/profile" className="flex items-center cursor-pointer">
              <Settings className="mr-2 h-4 w-4" />
              <span>Settings</span>
            </a>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => signOut({ callbackUrl: '/login' })}
            className="cursor-pointer"
          >
            <LogOut className="mr-2 h-4 w-4" />
            <span>Sign out</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  )
}
