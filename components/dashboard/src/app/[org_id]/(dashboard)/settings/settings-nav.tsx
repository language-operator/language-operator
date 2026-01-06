'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import { Users, Building2, UserCircle } from 'lucide-react'
import { useOrganization } from '@/components/organization-provider'

export function SettingsNav() {
  const pathname = usePathname()
  const { getOrgUrl } = useOrganization()

  const settingsNavigation = [
    {
      name: 'Users',
      href: getOrgUrl('/settings/users'),
      icon: Users
    },
    {
      name: 'Orgs',
      href: getOrgUrl('/settings/organizations'),
      icon: Building2
    },
    {
      name: 'Profile',
      href: getOrgUrl('/settings/profile'),
      icon: UserCircle
    }
  ]

  return (
    <nav className="flex flex-col w-24 gap-2 pr-4">
      {settingsNavigation.map((item) => {
        const isActive = pathname === item.href || pathname.startsWith(item.href + '/')
        const Icon = item.icon
        return (
          <Link
            key={item.name}
            href={item.href}
            className={cn(
              'flex flex-col items-center gap-1 rounded-lg px-2 py-3 text-xs font-medium transition-colors',
              isActive
                ? 'text-amber-600 dark:text-amber-400'
                : 'text-stone-600 hover:bg-stone-100 hover:text-stone-900 dark:text-stone-400 dark:hover:bg-stone-800 dark:hover:text-stone-300'
            )}
          >
            <Icon className="w-6 h-6" />
            {item.name}
          </Link>
        )
      })}
    </nav>
  )
}