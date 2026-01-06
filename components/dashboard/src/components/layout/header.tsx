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
import { LogOut, Settings, Building2, Copy } from 'lucide-react'
import { OrganizationSwitcher } from '@/components/organization/organization-switcher'
import { ThemeToggle } from '@/components/theme-toggle'
import { ConnectionStatus } from '@/components/ui/connection-status'
import { useWatchClusters } from '@/hooks/use-watch'
import { useActiveOrganization } from '@/hooks/use-organizations'
import { toast } from 'sonner'
import { useOrganization } from '@/components/organization-provider'

export function Header() {
  const { data: session } = useSession()
  const { organization: activeOrganization } = useActiveOrganization()
  const { getOrgUrl } = useOrganization()

  // Connect to cluster watch for connection status (disabled in dev if no K8s)
  const isDev = process.env.NODE_ENV === 'development'
  const watchStatus = useWatchClusters({ 
    enabled: !isDev, // Disable in development to avoid connection loops
    onEvent: (event) => {
      console.log('Header received watch event:', event.type, event.resource)
    }
  })

  const getInitials = (name: string | null | undefined) => {
    if (!name) return '?'
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }

  const copyNamespace = async () => {
    if (activeOrganization?.namespace) {
      try {
        await navigator.clipboard.writeText(activeOrganization.namespace)
        toast.success('Namespace copied to clipboard')
      } catch (err) {
        toast.error('Failed to copy namespace')
      }
    }
  }

  return (
    <header className="flex h-16 items-center justify-between bg-gradient-to-b from-stone-100 to-stone-200 border-b border-stone-800/80 px-6 dark:from-stone-900 dark:to-stone-950 dark:border-stone-600/80">
      <div className="flex items-center gap-4">
        <OrganizationSwitcher />
      </div>

      <div className="flex items-center gap-4">
        <ConnectionStatus
          isConnected={watchStatus.isConnected}
          lastEvent={watchStatus.lastEvent}
          connectionError={watchStatus.connectionError}
          reconnectCount={watchStatus.reconnectCount}
          onReconnect={watchStatus.reconnect}
        />
        {activeOrganization?.namespace && (
          <div className="flex items-center gap-2 text-sm text-stone-600 dark:text-stone-400">
            <span className="font-mono bg-stone-100 dark:bg-stone-800 px-2 py-1 rounded">
              {activeOrganization.namespace}
            </span>
            <button
              onClick={copyNamespace}
              className="p-1 hover:bg-stone-200 dark:hover:bg-stone-700 rounded transition-colors"
              title="Copy namespace to clipboard"
            >
              <Copy className="h-4 w-4" />
            </button>
          </div>
        )}
        <ThemeToggle />
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
                <p className="text-sm font-light text-stone-900 dark:text-stone-300">{session?.user?.name}</p>
                <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">{session?.user?.email}</p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <a href={getOrgUrl('/settings/organizations')} className="flex items-center cursor-pointer">
                <Building2 className="mr-2 h-4 w-4" />
                <span>Organizations</span>
              </a>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <a href={getOrgUrl('/settings/profile')} className="flex items-center cursor-pointer">
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
      </div>
    </header>
  )
}
