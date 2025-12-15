'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'

const settingsNavigation = [
  {
    name: 'Users',
    href: '/settings/users'
  },
  {
    name: 'Organizations',
    href: '/settings/organizations'
  },
  {
    name: 'Profile',
    href: '/settings/profile'
  }
]

export function SettingsNav() {
  const pathname = usePathname()

  return (
    <nav className="flex flex-col w-64 gap-1">
      <h2 className="text-lg font-semibold mb-4">Settings</h2>
      {settingsNavigation.map((item) => {
        const isActive = pathname === item.href
        return (
          <Link
            key={item.name}
            href={item.href}
            className={cn(
              'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              isActive
                ? 'bg-gray-200 text-gray-900'
                : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
            )}
          >
            {item.name}
          </Link>
        )
      })}
    </nav>
  )
}