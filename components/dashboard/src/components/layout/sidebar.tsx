'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import {
  Home,
  Bot,
  Cpu,
  Wrench,
  Users,
  Cloud,
  Settings,
} from 'lucide-react'

const navigation = [
  { name: 'Overview', href: '/', icon: Home },
  { name: 'Agents', href: '/agents', icon: Bot },
  { name: 'Models', href: '/models', icon: Cpu },
  { name: 'Tools', href: '/tools', icon: Wrench },
  { name: 'Personas', href: '/personas', icon: Users },
  { name: 'Clusters', href: '/clusters', icon: Cloud },
]

export function Sidebar() {
  const pathname = usePathname()

  return (
    <div className="flex h-screen w-64 flex-col border-r bg-gray-50">
      <div className="flex h-16 items-center border-b px-6">
        <h1 className="text-xl font-bold">Language Operator</h1>
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {navigation.map((item) => {
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
              <item.icon className="h-5 w-5" />
              {item.name}
            </Link>
          )
        })}
      </nav>
      <div className="border-t p-3">
        <Link
          href="/settings"
          className={cn(
            'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
            pathname.startsWith('/settings')
              ? 'bg-gray-200 text-gray-900'
              : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
          )}
        >
          <Settings className="h-5 w-5" />
          Settings
        </Link>
      </div>
    </div>
  )
}
