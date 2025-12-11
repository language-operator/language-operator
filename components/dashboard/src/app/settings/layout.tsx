import { Metadata } from 'next'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'

export const metadata: Metadata = {
  title: 'Settings - Language Operator Dashboard',
  description: 'Manage your account and organization settings',
}

const settingsNavigation = [
  {
    name: 'Organizations',
    href: '/settings/organizations'
  },
  {
    name: 'Profile',
    href: '/settings/profile'
  }
]

interface SettingsLayoutProps {
  children: React.ReactNode
}

export default function SettingsLayout({ children }: SettingsLayoutProps) {
  return (
    <div className="flex flex-1 gap-6">
      <nav className="flex flex-col w-64 gap-1">
        <h2 className="text-lg font-semibold mb-4">Settings</h2>
        {settingsNavigation.map((item) => (
          <Link
            key={item.name}
            href={item.href}
            className={cn(
              'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
            )}
          >
            {item.name}
          </Link>
        ))}
      </nav>
      <div className="flex-1">
        {children}
      </div>
    </div>
  )
}